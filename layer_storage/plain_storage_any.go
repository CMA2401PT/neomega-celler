package layer_storage

import ()

type Plain[T comparable] struct {
	DefaultValue T
	Data         []T
}

func (Plain[T]) Make() *Plain[T] {
	return &Plain[T]{}
}

func (s *Plain[T]) posToIdx(pos Position) int {
	return (int(pos.X&0xf) << 8) | (int(pos.Z&0xf) << 4) | int(pos.Y&0xf)
}

func (s *Plain[T]) idxToPos(i int) Position {
	return Position{X: uint8(i >> 8 & 0xf), Y: uint8(i & 0xf), Z: uint8(i >> 4 & 0xf)}
}

func (s *Plain[T]) Get(pos Position) T {
	return s.Data[s.posToIdx(pos)]
}

func (s *Plain[T]) IterNonDefault(fn func(Position, T) (next bool)) {
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

func (s *Plain[T]) Set(pos Position, b T) {
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

func (s *Plain[T]) CopyFrom(src Read[T]) {
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
		defaultV := src.Default()
		s.InitWithDefault(defaultV)
		for p, v := range src.IterNonDefault {
			s.Set(p, v)
		}
	}
}

func MakePlainStorage[T comparable](defaultV T) *Plain[T] {
	storage := &Plain[T]{}
	storage.InitWithDefault(defaultV)
	return storage
}
