package celler

import "io"

type Out interface {
	Dump(io.Writer)
}

type In interface {
	Load(io.Reader)
}
