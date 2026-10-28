package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"neomega-blocks/ir"

	"celler/internal/nbtutil"
	blocks "neomega-blocks/presets/block-converts/v1"

	nbt "neomega-nbt/presets/v1"
)

type ChunkDecoder struct{}

var cd = ChunkDecoder{}

func (cd *ChunkDecoder) DecodeChunkFromData(data []byte) (*ChunkWithChangeMark, error) {

	chunkNBT, err := nbtutil.Decode(data, nbt.EncodingBigEndian)
	if err != nil {
		return nil, err
	}

	blockStatesI := chunkNBT["block_entities"]
	snbts := map[celler.ChunkBlockPos]celler.SNBT{}
	{
		if blockStatesI != nil {
			entities, ok := nbtutil.Compounds(blockStatesI)
			if !ok {
				return nil, fmt.Errorf("block_entities is not a list")
			}
			snbts = cd.DecodeNBT(entities)
		}
	}
	if sectionsI, ok := chunkNBT["sections"]; ok {
		return cd.decodeChunkFromSections(sectionsI, snbts)
	}
	if sectionsI, ok := chunkNBT["Sections"]; ok {
		return cd.decodeChunkFromSections(sectionsI, snbts)
	}
	return nil, fmt.Errorf("unrecognized section structure")
}

func (cd *ChunkDecoder) decodeChunkFromSections(sectionsI any, snbts map[celler.ChunkBlockPos]celler.SNBT) (out *ChunkWithChangeMark, err error) {
	errStr := ""
	out = &ChunkWithChangeMark{
		Subchunks: make(map[celler.ChunkUnit]*subchunk.Simple[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.PlaceHolder[uint32], *layer_storage.Map[celler.SNBT]]),
		Changed:   false,
	}
	sections, ok := nbtutil.Compounds(sectionsI)
	if !ok {
		return nil, fmt.Errorf("sections is not a list")
	} else {
		for _, sectionI := range sections {
			y, c, err := cd.decodeSubChunkFromSection(sectionI)
			if c != nil {
				out.Subchunks[y] = c
			}
			if err != nil {
				errStr += ";" + err.Error()
			}

		}
	}
	if errStr == "" {
		err = nil
	} else {
		err = errors.New(errStr[1:])
	}
	for pos, snbt := range snbts {
		cY := pos.Y >> 4
		c := out.Subchunks[celler.ChunkUnit(cY)]
		if c != nil {
			c.BlockSNBTStorage.Set(celler.ToSubChunkBlockPos(pos.X, pos.Y&0xf, pos.Z), snbt)
		}
	}
	return out, err
}

func (cd *ChunkDecoder) decodeSubChunkFromSection(sectionI any) (celler.ChunkUnit, *subchunk.Simple[
	uint32, uint32,
	*layer_storage.Palette[uint32],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
], error) {
	section, ok := sectionI.(map[string]any)
	if !ok {
		return 0, nil, fmt.Errorf("section is not a compound")
	}
	Y, ok := section["Y"].(int8)
	if !ok {
		return 0, nil, fmt.Errorf("section has no byte Y")
	}
	states, ok := section["block_states"].(map[string]any)
	if !ok {
		return celler.ChunkUnit(Y), nil, nil
	} // Light-only section.
	entries, ok := nbtutil.Compounds(states["palette"])
	if !ok || len(entries) == 0 {
		return celler.ChunkUnit(Y), nil, fmt.Errorf("empty block palette")
	}
	palette := make([]uint32, len(entries))
	for i, entry := range entries {
		palette[i] = cd.DecodeBlock(entry)
	}
	sc := layer_storage.New(palette[0]).Palette()
	// A single-entry Java palette omits data and fills the entire section.
	if len(palette) > 1 {
		level := max(4, bits.Len(uint(len(palette)-1)))
		perLong := 64 / level
		data, ok := states["data"].([]int64)
		if !ok || len(data) < (4096+perLong-1)/perLong {
			return celler.ChunkUnit(Y), nil, fmt.Errorf("truncated block state data")
		}
		sc.Fill(level, palette[0])
		sc.SetPalette(palette)
		mask := uint64(1<<level) - 1
		for i := 0; i < 4096; i++ {
			index := int(uint64(data[i/perLong]) >> ((i % perLong) * level) & mask)
			if index >= len(palette) {
				return celler.ChunkUnit(Y), nil, fmt.Errorf("palette index %d out of range", index)
			}
			sc.SetIndex(celler.ToSubChunkBlockPos(i&15, i>>8, i>>4&15), index)
		}
	}

	return celler.ChunkUnit(Y), &subchunk.Simple[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.PlaceHolder[uint32], *layer_storage.Map[celler.SNBT]]{
		CurrentVersion: 2, MetaData: nil,
		BlockIDStorage:     sc,
		BlockAttachStorage: layer_storage.New(uint32(0)).PlaceHolder(),
		BlockSNBTStorage:   layer_storage.New(celler.SNBT("")).Map(),
	}, nil
}

func (cd *ChunkDecoder) DecodeNBT(nbts []map[string]any) map[celler.ChunkBlockPos]celler.SNBT {
	if len(nbts) == 0 {
		return nil
	}
	out := map[celler.ChunkBlockPos]celler.SNBT{}
	for _, blockNbt := range nbts {
		x := nbtutil.Coordinate(blockNbt, "x", math.MaxInt)
		y := nbtutil.Coordinate(blockNbt, "y", math.MaxInt16)
		z := nbtutil.Coordinate(blockNbt, "z", math.MaxInt)
		delete(blockNbt, "x")
		delete(blockNbt, "y")
		delete(blockNbt, "z")
		snbt, err := nbtutil.ToSNBT(blockNbt)
		if err != nil {
			fmt.Printf("encode %v @ %v %v %v -> snbt err: %v\n", x, y, z, blockNbt, err)
			continue
		}
		out[celler.ChunkBlockPos{
			X: uint8(x & 0xf), Z: uint8(z & 0xf), Y: int16(y),
		}] = celler.SNBT(snbt)
	}
	return out
}

func (cd *ChunkDecoder) DecodeBlock(blockNbt map[string]any) uint32 {
	if blockNbt == nil {
		return ir.IRBlock("minecraft:air []").Hash()
	}
	blockName := blockNbt["Name"].(string)
	var props map[string]any
	if propsI, ok := blockNbt["Properties"]; ok && propsI != nil {
		props = propsI.(map[string]any)
	}
	blk, err := ir.NewIRBlockFromGo(blockName, props)
	if err != nil {
		return ir.IRBlock("minecraft:air []").Hash()
	}
	converted, ok := blocks.ConvertFromJava(blk)
	if !ok {
		return ir.IRBlock("minecraft:air []").Hash()
	}
	return converted.Hash()
}
