package simple

import (
	"celler"
)

type BasicSubchunk[Data any] interface {
	*Data
	Version() celler.ChunkVersion
}

type Simple[Data any, DataPtr BasicSubchunk[Data]] struct {
	Data     map[celler.SubChunkPos]DataPtr
	hasLast  bool
	lastPos  celler.SubChunkPos
	lastPtr  DataPtr
	NewEmpty func() DataPtr
}

func NewSimple[BlockIDT, AttachT ~uint32, Data any, DataPtr StandardSubChunk[Data, celler.SubChunkDataChange[BlockIDT, AttachT]]](
	bid BlockIDT, attach AttachT,
	makeNew func(BlockIDT, AttachT) DataPtr,
) *Simple[Data, DataPtr] {
	return &Simple[Data, DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
	}
}

func (s *Simple[Data, DataPtr]) IsEmpty() bool {
	return len(s.Data) == 0
}

func (s *Simple[Data, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil || s.Data == nil {
		return
	}
	for p, d := range s.Data {
		if !cb(p, d) {
			return
		}
	}
}

func (s *Simple[Data, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	if s.lastPos == pos && s.hasLast {
		return s.lastPtr
	}
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = s.Data[pos]
	return s.lastPtr
}

func (s *Simple[Data, DataPtr]) GetOrCreate(pos celler.SubChunkPos) DataPtr {
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		s.Set(pos, current)
	}
	return current
}

func (s *Simple[Data, DataPtr]) Drop(pos celler.SubChunkPos) {
	if s.Data == nil {
		return
	}
	delete(s.Data, pos)
	if s.lastPos == pos {
		s.lastPtr = nil
	}
}

func (s *Simple[Data, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
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
	s.Data[pos] = data
}

func (s *Simple[Data, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	current := s.Get(pos)
	if current == nil {
		return 0
	}
	return current.Version()
}
