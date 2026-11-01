// Package java reads and edits native Java world records.
package java

import (
	"fmt"
	"io/fs"
	"maps"
	"math/bits"

	"celler"
	"celler/layer_storage"
	"neomega-blocks/ir"
	encoder_go "neomega-nbt/encoder-go-object"
	nbt "neomega-nbt/presets/v1"
)

// Chunk retains the complete root document. Section edits are applied explicitly
// with SetSection; other fields remain available in NBT with their native types.
type Chunk struct{ NBT nbt.Document }

func DecodeChunk(data []byte) (*Chunk, error) {
	doc, err := nbt.DecodeDocument(data, nbt.EncodingBigEndian)
	if err != nil {
		return nil, err
	}
	return &Chunk{NBT: doc}, nil
}
func (c *Chunk) Encode() ([]byte, error) { return nbt.EncodeDocument(c.NBT, nbt.EncodingBigEndian) }

// NewChunk initializes a chunk document with explicit format version. World
// generation fields and section contents are supplied by the caller.
func NewChunk(pos celler.ChunkPos, dataVersion int32) *Chunk {
	return &Chunk{NBT: nbt.Document{Value: nbt.Compound{
		"xPos": pos.X, "zPos": pos.Z, "DataVersion": dataVersion, "sections": []nbt.Compound{},
	}}}
}
func (c *Chunk) Position() (celler.ChunkPos, error) {
	x, xOK := c.NBT.Value["xPos"].(int32)
	z, zOK := c.NBT.Value["zPos"].(int32)
	if !xOK || !zOK {
		return celler.ChunkPos{}, fmt.Errorf("chunk requires Int xPos and zPos")
	}
	return celler.ChunkPos{X: x, Z: z}, nil
}

type BlockEntity struct {
	// 保留绝对坐标和原始 NBT，调用方可按自己的 section 或实体类型组织数据。
	Pos celler.BlockPos
	NBT nbt.Compound
}

type BlockEntityIssue struct {
	// 单项错误不阻止读取同一区块中的其他实体，Index 对应原列表下标。
	Index int
	Err   error
}

func (c *Chunk) BlockEntities() ([]BlockEntity, []BlockEntityIssue, error) {
	// 列表损坏与单项坐标损坏分开返回，便于消费者保留可用的实体。
	value, exists := c.NBT.Value["block_entities"]
	if !exists {
		return nil, nil, nil
	}
	values, ok := nbt.Compounds(value)
	if !ok {
		return nil, nil, fmt.Errorf("block_entities must be a compound list (%T)", value)
	}
	if len(values) == 0 {
		return nil, nil, nil
	}
	chunk, err := c.Position()
	if err != nil {
		return nil, nil, err
	}
	var entities []BlockEntity
	var issues []BlockEntityIssue
	for index, value := range values {
		x, xOK := value["x"].(int32)
		y, yOK := value["y"].(int32)
		z, zOK := value["z"].(int32)
		pos := celler.BlockPos{X: x, Y: y, Z: z}
		if !xOK || !yOK || !zOK || pos.Chunk() != chunk {
			issues = append(issues, BlockEntityIssue{
				Index: index,
				Err:   fmt.Errorf("block entity requires Int x/y/z inside its chunk"),
			})
			continue
		}
		entities = append(entities, BlockEntity{Pos: pos, NBT: value})
	}
	return entities, issues, nil
}

// Section contains one block palette and 4×4×4 biome samples in Y-Z-X order.
// Nil Blocks or Biomes means that field is absent, e.g. a light-only section.
// NBT retains the section's other fields, including light arrays.
type Section struct {
	Y       int32
	Blocks  *layer_storage.Palette[ir.IRBlock]
	Biomes  *[64]string
	NBT     nbt.Compound
	entries []nbt.Compound
	states  []ir.IRBlock
}

func (c *Chunk) sectionList() ([]nbt.Compound, error) {
	if _, ok := c.NBT.Value["Level"]; ok {
		return nil, fmt.Errorf("unsupported Java section layout: Level")
	}
	if _, ok := c.NBT.Value["Sections"]; ok {
		return nil, fmt.Errorf("unsupported Java section layout: Sections")
	}
	value, exists := c.NBT.Value["sections"]
	if !exists {
		return nil, nil
	}
	sections, ok := nbt.Compounds(value)
	if !ok {
		return nil, fmt.Errorf("sections must be a compound list")
	}
	return sections, nil
}

func (c *Chunk) Section(y int32) (*Section, error) {
	sections, err := c.sectionList()
	if err != nil {
		return nil, err
	}
	for _, value := range sections {
		sy, ok := value["Y"].(int8)
		if !ok {
			return nil, fmt.Errorf("section requires Byte Y")
		}
		if int32(sy) == y {
			return decodeSection(value)
		}
	}
	return nil, fs.ErrNotExist
}
func (c *Chunk) SectionYs() ([]int32, error) {
	sections, err := c.sectionList()
	if err != nil {
		return nil, err
	}
	ys := make([]int32, 0, len(sections))
	for _, section := range sections {
		y, ok := section["Y"].(int8)
		if !ok {
			return nil, fmt.Errorf("section requires Byte Y")
		}
		ys = append(ys, int32(y))
	}
	return ys, nil
}
func (c *Chunk) SetSection(section *Section) error {
	value, err := section.encode()
	if err != nil {
		return err
	}
	sections, err := c.sectionList()
	if err != nil {
		return err
	}
	next := append([]nbt.Compound(nil), sections...)
	for i, s := range next {
		if y, ok := s["Y"].(int8); ok && int32(y) == section.Y {
			next[i] = value
			c.NBT.Value["sections"] = next
			return nil
		}
	}
	c.NBT.Value["sections"] = append(next, value)
	return nil
}
func (c *Chunk) DeleteSection(y int32) error {
	sections, err := c.sectionList()
	if err != nil {
		return err
	}
	next := make([]nbt.Compound, 0, len(sections))
	for _, s := range sections {
		sy, ok := s["Y"].(int8)
		if !ok {
			return fmt.Errorf("section requires Byte Y")
		}
		if int32(sy) != y {
			next = append(next, s)
		}
	}
	c.NBT.Value["sections"] = next
	return nil
}

func decodeSection(value nbt.Compound) (*Section, error) {
	y, ok := value["Y"].(int8)
	if !ok {
		return nil, fmt.Errorf("section requires Byte Y")
	}
	s := &Section{Y: int32(y), NBT: maps.Clone(value)}
	if raw, exists := value["block_states"]; exists {
		states, ok := raw.(nbt.Compound)
		if !ok {
			return nil, fmt.Errorf("block_states must be a compound")
		}
		entries, ok := nbt.Compounds(states["palette"])
		if !ok || len(entries) == 0 || len(entries) > 4096 {
			return nil, fmt.Errorf("invalid block palette")
		}
		palette := make([]ir.IRBlock, len(entries))
		for i, entry := range entries {
			name, ok := entry["Name"].(string)
			if !ok {
				return nil, fmt.Errorf("palette entry %d requires Name", i)
			}
			props := map[string]any{}
			if raw, exists := entry["Properties"]; exists {
				var ok bool
				props, ok = raw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("Properties must be a compound")
				}
				for key, value := range props {
					if _, ok := value.(string); !ok {
						return nil, fmt.Errorf("Java property %s must be a string", key)
					}
				}
			}
			block, err := ir.NewIRBlockFromGo(name, props)
			if err != nil {
				return nil, err
			}
			palette[i] = block
		}
		indices, err := unpack(states, len(entries), 4096, 4)
		if err != nil {
			return nil, err
		}
		s.Blocks = layer_storage.MakePaletteStorage(palette[0])
		s.Blocks.Fill(bits.Len(uint(len(palette)-1)), palette[0])
		s.Blocks.SetPalette(palette)
		for i, index := range indices {
			s.Blocks.SetIndex(layer_storage.At(i&15, i>>8, i>>4&15), index)
		}
		s.entries = entries
		s.states = palette
	}
	if raw, exists := value["biomes"]; exists {
		biomes, ok := raw.(nbt.Compound)
		if !ok {
			return nil, fmt.Errorf("biomes must be a compound")
		}
		palette, ok := biomes["palette"].([]string)
		if !ok || len(palette) == 0 || len(palette) > 64 {
			return nil, fmt.Errorf("invalid biome palette")
		}
		indices, err := unpack(biomes, len(palette), 64, 1)
		if err != nil {
			return nil, err
		}
		s.Biomes = new([64]string)
		for i, index := range indices {
			s.Biomes[i] = palette[index]
		}
	}
	return s, nil
}

func (s *Section) encode() (nbt.Compound, error) {
	if s.Y < -128 || s.Y > 127 {
		return nil, fmt.Errorf("section Y outside Byte range: %d", s.Y)
	}
	value := maps.Clone(s.NBT)
	if value == nil {
		value = nbt.Compound{}
	}
	value["Y"] = int8(s.Y)
	if s.Blocks == nil {
		delete(value, "block_states")
	} else {
		palette := s.Blocks.GetUsedPalette()
		entries := make([]nbt.Compound, len(palette))
		for i, block := range palette {
			if !block.IsTidy() {
				return nil, fmt.Errorf("palette entry %d must be tidy IR", i)
			}
			if i < len(s.states) && block == s.states[i] {
				entries[i] = s.entries[i]
				continue
			}
			var props map[string]any
			if err := block.States().CastToString().DecodeToHL(encoder_go.NewGoObjEncoder(&props, encoder_go.EncodeOption{})); err != nil {
				return nil, err
			}
			entries[i] = nbt.Compound{"Name": block.NameWithNameSpace()}
			if len(props) > 0 {
				entries[i]["Properties"] = props
			}
		}
		states, _ := value["block_states"].(nbt.Compound)
		states = maps.Clone(states)
		if states == nil {
			states = nbt.Compound{}
		}
		states["palette"] = entries
		indices := make([]int, 4096)
		for i := range indices {
			indices[i] = s.Blocks.GetIndex(layer_storage.At(i&15, i>>8, i>>4&15))
		}
		if err := pack(states, indices, len(palette), 4); err != nil {
			return nil, err
		}
		value["block_states"] = states
	}
	if s.Biomes == nil {
		delete(value, "biomes")
	} else {
		palette := []string{}
		lookup := map[string]int{}
		indices := make([]int, 64)
		for i, name := range s.Biomes {
			index, ok := lookup[name]
			if !ok {
				index = len(palette)
				lookup[name] = index
				palette = append(palette, name)
			}
			indices[i] = index
		}
		biomes, _ := value["biomes"].(nbt.Compound)
		biomes = maps.Clone(biomes)
		if biomes == nil {
			biomes = nbt.Compound{}
		}
		biomes["palette"] = palette
		if err := pack(biomes, indices, len(palette), 1); err != nil {
			return nil, err
		}
		value["biomes"] = biomes
	}
	return value, nil
}

// Modern Java palettes pack whole indexes into 64-bit words, leaving padding.
func unpack(value nbt.Compound, paletteSize, count, minBits int) ([]int, error) {
	out := make([]int, count)
	if paletteSize == 1 {
		return out, nil
	}
	width := max(minBits, bits.Len(uint(paletteSize-1)))
	perWord := 64 / width
	words, ok := nbt.ArrayValues[int64](value["data"])
	if !ok || len(words) != (count+perWord-1)/perWord {
		return nil, fmt.Errorf("invalid palette data length")
	}
	mask := uint64(1<<width) - 1
	for i := range out {
		out[i] = int(uint64(words[i/perWord]) >> ((i % perWord) * width) & mask)
		if out[i] >= paletteSize {
			return nil, fmt.Errorf("palette index out of range at %d", i)
		}
	}
	return out, nil
}
func pack(value nbt.Compound, indices []int, paletteSize, minBits int) error {
	if paletteSize < 1 {
		return fmt.Errorf("empty palette")
	}
	width := max(minBits, bits.Len(uint(paletteSize-1)))
	perWord := 64 / width
	words := make([]int64, (len(indices)+perWord-1)/perWord)
	for i, index := range indices {
		if index < 0 || index >= paletteSize {
			return fmt.Errorf("palette index out of range at %d", i)
		}
		words[i/perWord] |= int64(uint64(index) << ((i % perWord) * width))
	}
	if paletteSize == 1 {
		delete(value, "data")
	} else {
		value["data"] = nbt.Array(words)
	}
	return nil
}
