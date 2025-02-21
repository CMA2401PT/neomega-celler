package simple

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"testing"
)

func TestSimple(t *testing.T) {
	makerFn := func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]] {
		return NewSimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	}
	storage, datas, removes := testStorageWrite(t, makerFn)
	testStorageRead(t, storage, datas, removes)
}
