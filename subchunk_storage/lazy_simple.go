package subchunk_storage

import (
	"celler"
	"celler/subchunk_storage/low_io"
)

type BasicSubchunkWithIO[Data any] interface {
	*Data
	Version() celler.ChunkVersion
	celler.In
	celler.Out
}

type LazySimple[Data any, DataPtr BasicSubchunkWithIO[Data]] struct {
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
) *LazySimple[Data, DataPtr] {
	return &LazySimple[Data, DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
	}
}

func (s *LazySimple[Data, DataPtr]) IsEmpty() bool {
	return len(s.Data) == 0 && len(s.LazyVersions) == 0
}

func (s *LazySimple[Data, DataPtr]) LoadData(reader low_io.SubChunkDataReadCloser) {
	s.LazyVersions = reader.CurrentVersion()
	s.LazyReader = reader
	s.DropData = make(map[celler.SubChunkPos]struct{})
}

func (s *LazySimple[Data, DataPtr]) DumpData(writer low_io.SubChunkDataWriteCloser) {
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

func (s *LazySimple[Data, DataPtr]) InsightFn() func(celler.SubChunkPos) (hash uint64) {
	return nil
}

func (s *LazySimple[Data, DataPtr]) loadSubchunk(p celler.SubChunkPos) DataPtr {
	d := s.NewEmpty()
	rd := s.LazyReader.GetReader(p)
	d.Load(rd)
	rd.Close()
	return d

}

func (s *LazySimple[Data, DataPtr]) ensureLoaded(pos celler.SubChunkPos) (ptr DataPtr) {
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

func (s *LazySimple[Data, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
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

func (s *LazySimple[Data, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
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

func (s *LazySimple[Data, DataPtr]) GetOrCreate(pos celler.SubChunkPos) DataPtr {
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		s.Set(pos, current)
	}
	return current
}

func (s *LazySimple[Data, DataPtr]) Drop(pos celler.SubChunkPos) {
	if s.LazyVersions != nil {
		s.DropData[pos] = struct{}{}
	}
	if s.Data == nil {
		return
	}
	delete(s.Data, pos)
	if s.lastPos == pos {
		s.lastPtr = nil
	}
}
func (s *LazySimple[Data, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	if data == nil {
		s.Drop(pos)
		return
	}
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = data
	if s.Data == nil {
		s.Data = make(map[celler.SubChunkPos]DataPtr)
	}
	if s.LazyVersions != nil {
		delete(s.DropData, pos)
	}
	s.Data[pos] = data
}

func (s *LazySimple[Data, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
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
