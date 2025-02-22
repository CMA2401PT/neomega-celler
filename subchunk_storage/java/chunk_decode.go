package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"errors"
	"fmt"
	"math"
	"neomega_blocks"
	"neomega_blocks/description"
	"neomega_blocks/preset"
	"neomega_nbt"
	"neomega_nbt/caster"
	"neomega_nbt/encoding"
)

type ChunkDecoder struct{}

var cd = ChunkDecoder{}

func (cd *ChunkDecoder) DecodeChunkFromData(data []byte) (*ChunkWithChangeMark, error) {

	_, chunkNBTI, err := neomega_nbt.NBTDecode[encoding.BigEndian](data)
	if err != nil {
		return nil, err
	}
	chunkNBT, ok := chunkNBTI.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%v is not map[string]any", chunkNBT)
	}
	blockStatesI := chunkNBT["block_entities"]
	snbts := map[celler.ChunkBlockPos]celler.SNBT{}
	{
		if blockStatesI != nil {
			snbts = cd.DecodeNBT(blockStatesI.([]any))
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
		Subchunks: make(map[celler.ChunkUnit]*subchunk.Simple[neomega_blocks.UIBI, uint32, *layer_storage.Palette[neomega_blocks.UIBI], *layer_storage.PlaceHolder[uint32], *layer_storage.Map[celler.SNBT]]),
		Changed:   false,
	}
	sections, ok := sectionsI.([]any)
	if !ok {
		panic("")
	} else {
		for _, sectionI := range sections {
			y, c, err := cd.decodeSubChunkFromSection(sectionI)
			out.Subchunks[y] = c
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
	neomega_blocks.UIBI, uint32,
	*layer_storage.Palette[neomega_blocks.UIBI],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
], error) {
	section := sectionI.(map[string]any)
	Y := section["Y"].(int8)
	// BlockLight := section["BlockLight"].([]int8)
	BlockStates := section["block_states"].(map[string]any)
	BlocksDataI := BlockStates["data"]
	if BlocksDataI == nil {
		return celler.ChunkUnit(Y), nil, nil
	}
	BlocksData := BlocksDataI.([]int64)
	BlockPaletteI := BlockStates["palette"].([]any)
	palette := make([]neomega_blocks.UIBI, len(BlockPaletteI))
	if len(BlockStates) == 0 {
		return celler.ChunkUnit(Y), nil, nil
	}

	var level int
	var get func(blockI int) int
	{
		blocksPerLong := (4096 + len(BlocksData) - 1) / len(BlocksData)
		bitsPerBlock := 64 / blocksPerLong
		level = bitsPerBlock
		origMask := int(1<<bitsPerBlock - 1)
		get = func(blockI int) int {
			longI := blockI / blocksPerLong
			offsetBit := (blockI - longI*blocksPerLong) * bitsPerBlock
			return int(BlocksData[longI]>>offsetBit) & origMask
		}
	}

	sc := &layer_storage.Palette[neomega_blocks.UIBI]{}
	sc.Fill(level, neomega_blocks.AIR)
	for bpi, bp := range BlockPaletteI {
		palette[bpi] = cd.DecodeBlock(bp.(map[string]any))
	}

	_i := 0
	for _y := uint8(0); _y < 16; _y++ {
		for _z := uint8(0); _z < 16; _z++ {
			for _x := uint8(0); _x < 16; _x++ {
				v := get(_i)
				_i++
				sc.SetIndex(celler.ToSubChunkBlockPos(_x, _y, _z), v)
			}
		}
	}

	sc.SetPalette(palette)
	return celler.ChunkUnit(Y), &subchunk.Simple[neomega_blocks.UIBI, uint32, *layer_storage.Palette[neomega_blocks.UIBI], *layer_storage.PlaceHolder[uint32], *layer_storage.Map[celler.SNBT]]{
		CurrentVersion: 2, MetaData: nil,
		BlockIDStorage:     sc,
		BlockAttachStorage: layer_storage.New(uint32(0)).PlaceHolder(),
		BlockSNBTStorage:   layer_storage.New(celler.SNBT("")).Map(),
	}, nil
}

func (cd *ChunkDecoder) DecodeNBT(nbts []any) map[celler.ChunkBlockPos]celler.SNBT {
	if len(nbts) == 0 {
		return nil
	}
	out := map[celler.ChunkBlockPos]celler.SNBT{}
	for _, blockNbtI := range nbts {
		blockNbt, ok := blockNbtI.(map[string]any)
		if !ok {
			continue
		}
		x := neomega_nbt.ReadFromNBT(blockNbt, "x", math.MaxInt)
		y := neomega_nbt.ReadFromNBT(blockNbt, "y", math.MaxInt16)
		z := neomega_nbt.ReadFromNBT(blockNbt, "z", math.MaxInt)
		delete(blockNbt, "x")
		delete(blockNbt, "y")
		delete(blockNbt, "z")
		snbt, err := neomega_nbt.SNBTEncode(blockNbt, caster.DefaultCaster)
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

func (cd *ChunkDecoder) DecodeBlock(blockNbt map[string]any) neomega_blocks.UIBI {
	if blockNbt == nil {
		return neomega_blocks.AIR
	}
	blockName := blockNbt["Name"].(string)
	var props map[string]any
	if propsI, ok := blockNbt["Properties"]; ok && propsI != nil {
		props = propsI.(map[string]any)
	}
	if props != nil {
		delete(props, "waterlogged")
	}
	blk, err := description.ParseBlockState(blockName, props, description.DefaultCaster)
	if err != nil {
		return neomega_blocks.AIR
	}
	id, _ := preset.JavaToUIBI.BlockToBestID(blk, description.ScoreFnDefault)
	return id
}
