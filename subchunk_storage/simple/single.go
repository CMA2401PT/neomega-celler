package simple

import (
	"celler"
)

type Single[BlockIDT ~uint32, AttachT ~uint32, Data celler.SubChunk[BlockIDT, AttachT]] struct {
	lastPos  celler.SubChunkPos
	lastPtr  *Data
	NewEmpty func() Data
}

func NewSingle[BlockIDT ~uint32, AttachT ~uint32, Data celler.SubChunk[BlockIDT, AttachT]](
	makeNew func() Data,
) *Single[BlockIDT, AttachT, Data] {
	return &Single[BlockIDT, AttachT, Data]{
		NewEmpty: makeNew,
	}
}

func (s *Single[BlockIDT, AttachT, Data]) IterAll(cb func(celler.SubChunkPos) (next bool)) {
	if s == nil || s.lastPtr == nil {
		return
	}
	cb(s.lastPos)
}

func (s *Single[BlockIDT, AttachT, Data]) Get(pos celler.SubChunkPos) (d Data) {
	if s == nil {
		return d
	}
	if s.lastPos == pos && s.lastPtr != nil {
		return *s.lastPtr
	}
	return d
}

func (s *Single[BlockIDT, AttachT, Data]) GetOrEmpty(pos celler.SubChunkPos) Data {
	current := s.Get(pos)
	if current.Version() == 0 {
		current = s.NewEmpty()
	}
	return current
}

func (s *Single[BlockIDT, AttachT, Data]) Drop(pos celler.SubChunkPos) {
	if s == nil {
		return
	}
	if s.lastPos == pos {
		s.lastPtr = nil
	}
}

func (s *Single[BlockIDT, AttachT, Data]) Set(pos celler.SubChunkPos, data Data) {
	if data.Version() == 0 {
		s.Drop(pos)
		return
	}
	s.lastPos = pos
	s.lastPtr = &data
}

func (p *Single[BlockIDT, AttachT, Data]) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[BlockIDT, AttachT] {
	return p.Get(position).QueryChange(version)
}

func (p *Single[BlockIDT, AttachT, Data]) ApplyChange(position celler.SubChunkPos, change celler.SubChunkChange[BlockIDT, AttachT]) {
	if change.Valid {
		return
	}
	if change.Drop {
		p.Drop(position)
	}
	init := p.GetOrEmpty(position)
	init.ApplyChange(change)
	p.Set(position, init)
}
