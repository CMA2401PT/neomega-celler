package subchunk_storage

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"testing"
)

func TestGrid(t *testing.T) {
	testStorage(t, func() celler.SubChunkStorage[
		*subchunk.Versioned[
			uint32, uint32,
			*layer_storage.Palette[uint32],
			*layer_storage.Palette[uint32],
			*layer_storage.Map[celler.SNBT],
		],
		celler.SubChunkDataChange[uint32, uint32],
	] {
		return NewGrid(uint32(100), uint32(200), subchunk.MakeStandardVersioned, 16, 16)
	})
}
