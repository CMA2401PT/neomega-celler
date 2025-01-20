package subchunk

import (
	"celler"
	"celler/layer_storage"
)

func MakeStandardSimple[BlockIDT ~uint32, AttachT ~uint32](blockID BlockIDT, attach AttachT) *Simple[BlockIDT, AttachT,
	*layer_storage.Palette[BlockIDT],
	*layer_storage.Palette[AttachT],
	*layer_storage.Map[celler.SNBT],
] {
	return &Simple[BlockIDT, AttachT,
		*layer_storage.Palette[BlockIDT],
		*layer_storage.Palette[AttachT],
		*layer_storage.Map[celler.SNBT],
	]{
		CurrentVersion:     RandomInitVersion(),
		MetaData:           map[string]string{},
		BlockIDStorage:     layer_storage.MakePaletteStorage(blockID),
		BlockAttachStorage: layer_storage.MakePaletteStorage(attach),
		BlockSNBTStorage:   layer_storage.MakeMapStorage(celler.SNBT("")),
	}
}

func MakeStandardVersioned[BlockIDT ~uint32, AttachT ~uint32](blockID BlockIDT, attach AttachT) *Versioned[BlockIDT, AttachT,
	*layer_storage.Palette[BlockIDT],
	*layer_storage.Palette[AttachT],
	*layer_storage.Map[celler.SNBT],
] {
	return &Versioned[BlockIDT, AttachT,
		*layer_storage.Palette[BlockIDT],
		*layer_storage.Palette[AttachT],
		*layer_storage.Map[celler.SNBT],
	]{
		Simple:      *MakeStandardSimple(blockID, attach),
		versionData: make([]celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT], 0),
	}
}
