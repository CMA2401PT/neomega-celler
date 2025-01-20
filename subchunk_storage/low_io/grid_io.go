package low_io

// 我们不仅需要考虑写放大问题以优化写操作，
// 还要考虑多个 Grids 共用一片存储的问题
type GridStorage interface {
	DataWriter(uint64) SubChunkDataWriteCloser
	DataReader(uint64) SubChunkDataReadCloser
	IterGrid(func(uint64) (next bool))
	DropGrid(uint64)
}
