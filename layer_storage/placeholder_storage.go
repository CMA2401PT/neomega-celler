package layer_storage

type PlaceHolder[T comparable] struct {
	value T
}

func (PlaceHolder[T]) Make() *PlaceHolder[T] {
	return &PlaceHolder[T]{}
}

func MakePlaceHolder[T comparable](defaultV T) *PlaceHolder[T] {
	return &PlaceHolder[T]{value: defaultV}
}

func (s *PlaceHolder[T]) Get(Position) T {
	return s.value
}

func (s *PlaceHolder[T]) IterNonDefault(fn func(Position, T) (next bool)) {
	return
}

func (s *PlaceHolder[T]) Set(Position, T) {
}

func (s *PlaceHolder[T]) Default() T {
	return s.value
}

func (s *PlaceHolder[T]) InitWithDefault(data T) {
	s.value = data
}

func (s *PlaceHolder[T]) Same() (d T, allSame bool) {
	return s.value, true
}

func (s *PlaceHolder[B]) CopyFrom(src Read[B]) {
	s.value = src.Default()
}
