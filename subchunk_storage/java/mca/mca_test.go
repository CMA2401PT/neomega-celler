package mca

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMCA(t *testing.T) {
	srcStor, err := OpenDirReadOnly("../data/region")
	if err != nil {
		t.Fatal(err)
	}

	dstStor, err := OpenDirReadWrite("../cpy/region")
	if err != nil {
		t.Fatal(err)
	}
	dstKV := NewReadWriteMCA(dstStor)

	srcKV := NewReadOnlyMCA(srcStor)
	// data, err := srcKV.Get(celler.ChunkPos{1, 1})
	// fmt.Println(len(data), err)

	for chunkPos := range srcKV.IterAll {
		data, _ := srcKV.Get(chunkPos)
		dstKV.Put(chunkPos, data)
	}
	dstKV.Close()

	dstStor, err = OpenDirReadWrite("../cpy/region")
	if err != nil {
		t.Fatal(err)
	}
	dstKV = NewReadWriteMCA(dstStor)
	// data, err := srcKV.Get(celler.ChunkPos{1, 1})
	// fmt.Println(len(data), err)

	for chunkPos := range srcKV.IterAll {
		fmt.Println(chunkPos)
		data, _ := srcKV.Get(chunkPos)
		dataBack, _ := dstKV.Get(chunkPos)
		if !bytes.Equal(data, dataBack) {
			t.Fatal("not same")
		}
	}
	dstKV.Close()

	// rd, err := stor.OpenRegion(regionPos{1, -2})
	// fmt.Println(rd, err)
	// rr, err := newRoRegion(rd)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// data, err := rr.GetChunkData(inRegionPos{Cx: 24, Cz: 10})
	// fmt.Println(data, err)
	// fmt.Println(rr)
}
