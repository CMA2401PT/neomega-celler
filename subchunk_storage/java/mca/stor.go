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

type Writer interface {
	io.Writer
	io.WriterAt
	io.Closer
}

type RW interface {
	Reader
	Writer
}

type ReadWriteStor interface {
	IterRegions(func(RegionPos) bool)
	OpenRegion(RegionPos) (RW, error)
	io.Closer
}
