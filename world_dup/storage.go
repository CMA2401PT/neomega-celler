package world

import "celler"

type ChunkedStorage[S any] struct {
	storage map[celler.ChunkPos]S
}

func (s ChunkedStorage[S]) Set(pos celler.ChunkPos, data S) {
	s.storage[pos] = data
}
func (s ChunkedStorage[S]) Iter(fn func(pos celler.ChunkPos, data S) (next bool)) {
	for pos, d := range s.storage {
		if !fn(pos, d) {
			return
		}
	}
}

func (s ChunkedStorage[S]) Filter(center celler.CenteredArea) {
	ns := ChunkedStorage[S]{
		storage: map[celler.ChunkPos]S{},
	}
	for pos, data := range s.Iter {
		if center.Has(pos.X, pos.Z) {
			ns.Set(pos, data)
		}
	}
	s.storage = ns.storage
}

func (s ChunkedStorage[S]) Get(pos celler.ChunkPos) (S, bool) {
	v, found := s.storage[pos]
	return v, found
}

type SubChunkChunkStorage[S any] struct {
	posFlag, negFlag uint32
	pos              []S
	neg              []S
}

func (s SubChunkChunkStorage[S]) Iter(fn func(y celler.ChunkUnit, data S) (next bool)) {
	for y := -8; y < 0; y++ {
		ny := -y - 1
		if (s.negFlag & 1 << ny) != 0 {
			if !fn(celler.ChunkUnit(y), s.neg[ny]) {
				return
			}
		}
	}
	for y := 0; y < 24; y++ {
		if (s.posFlag & 1 << y) != 0 {
			if !fn(celler.ChunkUnit(y), s.pos[y]) {
				return
			}
		}
	}
}

func (s SubChunkChunkStorage[S]) Set(y celler.ChunkUnit, data S) {
	if y < 0 {
		ny := int(-y - 1)
		s.negFlag |= 1 << ny
		if len(s.neg) <= ny {
			s.neg = append(s.neg, make([]S, ny-len(s.neg)+1)...)
		}
		s.neg[ny] = data
	} else {
		s.posFlag |= 1 << y
		if len(s.pos) < int(y) {
			s.pos = append(s.pos, make([]S, int(y)-len(s.pos)+1)...)
		}
		s.pos[y] = data
	}
}

func (s SubChunkChunkStorage[S]) Get(y int) (S, bool) {
	if y < 0 {
		ny := -y - 1
		return s.neg[ny], (s.negFlag & (1 << ny)) != 0
	} else {
		s.posFlag |= 1 << y
		if len(s.pos) < y {
			s.pos = append(s.pos, make([]S, y-len(s.pos)+1)...)
		}
		return s.pos[y], (s.posFlag & (1 << y)) != 0
	}
}

type SubChunkStorage[S any] ChunkedStorage[SubChunkChunkStorage[S]]

func (s SubChunkStorage[S]) Filter(center celler.CenteredArea) {
	(ChunkedStorage[SubChunkChunkStorage[S]])(s).Filter(center)
}

func (s SubChunkStorage[S]) Set(pos celler.SubChunkPos, data S) {
	if cs, found := s.storage[pos.ChunkPos]; found {
		cs.Set(pos.Y, data)
	} else {
		var cs SubChunkChunkStorage[S]
		cs.Set(pos.Y, data)
		s.storage[pos.ChunkPos] = cs
	}
}

func (s SubChunkStorage[S]) Get(pos celler.SubChunkPos) (data S, found bool) {
	if cs, found := s.storage[pos.ChunkPos]; !found {
		return data, false
	} else {
		return cs.Get(int(pos.Y))
	}
}

func (s SubChunkStorage[S]) Iter(fn func(pos celler.SubChunkPos, data S) (next bool)) {
	for cpos, d := range s.storage {
		for y, data := range d.Iter {
			if !fn(celler.SubChunkPos{ChunkPos: cpos, Y: y}, data) {
				return
			}
		}
	}
}
