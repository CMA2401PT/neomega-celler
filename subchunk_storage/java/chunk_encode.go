package java

import (
	"celler"
	"celler/layer_storage"
	"fmt"
	"math/bits"

	"celler/internal/nbtutil"
	blocks "neomega-blocks/presets/block-converts/v1"

	nbt "neomega-nbt/presets/v1"
)

type ChunkEncoder struct{}

var chunkEncode = ChunkEncoder{}

func (ce ChunkEncoder) EncodeChunk(pos celler.ChunkPos, c *ChunkWithChangeMark) []byte {
	if c == nil {
		data, _ := nbtutil.Encode(map[string]any{}, nbt.EncodingBigEndian)
		return data
	}
	if c.Subchunks == nil {
		return nil
	}
	chunkNBT := map[string]any{}
	var blockStates []any
	{
		snbts := map[celler.ChunkBlockPos]celler.SNBT{}
		for y, sc := range c.Subchunks {
			for iscp, snbt := range sc.BlockSNBTStorage.IterNonDefault {
				p := celler.ChunkBlockPos{
					X: iscp.X, Z: iscp.Z, Y: int16(iscp.Y) | int16(y)<<4,
				}
				snbts[p] = snbt
			}
		}
		blockStates = ce.EncodeNBT(pos, snbts)
	}

	sections := []any{}
	for y, sc := range c.Subchunks {
		section := map[string]any{
			"Y":            int8(y),
			"block_states": ce.EncodeBlockStates(sc.BlockIDStorage),
		}
		sections = append(sections, section)
	}
	chunkNBT["sections"] = sections

	chunkNBT["block_entities"] = blockStates
	chunkNBT["yPos"] = int32(-4)
	chunkNBT["zPos"] = int32(pos.Z)
	chunkNBT["xPos"] = int32(pos.X)
	chunkNBT["Status"] = "minecraft:full"
	chunkNBT["InhabitedTime"] = int64(50)
	chunkNBT["DataVersion"] = int32(3839)
	data, _ := nbtutil.Encode(chunkNBT, nbt.EncodingBigEndian)
	return data
}

func (ChunkEncoder) EncodeNBT(cp celler.ChunkPos, snbts map[celler.ChunkBlockPos]celler.SNBT) []any {
	if len(snbts) == 0 {
		return []any{}
	}
	out := []any{}
	for icp, snbt := range snbts {
		X := int32(cp.X)<<4 | int32(icp.X&0xf)
		Z := int32(cp.Z)<<4 | int32(icp.Z&0xf)
		Y := int32(icp.Y)
		blockNBT, err := nbtutil.FromSNBT(string(snbt))
		if err != nil {
			fmt.Printf("snbt decode %v @ %v %v %v -> nbt err: %v\n", string(snbt), X, Y, Z, err)
			continue
		}
		if blockNBT == nil {
			fmt.Printf(" %v @ %v %v %v is not map[string]any\n", string(snbt), X, Y, Z)
			continue
		}
		blockNBT["x"] = X
		blockNBT["y"] = Y
		blockNBT["z"] = Z
		out = append(out, blockNBT)
	}
	return out
}

func (ChunkEncoder) EncodeBlockStates(p *layer_storage.Palette[uint32]) map[string]any {
	bs := map[string]any{}
	var blockData []int64
	blockPalette := []map[string]any{}
	ps := p.GetUsedPalette()
	for _, uibi := range ps {
		bedrock, _ := blocks.BedrockHashToBlockStrict(uibi)
		blk, ok := blocks.ConvertToJava(bedrock)
		if !ok {
			blk = "minecraft:air []"
		}
		blockNbt := map[string]any{}
		blockNbt["Name"] = blk.NameWithNameSpace()
		blockNbt["Properties"] = blk.States().GoMap()
		blockPalette = append(blockPalette, blockNbt)
	}

	if len(ps) == 1 {
		return map[string]any{"palette": blockPalette}
	}
	var set func(blockI int, v uint32)
	{
		bitsPerBlock := bits.Len(uint(len(ps) - 1))
		if bitsPerBlock <= 4 {
			bitsPerBlock = 4
		}
		blocksPerLong := 64 / bitsPerBlock
		totalLong := (4096 + blocksPerLong - 1) / blocksPerLong
		blockData = make([]int64, totalLong)
		set = func(blockI int, v uint32) {
			longI := blockI / blocksPerLong
			offsetBit := (blockI % blocksPerLong) * bitsPerBlock
			blockData[longI] |= int64(v) << int32(offsetBit)
		}
	}

	for p, v := range p.IterNonDefaultPaletteI {
		_i := int(p.X&0xf) + int(p.Z&0xf)<<4 + int(p.Y&0xf)<<8
		set(_i, uint32(v))
	}

	bs["data"] = blockData
	bs["palette"] = blockPalette
	return bs
}
