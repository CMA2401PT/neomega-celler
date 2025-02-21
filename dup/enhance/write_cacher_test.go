package enhance

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/grid"
	"celler/subchunk_storage/low_io"
	"celler/subchunk_storage/simple"
	"fmt"
	"testing"
)

func TestWriteCacher(t *testing.T) {
	newGrid := func() grid.GridSubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]]] {
		return simple.NewLazySimple(subchunk.StandardVersionedFactory(uint32(100), uint32(200)))
	}
	lowStorage := low_io.NewSimpleGridIO(low_io.NoCompress{}, (64 * 64))
	makerFn := func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]] {
		return NewWriteCacher(subchunk.StandardVersionedFactory(uint32(100), uint32(200)), grid.NewIOGrid(newGrid, 2, 2, 2, (64*64), lowStorage), (32 * 32))
	}

	storageI, datas, removes := testStorageWrite(t, makerFn)
	s := storageI.(*WriteCacher[
		subchunk.Versioned[
			uint32, uint32,
			*layer_storage.Palette[uint32],
			*layer_storage.Palette[uint32],
			*layer_storage.Map[celler.SNBT],
		],
		*subchunk.Versioned[
			uint32, uint32,
			*layer_storage.Palette[uint32],
			*layer_storage.Palette[uint32],
			*layer_storage.Map[celler.SNBT],
		],
		*grid.IOGrid[
			subchunk.Versioned[
				uint32, uint32,
				*layer_storage.Palette[uint32],
				*layer_storage.Palette[uint32],
				*layer_storage.Map[celler.SNBT]],
			*subchunk.Versioned[
				uint32, uint32,
				*layer_storage.Palette[uint32],
				*layer_storage.Palette[uint32],
				*layer_storage.Map[celler.SNBT],
			],
		],
	])
	s.FlushCache()
	s.Base.DumpAll()
	fmt.Printf("io rw: %v %v\n", lowStorage.ReaderCounter, lowStorage.WriterCounter)
	fmt.Printf("size: %v\n", lowStorage.TotalSize()/(1024*1024))
	testStorageRead(t, storageI, datas, removes)
}
