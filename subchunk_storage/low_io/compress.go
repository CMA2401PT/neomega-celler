package low_io

type Compressor interface {
	Compress(data []byte) []byte
	Decompress(data []byte) []byte
}

type NoCompress struct {
}

func (NoCompress) Compress(data []byte) []byte {
	return data
}

func (NoCompress) Decompress(data []byte) []byte {
	return data
}
