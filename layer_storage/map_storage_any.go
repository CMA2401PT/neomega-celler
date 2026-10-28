package layer_storage

import (
	"celler"
	"celler/storage_io"
	"fmt"
	"io"
)

type Map[T storage_io.Number | ~string] struct {
	DefaultValue T
	Data         map[uint16]T
}

func (Map[T]) Make() *Map[T] {
	return &Map[T]{}
}

func (s *Map[T]) posToIdx(pos celler.SubChunkBlockPos) uint16 {
	return (uint16(pos.X&0xf) << 8) | (uint16(pos.Z&0xf) << 4) | uint16(pos.Y&0xf)
}

func (s *Map[T]) idxToPos(i uint16) celler.SubChunkBlockPos {
	return celler.ToSubChunkBlockPos(i>>8&0xf, i&0xf, i>>4&0xf)
}

func (s *Map[T]) IterNonDefault(fn func(celler.SubChunkBlockPos, T) (next bool)) {
	if s.Data == nil {
		return
	}
	for i, v := range s.Data {
		if !fn(s.idxToPos(i), v) {
			return
		}
	}
}

func (s *Map[T]) Get(pos celler.SubChunkBlockPos) T {
	if value, ok := s.Data[s.posToIdx(pos)]; ok {
		return value
	}
	return s.DefaultValue
}

func (s *Map[T]) Default() T {
	return s.DefaultValue
}

func (s *Map[T]) InitWithDefault(data T) {
	s.DefaultValue = data
	s.Data = map[uint16]T{}
}

func (s *Map[T]) Set(pos celler.SubChunkBlockPos, b T) {
	if b == s.DefaultValue {
		if s.Data != nil {
			delete(s.Data, s.posToIdx(pos))
		}
		return
	}
	if s.Data == nil {
		s.Data = map[uint16]T{}
	}
	s.Data[s.posToIdx(pos)] = b
}

func (s *Map[T]) Same() (d T, allSame bool) {
	if s.Data == nil || len(s.Data) == 0 {
		return s.DefaultValue, true
	}
	// Missing entries still contain DefaultValue. A partial map with
	// non-default entries therefore cannot describe a uniform layer.
	if len(s.Data) < 4096 {
		for _, v := range s.Data {
			if v != s.DefaultValue {
				return d, false
			}
		}
		return s.DefaultValue, true
	}
	var firstV T
	for _, v := range s.Data {
		firstV = v
		break
	}
	for _, v := range s.Data {
		if v != firstV {
			return d, false
		}
	}
	return firstV, true
}

func (s *Map[T]) CopyFrom(src celler.LayerStorageRead[T]) {
	if placeHolderSrc, ok := src.(*PlaceHolder[T]); ok {
		s.InitWithDefault(placeHolderSrc.Default())
		return
	}
	if paletteSrc, ok := src.(*Palette[T]); ok {
		defaultV := paletteSrc.Default()
		s.InitWithDefault(defaultV)
		for p, v := range paletteSrc.IterNonDefault {
			s.Data[s.posToIdx(p)] = v
		}
		return
	}
	if mapSrc, ok := src.(*Map[T]); ok {
		defaultV := mapSrc.Default()
		s.InitWithDefault(defaultV)
		s.Data = make(map[uint16]T)
		for p, v := range mapSrc.Data {
			s.Data[p] = v
		}
		return
	}
	if plainSrc, ok := src.(*Plain[T]); ok {
		defaultV := src.Default()
		s.InitWithDefault(defaultV)
		for p, v := range plainSrc.IterNonDefault {
			s.Data[s.posToIdx(p)] = v
		}
		return
	}
	{
		fmt.Printf("slow copy from: %T\n", src)
		defaultV := src.Default()
		s.InitWithDefault(defaultV)
		for p, v := range src.IterNonDefault {
			s.Set(p, v)
		}
		return
	}
}

func MakeMapStorage[T storage_io.Number | ~string](defaultV T) *Map[T] {
	storage := &Map[T]{}
	storage.InitWithDefault(defaultV)
	return storage
}

func (s *Map[T]) Dump(w io.Writer) {
	storage_io.WriteValue(w, s.DefaultValue)
	storage_io.WriteMap(w, s.Data, true)
}

func (s *Map[B]) Load(r io.Reader) {
	s.DefaultValue = storage_io.ReadValue[B](r)
	s.Data = storage_io.ReadMap[uint16, B, map[uint16]B](r)
}
