package java

import (
	"celler/subchunk_storage/java/mca"
	"fmt"
	"neomega_blocks/preset"
	"testing"
)

func TestReadOnly(t *testing.T) {
	preset.Load()
	mcaDB, err := mca.NewReadOnlyMCAKvFromDir("./data/region")
	if err != nil {
		panic(err)
	}

	dec := &ChunkDecoder{}

	for cp := range mcaDB.IterAll {
		fmt.Println(cp)
		data, _ := mcaDB.Get(cp)
		fmt.Println(len(data))
		// _, chunkNBT, err := neomega_nbt.NBTDecode[encoding.BigEndian](data)
		// fmt.Println(chunkNBT, err)
		dec.DecodeChunkFromData(data)
	}
}
