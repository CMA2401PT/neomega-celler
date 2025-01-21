package enhance

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage"
	"celler/subchunk_storage/low_io"
	"fmt"
	"testing"
)

func TestWriteCacher(t *testing.T) {
	newGrid := func() subchunk_storage.GridSubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]]] {
		return subchunk_storage.NewLazySimple(subchunk.StandardVersionedFactory(uint32(100), uint32(200)))
	}
	lowStorage := low_io.NewSimpleGridIO(low_io.NoCompress{}, (64 * 64))
	makerFn := func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]] {
		return NewWriteCacher(subchunk.StandardVersionedFactory(uint32(100), uint32(200)), subchunk_storage.NewIOGrid(newGrid, 2, 2, 2, (64*64), lowStorage), (32 * 32))
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
		*subchunk_storage.IOGrid[
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
