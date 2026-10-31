package bedrock

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"sort"

	"celler"
	nbt "neomega-nbt/presets/v1"
)

const (
	TagData3D         byte = 0x2b
	TagVersion        byte = 0x2c
	TagData2D         byte = 0x2d
	TagSubChunk       byte = 0x2f
	TagBlockEntities  byte = 0x31
	TagEntities       byte = 0x32
	TagFinalizedState byte = 0x36
)

// Chunk holds owned record bytes, indexed by the suffix after coordinates and
// dimension. Decode individual records only when their contents are needed.
type Chunk struct {
	Pos       celler.ChunkPos
	Dimension int32
	Records   map[string][]byte
}

func NewChunk(pos celler.ChunkPos, dimension int32) *Chunk {
	return &Chunk{Pos: pos, Dimension: dimension, Records: map[string][]byte{}}
}
func sectionKey(y int32) string { return string([]byte{TagSubChunk, byte(int8(y))}) }
func (c *Chunk) Section(y int32) (*Section, error) {
	if y < -128 || y > 127 {
		return nil, fmt.Errorf("section Y outside Byte range")
	}
	data, ok := c.Records[sectionKey(y)]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return DecodeSection(data, y)
}
func (c *Chunk) SectionYs() []int32 {
	var ys []int32
	for key := range c.Records {
		if len(key) == 2 && key[0] == TagSubChunk {
			ys = append(ys, int32(int8(key[1])))
		}
	}
	sort.Slice(ys, func(i, j int) bool { return ys[i] < ys[j] })
	return ys
}
func (c *Chunk) SetSection(section *Section) error {
	data, err := section.Encode()
	if err != nil {
		return err
	}
	if c.Records == nil {
		c.Records = map[string][]byte{}
	}
	c.Records[sectionKey(section.Y)] = data
	return nil
}
func (c *Chunk) DeleteSection(y int32) error {
	if y < -128 || y > 127 {
		return fmt.Errorf("section Y outside Byte range")
	}
	delete(c.Records, sectionKey(y))
	return nil
}
func (c *Chunk) BlockEntities() ([]nbt.Document, error) {
	return nbt.DecodeDocuments(c.Records[string([]byte{TagBlockEntities})], nbt.EncodingLittleEndian)
}
func (c *Chunk) SetBlockEntities(documents []nbt.Document) error {
	data, err := nbt.EncodeDocuments(documents, nbt.EncodingLittleEndian)
	if err != nil {
		return err
	}
	if c.Records == nil {
		c.Records = map[string][]byte{}
	}
	c.Records[string([]byte{TagBlockEntities})] = data
	return nil
}

// ChunkPrefix encodes native little-endian coordinates and an optional dimension.
func ChunkPrefix(pos celler.ChunkPos, dimension int32) []byte {
	key := binary.LittleEndian.AppendUint32(nil, uint32(pos.X))
	key = binary.LittleEndian.AppendUint32(key, uint32(pos.Z))
	if dimension != 0 {
		key = binary.LittleEndian.AppendUint32(key, uint32(dimension))
	}
	return key
}

// ParseChunkKey recognizes native chunk tags, including unknown tags in the
// reserved chunk tag range. Non-chunk database records remain accessible via DB.
func ParseChunkKey(key []byte) (pos celler.ChunkPos, dimension int32, suffix string, ok bool) {
	offset := 8
	switch len(key) {
	case 9, 10:
	case 13, 14:
		dimension = int32(binary.LittleEndian.Uint32(key[8:]))
		if dimension == 0 {
			return
		}
		offset = 12
	default:
		return
	}
	tag := key[offset]
	if !(tag >= 0x2b && tag <= 0x41 || tag == 0x76) {
		return
	}
	if len(key) == offset+2 && tag != TagSubChunk {
		return
	}
	if len(key) == offset+1 && tag == TagSubChunk {
		return
	}
	pos = celler.ChunkPos{X: int32(binary.LittleEndian.Uint32(key)), Z: int32(binary.LittleEndian.Uint32(key[4:]))}
	return pos, dimension, string(key[offset:]), true
}
