package layer_storage

import (
	"celler"
	"celler/storage_io"
	"io"
)

type PlaceHolder[T storage_io.Number | ~string] struct {
	value T
}

func (PlaceHolder[T]) Make() *PlaceHolder[T] {
	return &PlaceHolder[T]{}
}

func MakePlaceHolder[T storage_io.Number | ~string](defaultV T) *PlaceHolder[T] {
	return &PlaceHolder[T]{value: defaultV}
}

func (s *PlaceHolder[T]) Get(celler.SubChunkBlockPos) T {
	return s.value
}

func (s *PlaceHolder[T]) IterNonDefault(fn func(celler.SubChunkBlockPos, T) (next bool)) {
	return
}

func (s *PlaceHolder[T]) Set(celler.SubChunkBlockPos, T) {
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

func (s *PlaceHolder[T]) Dump(w io.Writer) {
	storage_io.WriteValue(w, s.value)
}

func (s *PlaceHolder[B]) Load(r io.Reader) {
	s.value = storage_io.ReadValue[B](r)
}

func (s *PlaceHolder[B]) CopyFrom(src celler.LayerStorageRead[B]) {
	s.value = src.Default()
}
