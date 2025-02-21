package simple

import (
	"celler"
)

type SimpleMemory[BlockIDT ~uint32, AttachT ~uint32, Data celler.SubChunk[BlockIDT, AttachT]] struct {
	Data     map[celler.SubChunkPos]Data
	lastPos  celler.SubChunkPos
	lastPtr  *Data
	NewEmpty func() Data
}

func NewSimpleMemory[BlockIDT ~uint32, AttachT ~uint32, Data celler.SubChunk[BlockIDT, AttachT]](
	makeNew func() Data,
) *SimpleMemory[BlockIDT, AttachT, Data] {
	return &SimpleMemory[BlockIDT, AttachT, Data]{
		NewEmpty: makeNew,
	}
}

func (s *SimpleMemory[BlockIDT, AttachT, Data]) IterAll(cb func(celler.SubChunkPos) (next bool)) {
	if s == nil || s.Data == nil {
		return
	}
	for p := range s.Data {
		if !cb(p) {
			return
		}
	}
}

func (s *SimpleMemory[BlockIDT, AttachT, Data]) Get(pos celler.SubChunkPos) (d Data) {
	if s == nil {
		return d
	}
	if s.lastPos == pos && s.lastPtr != nil {
		return *s.lastPtr
	}
	s.lastPos = pos
	d = s.Data[pos]
	s.lastPtr = &d
	return d
}

func (s *SimpleMemory[BlockIDT, AttachT, Data]) GetOrEmpty(pos celler.SubChunkPos) Data {
	current := s.Get(pos)
	if current.Version() == 0 {
		current = s.NewEmpty()
	}
	return current
}

func (s *SimpleMemory[BlockIDT, AttachT, Data]) Drop(pos celler.SubChunkPos) {
	if s == nil || s.Data == nil {
		return
	}
	delete(s.Data, pos)
	if s.lastPos == pos {
		s.lastPtr = nil
	}
}

func (s *SimpleMemory[BlockIDT, AttachT, Data]) Set(pos celler.SubChunkPos, data Data) {
	if data.Version() == 0 {
		s.Drop(pos)
		return
	}
	s.lastPos = pos
	s.lastPtr = &data
	if s.Data == nil {
		s.Data = make(map[celler.SubChunkPos]Data)
	}
	s.Data[pos] = data
}

func (p *SimpleMemory[BlockIDT, AttachT, Data]) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[BlockIDT, AttachT] {
	return p.Get(position).QueryChange(version)
}

func (p *SimpleMemory[BlockIDT, AttachT, Data]) ApplyChange(position celler.SubChunkPos, change celler.SubChunkChange[BlockIDT, AttachT]) {
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
