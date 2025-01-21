package subchunk_storage

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/low_io"
	"fmt"
	"testing"
)

func TestGridIO(t *testing.T) {
	// 随机数是难以压缩的，所以我们这边先不考虑压缩
	newGrid := func() GridSubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]]] {
		return NewLazySimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	}
	storage := low_io.NewSimpleGridIO(low_io.NoCompress{}, 2*(64*64)/(4*4))
	makerFn := func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]] {
		return NewIOGrid(newGrid, 2, 2, 2, (64 * 64 * 32), storage)
	}

	storageI, datas, removes := testStorageWrite(t, makerFn)
	s := storageI.(*IOGrid[subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	], *subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT]]])
	s.DumpAll()
	fmt.Printf("size: %v\n", storage.TotalSize()/(1024*1024))
	testStorageRead(t, storageI, datas, removes)
}
