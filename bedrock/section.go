// Package bedrock reads and edits native Bedrock world records.
package bedrock

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"

	"celler/layer_storage"
	"neomega-blocks/ir"
	encoder_go "neomega-nbt/encoder-go-object"
	nbt "neomega-nbt/presets/v1"
)

// Block retains both a tidy state and the version of that palette entry.
// Versions belong to entries: a section may contain states from different versions.
type Block struct {
	State   ir.IRBlock
	Version int32
}

// Layer contains one physical block layer. Original palette documents retain
// additional fields and root names for entries whose Block value is unchanged.
type Layer struct {
	Blocks   *layer_storage.Palette[Block]
	entries  []nbt.Document
	original []Block
}

func NewLayer(defaultBlock Block) *Layer {
	return &Layer{Blocks: layer_storage.MakePaletteStorage(defaultBlock)}
}

type Section struct {
	Y       int32
	Version byte
	Layers  []*Layer
}

// DecodeSection supports persistent subchunk versions 1, 8 and 9.
// keyY is taken from the database key; version 9 also stores Y in its payload.
func DecodeSection(data []byte, keyY int32) (*Section, error) {
	if keyY < -128 || keyY > 127 {
		return nil, fmt.Errorf("section Y outside Byte range")
	}
	r := bytes.NewReader(data)
	version, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	s := &Section{Y: keyY, Version: version}
	count := byte(1)
	switch version {
	case 1:
	case 8, 9:
		count, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		if version == 9 {
			y, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if int32(int8(y)) != keyY {
				return nil, fmt.Errorf("subchunk Y does not match key")
			}
		}
	default:
		return nil, fmt.Errorf("unsupported subchunk version %d", version)
	}
	for i := 0; i < int(count); i++ {
		layer, err := decodeLayer(r)
		if err != nil {
			return nil, fmt.Errorf("layer %d: %w", i, err)
		}
		s.Layers = append(s.Layers, layer)
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("%d trailing subchunk bytes", r.Len())
	}
	return s, nil
}

func readWords(r *bytes.Reader, width int) ([]uint32, error) {
	if width == 0 {
		return nil, nil
	}
	switch width {
	case 1, 2, 3, 4, 5, 6, 8, 16:
	default:
		return nil, fmt.Errorf("unsupported palette width %d", width)
	}
	perWord := 32 / width
	words := make([]uint32, (4096+perWord-1)/perWord)
	if err := binary.Read(r, binary.LittleEndian, words); err != nil {
		return nil, err
	}
	return words, nil
}
func decodeLayer(r *bytes.Reader) (*Layer, error) {
	header, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if header&1 != 0 {
		return nil, fmt.Errorf("runtime block palette in persistent subchunk")
	}
	width := int(header >> 1)
	words, err := readWords(r, width)
	if err != nil {
		return nil, err
	}
	count := uint32(1)
	if width != 0 {
		if err = binary.Read(r, binary.LittleEndian, &count); err != nil {
			return nil, err
		}
	}
	if count == 0 || count > 1<<width {
		return nil, fmt.Errorf("invalid palette size %d", count)
	}
	l := &Layer{entries: make([]nbt.Document, count), original: make([]Block, count)}
	for i := range l.entries {
		doc, err := nbt.ReadDocument(r, nbt.EncodingLittleEndian)
		if err != nil {
			return nil, err
		}
		name, ok := doc.Value["name"].(string)
		if !ok {
			return nil, fmt.Errorf("palette requires name")
		}
		states, ok := doc.Value["states"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unsupported palette entry without states")
		}
		block, err := ir.NewIRBlockFromGo(name, states)
		if err != nil {
			return nil, err
		}
		version, ok := doc.Value["version"].(int32)
		if !ok {
			return nil, fmt.Errorf("palette requires Int version")
		}
		l.entries[i] = doc
		l.original[i] = Block{State: block, Version: version}
	}
	l.Blocks = layer_storage.MakePaletteStorage(l.original[0])
	l.Blocks.Fill(width, l.original[0])
	l.Blocks.SetPalette(l.original)
	l.Blocks.Data = words
	for i := 0; i < 4096; i++ {
		if l.Blocks.GetIndex(layer_storage.At(i>>8, i&15, i>>4&15)) >= int(count) {
			return nil, fmt.Errorf("palette index out of range at %d", i)
		}
	}
	return l, nil
}

func (s *Section) Encode() ([]byte, error) {
	if s.Y < -128 || s.Y > 127 {
		return nil, fmt.Errorf("section Y outside Byte range")
	}
	if len(s.Layers) > 255 {
		return nil, fmt.Errorf("too many block layers")
	}
	var out bytes.Buffer
	out.WriteByte(s.Version)
	switch s.Version {
	case 1:
		if len(s.Layers) != 1 {
			return nil, fmt.Errorf("version 1 requires one layer")
		}
	case 8, 9:
		out.WriteByte(byte(len(s.Layers)))
		if s.Version == 9 {
			out.WriteByte(byte(int8(s.Y)))
		}
	default:
		return nil, fmt.Errorf("unsupported subchunk version %d", s.Version)
	}
	for i, layer := range s.Layers {
		if err := layer.encode(&out); err != nil {
			return nil, fmt.Errorf("layer %d: %w", i, err)
		}
	}
	return out.Bytes(), nil
}
func diskWidth(count int) int {
	width := bits.Len(uint(count - 1))
	if width == 7 {
		return 8
	}
	if width > 8 {
		return 16
	}
	return width
}
func (l *Layer) encode(out *bytes.Buffer) error {
	if l == nil || l.Blocks == nil {
		return fmt.Errorf("missing layer storage")
	}
	palette := l.Blocks.GetUsedPalette()
	if len(palette) == 0 || len(palette) > 65536 {
		return fmt.Errorf("invalid palette size")
	}
	// A one-bit palette is accepted by all supported subchunk versions.
	width := max(1, diskWidth(len(palette)))
	out.WriteByte(byte(width << 1))
	words, err := packWords(l.Blocks, width)
	if err != nil {
		return err
	}
	if err = binary.Write(out, binary.LittleEndian, words); err != nil {
		return err
	}
	binary.Write(out, binary.LittleEndian, uint32(len(palette)))
	for i, block := range palette {
		if !block.State.IsTidy() {
			return fmt.Errorf("palette entry %d must be tidy IR", i)
		}
		var doc nbt.Document
		if i < len(l.original) && l.original[i] == block {
			doc = l.entries[i]
		} else {
			var value nbt.Compound
			if err := block.State.DecodeToHL(encoder_go.NewGoObjEncoder(&value, encoder_go.EncodeOption{})); err != nil {
				return err
			}
			value["version"] = block.Version
			doc = nbt.Document{Value: value}
		}
		raw, err := nbt.EncodeDocument(doc, nbt.EncodingLittleEndian)
		if err != nil {
			return err
		}
		out.Write(raw)
	}
	return nil
}

func packWords[T comparable](p *layer_storage.Palette[T], width int) ([]uint32, error) {
	if width == 0 {
		return nil, nil
	}
	perWord := 32 / width
	words := make([]uint32, (4096+perWord-1)/perWord)
	for i := 0; i < 4096; i++ {
		index := p.GetIndex(layer_storage.At(i>>8, i&15, i>>4&15))
		if index > int(p.LastPaletteIdx) || index >= 1<<width {
			return nil, fmt.Errorf("palette index out of range at %d", i)
		}
		words[i/perWord] |= uint32(index) << ((i % perWord) * width)
	}
	return words, nil
}
