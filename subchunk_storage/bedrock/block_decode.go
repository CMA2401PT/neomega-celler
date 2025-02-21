package bedrock

import (
	"fmt"
	"neomega_blocks"
	"neomega_blocks/description"
	"neomega_blocks/preset"
	"neomega_nbt"
	"neomega_nbt/encoding"
	"unsafe"
)

var decodeCache map[string]struct {
	v   neomega_blocks.UIBI
	err error
} = nil

func DefaultBlockDecode(rawNbt []byte) (id neomega_blocks.UIBI, err error) {
	cacheKey := *(*string)(unsafe.Pointer(&rawNbt))
	if decodeCache == nil {
		decodeCache = make(map[string]struct {
			v   neomega_blocks.UIBI
			err error
		})
		preset.Load()
	}
	if i, found := decodeCache[cacheKey]; found {
		return i.v, i.err
	}
	cacheKey = string(rawNbt)
	defer func() {
		decodeCache[cacheKey] = struct {
			v   neomega_blocks.UIBI
			err error
		}{v: id, err: err}
		if err != nil {
			fmt.Println(err)
		}
	}()
	_, value, err := neomega_nbt.NBTDecode[encoding.LittleEndian](rawNbt)
	bNbt, ok := value.(map[string]any)
	if !ok {
		return neomega_blocks.AIR, fmt.Errorf("invalid nbt data")
	}
	var name string
	var state map[string]any
	nameI, ok := bNbt["name"]
	if ok {
		name, _ = nameI.(string)
	}
	stateI, ok := bNbt["states"]
	if ok {
		state, _ = stateI.(map[string]any)
	}
	blk, err := description.ParseBlockState(name, state, description.DefaultCaster)
	if err != nil {
		return neomega_blocks.AIR, fmt.Errorf("fail to parse nbt block")
	}
	uniID, found := preset.BedrockPrimaryOrJavaToUIBI.BlockToBestID(blk, description.ScoreFnDefault)
	if !found {
		return neomega_blocks.AIR, fmt.Errorf("cannot find uni id")
	}
	return uniID, nil
}
