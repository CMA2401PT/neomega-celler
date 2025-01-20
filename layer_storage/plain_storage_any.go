package layer_storage

import (
	"celler"
	"celler/storage_io"
	"fmt"
	"io"
)

type Plain[T storage_io.Number | ~string] struct {
	DefaultValue T
	Data         []T
}

func (Plain[T]) Make() *Plain[T] {
	return &Plain[T]{}
}

func (s *Plain[T]) posToIdx(pos celler.SubChunkBlockPos) int {
	return (int(pos.X&0xf) << 8) | (int(pos.Z&0xf) << 4) | int(pos.Y&0xf)
}

func (s *Plain[T]) idxToPos(i int) celler.SubChunkBlockPos {
	return celler.SubChunkBlockPos{X: uint8(i >> 8 & 0xf), Y: uint8(i & 0xf), Z: uint8(i >> 4 & 0xf)}
}

func (s *Plain[T]) Get(pos celler.SubChunkBlockPos) T {
	return s.Data[s.posToIdx(pos)]
}

func (s *Plain[T]) IterNonDefault(fn func(celler.SubChunkBlockPos, T) (next bool)) {
	if s.Data == nil {
		return
	}
	for i, v := range s.Data {
		if v == s.DefaultValue {
			continue
		}
		if !fn(s.idxToPos(i), v) {
			return
		}
	}
}

func (s *Plain[T]) Default() T {
	return s.DefaultValue
}

func (s *Plain[T]) InitWithDefault(data T) {
	s.DefaultValue = data
	if s.Data == nil {
		s.Data = make([]T, 4096)
	}
	for i := 0; i < 4096; i++ {
		s.Data[i] = data
	}
}

func (s *Plain[T]) Set(pos celler.SubChunkBlockPos, b T) {
	s.Data[s.posToIdx(pos)] = b
}

func (s *Plain[T]) Same() (d T, allSame bool) {
	first := s.Data[0]
	for i := 1; i < 4096; i++ {
		if first != s.Data[i] {
			return d, false
		}
	}
	return first, true
}

func (s *Plain[T]) CopyFrom(src celler.LayerStorageRead[T]) {
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
	if plainSrc, ok := src.(*Plain[T]); ok {
		s.DefaultValue = plainSrc.Default()
		if s.Data == nil {
			s.Data = make([]T, 4096)
		}
		copy(s.Data, plainSrc.Data)
		return
	}
	if mapSrc, ok := src.(*Map[T]); ok {
		defaultV := mapSrc.Default()
		s.InitWithDefault(defaultV)
		for p, v := range mapSrc.IterNonDefault {
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
	}
}

func MakePlainStorage[T storage_io.Number | ~string](defaultV T) *Plain[T] {
	storage := &Plain[T]{}
	storage.InitWithDefault(defaultV)
	return storage
}

func (s *Plain[T]) Dump(w io.Writer) {
	storage_io.WriteValue(w, s.DefaultValue)
	storage_io.WriteArrayWithoutCount(w, s.Data)
}

func (s *Plain[B]) Load(r io.Reader) {
	s.DefaultValue = storage_io.ReadValue[B](r)
	s.Data = storage_io.ReadArrayWithoutCount[B](r, 4096)
}
