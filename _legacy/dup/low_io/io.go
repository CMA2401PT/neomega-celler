package low_io

import (
	"celler"
	"io"
)

// 我们需要考虑写放大问题以优化写操作
type SubChunkDataWriter interface {
	CurrentVersion() map[celler.SubChunkPos]celler.ChunkVersion
	GetWriter(celler.SubChunkPos, celler.ChunkVersion) io.WriteCloser
	Drop(celler.SubChunkPos)
}

type SubChunkDataWriteCloser interface {
	SubChunkDataWriter
	io.Closer
}

type SubChunkDataReader interface {
	CurrentVersion() map[celler.SubChunkPos]celler.ChunkVersion
	GetReader(celler.SubChunkPos) io.ReadCloser
}

type SubChunkDataReadCloser interface {
	SubChunkDataReader
	io.Closer
}

type SubChunkDataIO interface {
	SubChunkDataWriteCloser
	SubChunkDataReadCloser
}
