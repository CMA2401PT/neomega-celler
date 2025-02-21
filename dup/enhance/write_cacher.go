package enhance

import (
	"celler"
	"sort"
)

type BasicSubchunk[Data any] interface {
	*Data
	Version() celler.ChunkVersion
}

type UnderlayGridedStorage[Data any] interface {
	celler.SubChunkStorage[Data]
	InsightFn(celler.SubChunkPos) (hash uint64)
}

type WriteCacher[Data any, DataPtr BasicSubchunk[Data], Base UnderlayGridedStorage[DataPtr]] struct {
	Data      map[celler.SubChunkPos]DataPtr
	hasLast   bool
	lastPos   celler.SubChunkPos
	lastPtr   DataPtr
	NewEmpty  func() DataPtr
	Base      Base
	MaxCaches int
}

func NewWriteCacher[Data any, DataPtr BasicSubchunk[Data], Base UnderlayGridedStorage[DataPtr]](
	makeNew func() DataPtr,
	base Base,
	maxCaches int,
) *WriteCacher[Data, DataPtr, Base] {
	return &WriteCacher[Data, DataPtr, Base]{
		NewEmpty:  makeNew,
		Base:      base,
		MaxCaches: maxCaches,
	}
}

func (s *WriteCacher[Data, DataPtr, Base]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil || s.Data == nil {
		s.Base.IterSubChunks(cb)
		return
	}
	for p, d := range s.Data {
		if d == nil {
			continue
		}
		if !cb(p, d) {
			return
		}
	}
	for p, d := range s.Base.IterSubChunks {
		if _, found := s.Data[p]; found {
			continue
		}
		if !cb(p, d) {
			return
		}
	}
}

func (s *WriteCacher[Data, DataPtr, Base]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	if s.lastPos == pos && s.hasLast {
		return s.lastPtr
	}
	if cached, found := s.Data[pos]; found {
		s.hasLast = true
		s.lastPos = pos
		s.lastPtr = cached
		return cached
	}
	return s.Base.Get(pos)
}

func (s *WriteCacher[Data, DataPtr, Base]) GetOrCreate(pos celler.SubChunkPos) DataPtr {
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		s.Set(pos, current)
	}
	return current
}

func (s *WriteCacher[Data, DataPtr, Base]) Drop(pos celler.SubChunkPos) {
	s.Set(pos, nil)
}

func (b *WriteCacher[Data, DataPtr, Base]) BatchSet(datas map[celler.SubChunkPos]DataPtr) {
	order := make([]struct {
		Pos  celler.SubChunkPos
		Data DataPtr
	}, 0, len(datas))
	for k, v := range datas {
		order = append(order, struct {
			Pos  celler.SubChunkPos
			Data DataPtr
		}{
			Pos:  k,
			Data: v,
		})
	}
	sort.Slice(order, func(i, j int) bool {
		return b.Base.InsightFn(order[i].Pos) < b.Base.InsightFn(order[j].Pos)
	})
	for _, r := range order {
		b.Base.Set(r.Pos, r.Data)
	}
}

func (b *WriteCacher[Data, DataPtr, Base]) BatchGet(poses []celler.SubChunkPos) (out []struct {
	Pos  celler.SubChunkPos
	Data DataPtr
}) {
	sort.Slice(poses, func(i, j int) bool {
		return b.Base.InsightFn(poses[i]) < b.Base.InsightFn(poses[j])
	})
	out = make([]struct {
		Pos  celler.SubChunkPos
		Data DataPtr
	}, 0, len(poses))
	for _, pos := range poses {
		out = append(out, struct {
			Pos  celler.SubChunkPos
			Data DataPtr
		}{
			Pos:  pos,
			Data: b.Get(pos),
		})
	}
	return out
}

func (s *WriteCacher[Data, DataPtr, Base]) FlushCache() {
	s.BatchSet(s.Data)
	s.Data = nil
	s.hasLast = false
}

func (s *WriteCacher[Data, DataPtr, Base]) Set(pos celler.SubChunkPos, data DataPtr) {
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = data
	if s.Data == nil {
		s.Data = make(map[celler.SubChunkPos]DataPtr)
	}
	s.Data[pos] = data
	if len(s.Data) > s.MaxCaches {
		s.FlushCache()
	}
}

func (s *WriteCacher[Data, DataPtr, Base]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	current := s.Get(pos)
	if current != nil {
		return current.Version()
	}
	return s.Base.GetVersion(pos)
}
