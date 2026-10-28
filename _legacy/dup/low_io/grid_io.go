package low_io

type GridPos struct {
	X, Y, Z int32
}

// 我们不仅需要考虑写放大问题以优化写操作，
// 还要考虑多个 Grids 共用一片存储的问题
type GridStorage interface {
	DataWriter(GridPos) SubChunkDataWriteCloser
	DataReader(GridPos) SubChunkDataReadCloser
	IterGrid(func(GridPos) (next bool))
	DropGrid(GridPos)
}
