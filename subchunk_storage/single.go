package subchunk_storage

import "celler"

type StandardSubChunk[Data any, Change any] interface {
	*Data
	celler.In
	celler.Out
	Version() celler.ChunkVersion
	QueryChange(celler.ChunkVersion) Change
	ApplyChange(Change)
}

type Single[Data any, Change any, DataPtr StandardSubChunk[Data, Change]] struct {
	Data     DataPtr
	Pos      celler.SubChunkPos
	NewEmpty func() DataPtr
}

func NewSingle[BlockIDT, AttachT ~uint32, Data any, DataPtr StandardSubChunk[Data, celler.SubChunkDataChange[BlockIDT, AttachT]]](
	bid BlockIDT, attach AttachT,
	makeNew func(BlockIDT, AttachT) DataPtr,
) *Single[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr] {
	return &Single[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
	}
}

func (s *Single[Data, Change, DataPtr]) InsightFn() func(celler.ChunkPos) (hash int) {
	return nil
}

func (s *Single[Data, Change, DataPtr]) IsEmpty() bool {
	if s.Data == nil {
		return true
	}
	return false
}

func (s *Single[Data, Change, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil || s.Data == nil {
		return
	}
	cb(s.Pos, s.Data)
}

func (s *Single[Data, Change, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if pos == s.Pos {
		return s.Data
	}
	return nil
}

func (s *Single[Data, Change, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	s.Pos = pos
	s.Data = data
}

func (s *Single[Data, Change, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	current := s.Get(pos)
	if current == nil {
		return 0
	}
	return current.Version()
}

func (s *Single[Data, Change, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
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

func (s *Single[Data, Change, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	if change.Drop {
		s.Data = nil
		return
	}
	s.Pos = pos
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		s.Data = current
	}
	current.ApplyChange(*change.Change)
}
