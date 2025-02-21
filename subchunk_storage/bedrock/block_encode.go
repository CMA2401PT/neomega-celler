package bedrock

import (
	"encoding/binary"
	"fmt"
	"neomega_blocks"
	"neomega_blocks/preset"
	"neomega_nbt"
	"neomega_nbt/caster"
	"neomega_nbt/encoding"
)

var encodeCache map[neomega_blocks.UIBI]struct {
	v   []byte
	err error
} = nil

const chunkVersion = 27

var blockVersion = int32(binary.BigEndian.Uint32([]byte{1, 20, 50, 0})) //1.20.50.0
var airNBT = []byte{10, 0, 0, 8, 4, 0, 110, 97, 109, 101, 13, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 97, 105, 114, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7, 0, 118, 101, 114, 115, 105, 111, 110, 0, 50, 20, 1, 0}

//					^TAG  TAG_S^ ^4    ^n   a   m    e    ^13    ^m    i   n     e    c   r    a   f    t    :   a   i  r  TAG_C^  ^6    ^s    t   a    t    e    s  ^END^TAG^7   ^v    e    r    s    i    o   n   6. 20. 21. 1 ^END

func DefaultBlockEncode(id neomega_blocks.UIBI) (rawNbt []byte, err error) {
	if encodeCache == nil {
		encodeCache = make(map[neomega_blocks.UIBI]struct {
			v   []byte
			err error
		})
		preset.Load()
		// eo, _ := DefaultBlockEncode(neomega_blocks.AIR)
		// if !bytes.Equal(eo, airNBT) {
		// 	panic("air or version changed!")
		// }
	}
	if i, found := encodeCache[id]; found {
		return i.v, i.err
	}
	defer func() {
		encodeCache[id] = struct {
			v   []byte
			err error
		}{v: rawNbt, err: err}
		if err != nil {
			fmt.Println(err)
		}
	}()

	block, found := preset.UIBIToBedrock_1_20_50.IDToBlock(id)
	if !found {
		return airNBT, fmt.Errorf("cannot find bedrock nbt")
	}
	states := block.GoMapProps()
	name := block.BaseName().LongName()

	rawNbt, err = neomega_nbt.NBTEncode[encoding.LittleEndian]("", map[string]any{
		"name":   string(name),
		"states": states,
		// "version": blockVersion,
	}, caster.DefaultCaster)
	if err != nil {
		return airNBT, fmt.Errorf("fail to encode nbt block")
	}

	return rawNbt, nil
}
