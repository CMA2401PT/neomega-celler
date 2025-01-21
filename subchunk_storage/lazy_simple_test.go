package subchunk_storage

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/low_io"
	"fmt"
	"testing"
	"time"
)

func TestLazySimpleIO(t *testing.T) {
	makerFn := func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]] {
		return NewLazySimple(subchunk.StandardVersionedFactory(uint32(100), uint32(200)))
	}

	storageI, datas, removes := testStorageWrite(t, makerFn)
	storage := storageI.(*LazySimple[subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	], *subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT]]])
	lowStorage := low_io.NewSimpleMemoryIOToBytes(nil)

	st := time.Now()
	storage.DumpData(lowStorage)
	lowStorage.Close()
	fmt.Printf("dump time: %v\n", time.Since(st))

	lowStorageBack := low_io.NewSimpleMemoryIOToBytes(lowStorage.Bytes())
	back := NewLazySimple(subchunk.StandardVersionedFactory(uint32(100), uint32(200)))
	st = time.Now()
	back.LoadData(lowStorageBack)
	fmt.Printf("load time: %v\n", time.Since(st))

	testStorageRead(t, back, datas, removes)
}
