// Package layer_storage stores 16×16×16 comparable values in palette, flat,
// sparse or uniform form. Flat indexes use X-Z-Y order: (x<<8)|(z<<4)|y.
package layer_storage

// Position is a coordinate within one 16³ layer.
type Position struct{ X, Y, Z uint8 }

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// At returns the local position, including for negative world coordinates.
func At[X, Y, Z integer](x X, y Y, z Z) Position {
	return Position{X: uint8(x & 15), Y: uint8(y & 15), Z: uint8(z & 15)}
}

type Read[T any] interface {
	Default() T
	IterNonDefault(func(Position, T) bool)
}
type Write[T any] interface {
	InitWithDefault(T)
	CopyFrom(Read[T])
}
type RandomRead[T any] interface {
	Read[T]
	Get(Position) T
}
type Storage[T any] interface {
	RandomRead[T]
	Write[T]
	Set(Position, T)
}
type StorageMaker[T any, S Storage[T]] interface{ Make() S }
