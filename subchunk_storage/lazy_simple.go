package subchunk_storage

import (
	"celler"
	"celler/subchunk_storage/low_io"
)

type LazySimple[Data any, Change any, DataPtr StandardSubChunk[Data, Change]] struct {
	Data         map[celler.SubChunkPos]DataPtr
	hasLast      bool
	lastPos      celler.SubChunkPos
	lastPtr      DataPtr
	NewEmpty     func() DataPtr
	LazyVersions map[celler.SubChunkPos]celler.ChunkVersion
	LazyReader   low_io.SubChunkDataReadCloser
	DropData     map[celler.SubChunkPos]struct{}
}

func NewLazySimple[BlockIDT, AttachT ~uint32, Data any, DataPtr StandardSubChunk[Data, celler.SubChunkDataChange[BlockIDT, AttachT]]](
	bid BlockIDT, attach AttachT,
	makeNew func(BlockIDT, AttachT) DataPtr,
) *LazySimple[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr] {
	return &LazySimple[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
	}
}

func (s *LazySimple[Data, Change, DataPtr]) IsEmpty() bool {
	return len(s.Data) == 0 && len(s.LazyVersions) == 0
}

func (s *LazySimple[Data, Change, DataPtr]) LoadData(reader low_io.SubChunkDataReadCloser) {
	s.LazyVersions = reader.CurrentVersion()
	s.LazyReader = reader
	s.DropData = make(map[celler.SubChunkPos]struct{})
}

func (s *LazySimple[Data, Change, DataPtr]) DumpData(writer low_io.SubChunkDataWriteCloser) {
	defer writer.Close()
	existVersions := writer.CurrentVersion()
	for p := range s.Data {
		d := s.Data[p]
		if d.Version() == existVersions[p] {
			continue
		}
		wt := writer.GetWriter(p, d.Version())
		d.Dump(wt)
		wt.Close()
	}

	for pos := range s.DropData {
		writer.Drop(pos)
	}
}

func (s *LazySimple[Data, Change, DataPtr]) InsightFn() func(celler.ChunkPos) (hash int) {
	return nil
}

func (s *LazySimple[Data, Change, DataPtr]) loadSubchunk(p celler.SubChunkPos) DataPtr {
	d := s.NewEmpty()
	rd := s.LazyReader.GetReader(p)
	d.Load(rd)
	rd.Close()
	return d

}

func (s *LazySimple[Data, Change, DataPtr]) ensureLoaded(pos celler.SubChunkPos) (ptr DataPtr) {
	if ptr, found := s.Data[pos]; found {
		return ptr
	}
	if s.LazyVersions == nil {
		return
	}
	if _, found := s.DropData[pos]; found {
		return nil
	}
	if _, found := s.LazyVersions[pos]; found {
		if s.Data == nil {
			s.Data = make(map[celler.SubChunkPos]DataPtr)
		}
		c := s.loadSubchunk(pos)
		s.Data[pos] = c
		return c
	}
	return nil
}

func (s *LazySimple[Data, Change, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil {
		return
	}
	for p, d := range s.Data {
		if !cb(p, d) {
			return
		}
	}
	for p := range s.LazyVersions {
		if _, found := s.Data[p]; found {
			continue
		}
		if _, found := s.DropData[p]; found {
			continue
		}
		if !cb(p, s.loadSubchunk(p)) {
			return
		}
	}
}

func (s *LazySimple[Data, Change, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	if s.lastPos == pos && s.hasLast {
		return s.lastPtr
	}
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = s.ensureLoaded(pos)
	return s.lastPtr
}

func (s *LazySimple[Data, Change, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = data
	if data == nil {
		if s.Data == nil {
			return
		} else {
			delete(s.Data, pos)
			if s.LazyVersions != nil {
				s.DropData[pos] = struct{}{}
			}
			return
		}
	}
	if s.Data == nil {
		s.Data = make(map[celler.SubChunkPos]DataPtr)
	}
	if s.LazyVersions != nil {
		delete(s.DropData, pos)
	}
	s.Data[pos] = data
}

func (s *LazySimple[Data, Change, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
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

func (s *LazySimple[Data, Change, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	if s == nil {
		return 0
	}
	if s.lastPos == pos && s.hasLast {
		return s.lastPtr.Version()
	}
	if ptr, found := s.Data[pos]; found {
		return ptr.Version()
	}
	if s.LazyVersions == nil {
		return 0
	}
	if _, found := s.DropData[pos]; found {
		return 0
	}
	return s.LazyVersions[pos]
}

func (s *LazySimple[Data, Change, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	if change.Drop {
		if s != nil && s.Data != nil {
			delete(s.Data, pos)
		}
		s.lastPtr = nil
		if s.LazyVersions != nil {
			s.DropData[pos] = struct{}{}
		}
		return
	}
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		if s.Data == nil {
			s.Data = make(map[celler.SubChunkPos]DataPtr)
		}
		s.Data[pos] = current
		if s.LazyVersions != nil {
			delete(s.DropData, pos)
		}
		s.lastPtr = current
	}
	current.ApplyChange(*change.Change)
}
