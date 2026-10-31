package bedrock

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"

	"celler/layer_storage"
)

// BiomeSection retains one native 16³ numeric palette. Nil Values repeats the
// preceding section. Runtime records the low palette-header bit as stored.
type BiomeSection struct {
	Values  *layer_storage.Palette[int32]
	Runtime bool
}

// Biomes3D stores the Data3D record in its native order. The first section's Y
// comes from the dimension's height definition, not from the record itself.
type Biomes3D struct {
	Heights  [256]int16
	Sections []BiomeSection
}

type Biomes2D struct {
	Heights [256]int16
	Values  [256]byte
}

func (c *Chunk) Biomes2D() (Biomes2D, error) {
	data, ok := c.Records[string([]byte{TagData2D})]
	if !ok {
		return Biomes2D{}, fs.ErrNotExist
	}
	if len(data) != 768 {
		return Biomes2D{}, fmt.Errorf("invalid Data2D length")
	}
	var out Biomes2D
	if err := binary.Read(bytes.NewReader(data[:512]), binary.LittleEndian, &out.Heights); err != nil {
		return Biomes2D{}, err
	}
	copy(out.Values[:], data[512:])
	return out, nil
}
func (c *Chunk) SetBiomes2D(b Biomes2D) {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, b.Heights)
	out.Write(b.Values[:])
	if c.Records == nil {
		c.Records = map[string][]byte{}
	}
	c.Records[string([]byte{TagData2D})] = out.Bytes()
}
func (c *Chunk) Biomes3D() (*Biomes3D, error) {
	data, ok := c.Records[string([]byte{TagData3D})]
	if !ok {
		return nil, fs.ErrNotExist
	}
	r := bytes.NewReader(data)
	out := &Biomes3D{}
	if err := binary.Read(r, binary.LittleEndian, &out.Heights); err != nil {
		return nil, err
	}
	for r.Len() > 0 {
		header, _ := r.ReadByte()
		if header == 255 {
			if len(out.Sections) == 0 {
				return nil, fmt.Errorf("first biome section cannot inherit")
			}
			out.Sections = append(out.Sections, BiomeSection{})
			continue
		}
		width := int(header >> 1)
		words, err := readWords(r, width)
		if err != nil {
			return nil, err
		}
		count := int32(1)
		if width != 0 {
			if err = binary.Read(r, binary.LittleEndian, &count); err != nil {
				return nil, err
			}
		}
		if count <= 0 || int64(count) > 1<<width {
			return nil, fmt.Errorf("invalid biome palette size")
		}
		values := make([]int32, count)
		if err = binary.Read(r, binary.LittleEndian, values); err != nil {
			return nil, err
		}
		palette := layer_storage.MakePaletteStorage(values[0])
		palette.Fill(width, values[0])
		palette.SetPalette(values)
		palette.Data = words
		for i := 0; i < 4096; i++ {
			if palette.GetIndex(layer_storage.At(i>>8, i&15, i>>4&15)) >= int(count) {
				return nil, fmt.Errorf("biome palette index out of range")
			}
		}
		out.Sections = append(out.Sections, BiomeSection{Values: palette, Runtime: header&1 != 0})
	}
	return out, nil
}
func (c *Chunk) SetBiomes3D(b *Biomes3D) error {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, b.Heights)
	for i, section := range b.Sections {
		if section.Values == nil {
			if i == 0 {
				return fmt.Errorf("first biome section cannot inherit")
			}
			out.WriteByte(255)
			continue
		}
		palette := section.Values.GetUsedPalette()
		if len(palette) == 0 || len(palette) > 65536 {
			return fmt.Errorf("invalid biome palette size")
		}
		width := diskWidth(len(palette))
		header := byte(width << 1)
		if section.Runtime {
			header |= 1
		}
		out.WriteByte(header)
		words, err := packWords(section.Values, width)
		if err != nil {
			return err
		}
		binary.Write(&out, binary.LittleEndian, words)
		if width != 0 {
			binary.Write(&out, binary.LittleEndian, int32(len(palette)))
		}
		binary.Write(&out, binary.LittleEndian, palette)
	}
	if c.Records == nil {
		c.Records = map[string][]byte{}
	}
	c.Records[string([]byte{TagData3D})] = out.Bytes()
	return nil
}
