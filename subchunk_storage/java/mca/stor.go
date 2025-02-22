package mca

import "io"

type Reader interface {
	io.Reader
	io.ReaderAt
	io.Closer
}

type ReadOnlyStor interface {
	IterRegions(func(RegionPos) bool)
	OpenRegion(RegionPos) (Reader, error)
	io.Closer
}
