package enhance

import "celler"

type ChangableData[Data any, Change any, ChangePtr *Change] interface {
	*Data
	QueryChange(celler.ChunkVersion) Change
	ApplyChange(Change)
	Version() celler.ChunkVersion
}

type BiDirSyncer[Change any, ChangePtr *Change, Data any, DataPtr ChangableData[Data, Change, ChangePtr]] struct {
	celler.SubChunkStorage[DataPtr]
}

func MakeBiDirSyncer[Change any, ChangePtr *Change, Data any, DataPtr ChangableData[Data, Change, ChangePtr]](storage celler.SubChunkStorage[DataPtr]) BiDirSyncer[Change, ChangePtr, Data, DataPtr] {
	return BiDirSyncer[Change, ChangePtr, Data, DataPtr]{
		SubChunkStorage: storage,
	}
}

func (s BiDirSyncer[Change, ChangePtr, Data, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
	current := s.Get(pos)
	if current == nil {
		if version == 0 {
			return celler.SubChunkChange[Change]{
				Valid: true,
			}
		} else {
			return celler.SubChunkChange[Change]{
				Valid: false,
				Drop:  true,
			}
		}
	}
	if current.Version() == version {
		return celler.SubChunkChange[Change]{
			Valid: true,
		}
	} else {
		change := current.QueryChange(version)
		return celler.SubChunkChange[Change]{
			Valid:  false,
			Drop:   false,
			Change: &change,
		}
	}
}

func (s BiDirSyncer[Change, ChangePtr, Data, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	if change.Drop {
		s.Drop(pos)
		return
	}
	current := s.GetOrCreate(pos)
	current.ApplyChange(*change.Change)
}
