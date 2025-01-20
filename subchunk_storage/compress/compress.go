package compress

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/golang/snappy"
)

type SnappyCompress struct {
}

func (SnappyCompress) Compress(data []byte) []byte {
	return snappy.Encode(nil, data)
}

func (SnappyCompress) Decompress(data []byte) []byte {
	out, _ := snappy.Decode(nil, data)
	return out
}

type ZlibCompress struct {
}

func (ZlibCompress) Compress(data []byte) []byte {
	buf := bytes.NewBuffer(nil)
	w := zlib.NewWriter(buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func (ZlibCompress) Decompress(data []byte) []byte {
	buf := bytes.NewBuffer(data)
	r, _ := zlib.NewReader(buf)
	out, _ := io.ReadAll(r)
	return out
}
