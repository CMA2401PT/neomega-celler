package subchunk

import (
	"celler"
	"celler/storage_io"
	"fmt"
	"io"
)

type Simple[
	BlockIDT ~uint32, AttachT ~uint32,
	BlockStorageT interface {
		celler.Storage[BlockIDT]
		celler.In
		celler.Out
	},
	AttachStorageT interface {
		celler.Storage[AttachT]
		celler.In
		celler.Out
	},
	SNBTStorageT interface {
		celler.Storage[celler.SNBT]
		celler.In
		celler.Out
	},
] struct {
	CurrentVersion     celler.ChunkVersion
	MetaData           map[string]string
	BlockIDStorage     BlockStorageT
	BlockAttachStorage AttachStorageT
	BlockSNBTStorage   SNBTStorageT
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Version() celler.ChunkVersion {
	if sc == nil {
		return 0
	}
	return sc.CurrentVersion
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Meta() map[string]string {
	return sc.MetaData
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) BlockID() celler.LayerStorageRandomRead[BlockIDT] {
	return sc.BlockIDStorage
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) BlockAttach() celler.LayerStorageRandomRead[AttachT] {
	return sc.BlockAttachStorage
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) BlockSNBT() celler.LayerStorageRandomRead[celler.SNBT] {
	return sc.BlockSNBTStorage
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) QueryChange(cacheVersion celler.ChunkVersion) celler.SubChunkChange[BlockIDT, AttachT] {
	if sc == nil {
		return celler.SubChunkChange[BlockIDT, AttachT]{
			Valid: cacheVersion == 0,
			Drop:  true,
		}
	}
	if cacheVersion == sc.CurrentVersion {
		// no change
		return celler.SubChunkChange[BlockIDT, AttachT]{
			Valid: true,
		}
	}
	if cacheVersion > sc.CurrentVersion {
		panic(fmt.Errorf("version %v > %v", cacheVersion, sc.CurrentVersion))
	}
	return celler.SubChunkChange[BlockIDT, AttachT]{
		Valid: false,
		Drop:  false,
		Base:  sc,
	}
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) changeBlock(delta celler.BlockChange[BlockIDT, AttachT]) {
	if delta.Type&celler.BlockDeltaChangeBlockID != 0 {
		sc.BlockIDStorage.Set(delta.Pos, delta.BlockID)
	}
	if delta.Type&celler.BlockDeltaChangeAttach != 0 {
		sc.BlockAttachStorage.Set(delta.Pos, delta.Attach)
	}
	if delta.Type&celler.BlockDeltaChangeSNBT != 0 {
		sc.BlockSNBTStorage.Set(delta.Pos, delta.SNBT)
	}
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlock(delta celler.BlockChange[BlockIDT, AttachT]) {
	sc.CurrentVersion += 1
	sc.changeBlock(delta)
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlockWithVersion(delta celler.BlockChange[BlockIDT, AttachT], newVersion celler.ChunkVersion) {
	sc.CurrentVersion = newVersion
	sc.changeBlock(delta)
}

func (sc *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ApplyChange(change celler.SubChunkChange[BlockIDT, AttachT]) {
	if change.Valid {
		return
	}
	if change.Drop {
		panic("drop must be handled by subchunk holder, not subchunk itself")
	}
	if change.Base != nil {
		if change.Base.Version() > sc.CurrentVersion {
			sc.CurrentVersion = change.Base.Version()
			sc.MetaData = map[string]string{}
			for k, v := range change.Base.Meta() {
				sc.MetaData[k] = v
			}
			sc.BlockIDStorage.CopyFrom(change.Base.BlockID())
			sc.BlockAttachStorage.CopyFrom(change.Base.BlockAttach())
			sc.BlockSNBTStorage.CopyFrom(change.Base.BlockSNBT())
		}
	}
	if len(change.BlockChanges) > 0 {
		for _, delta := range change.BlockChanges {
			if delta.BeforeVersion < sc.CurrentVersion {
				continue
			}
			sc.ChangeBlockWithVersion(delta.BlockChange, delta.AfterVersion)
		}
	}
}

func (s *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Dump(w io.Writer) {
	storage_io.WriteValue(w, s.CurrentVersion)
	storage_io.WriteMap(w, s.MetaData, true)
	s.BlockIDStorage.Dump(w)
	s.BlockAttachStorage.Dump(w)
	s.BlockSNBTStorage.Dump(w)
}

func (s *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Load(r io.Reader) {
	s.CurrentVersion = storage_io.ReadValue[celler.ChunkVersion](r)
	s.MetaData = storage_io.ReadMap[string, string, map[string]string](r)
	s.BlockIDStorage.Load(r)
	s.BlockAttachStorage.Load(r)
	s.BlockSNBTStorage.Load(r)
}

func MakeSimpleSubChunk[
	BlockIDT ~uint32, AttachT ~uint32,
	BlockStorageT interface {
		celler.Storage[BlockIDT]
		celler.In
		celler.Out
	},
	AttachStorageT interface {
		celler.Storage[AttachT]
		celler.In
		celler.Out
	},
	SNBTStorageT interface {
		celler.Storage[celler.SNBT]
		celler.In
		celler.Out
	},
](
	CurrentVersion celler.ChunkVersion,
	MetaData map[string]string,
	BlockIDStorage BlockStorageT,
	BlockAttachStorage AttachStorageT,
	BlockSNBTStorage SNBTStorageT,
) *Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT] {
	if MetaData == nil {
		MetaData = make(map[string]string)
	}
	return &Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]{
		CurrentVersion:     CurrentVersion,
		MetaData:           MetaData,
		BlockIDStorage:     BlockIDStorage,
		BlockAttachStorage: BlockAttachStorage,
		BlockSNBTStorage:   BlockSNBTStorage,
	}
}
