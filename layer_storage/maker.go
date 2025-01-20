package layer_storage

import "celler/storage_io"

type Maker[T storage_io.Number | ~string] struct {
	defaultV T
}

func (m Maker[T]) PlaceHolder() *PlaceHolder[T] {
	storage := &PlaceHolder[T]{}
	storage.InitWithDefault(m.defaultV)
	return storage
}

func (m Maker[T]) Plain() *Plain[T] {
	storage := &Plain[T]{}
	storage.InitWithDefault(m.defaultV)
	return storage
}

func (m Maker[T]) Palette() *Palette[T] {
	storage := &Palette[T]{}
	storage.InitWithDefault(m.defaultV)
	return storage
}

func (m Maker[T]) Map() *Map[T] {
	storage := &Map[T]{}
	storage.InitWithDefault(m.defaultV)
	return storage
}

func New[T storage_io.Number | ~string](v T) Maker[T] {
	return Maker[T]{v}
}
