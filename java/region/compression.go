package region

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/pierrec/lz4/v4"
)

const (
	Gzip         byte = 1
	Zlib         byte = 2
	Uncompressed byte = 3
	LZ4          byte = 4
)

func (r Record) Decode() ([]byte, error) {
	var reader io.ReadCloser
	var err error
	switch r.Compression {
	case Uncompressed:
		return bytes.Clone(r.Data), nil
	case Gzip:
		reader, err = gzip.NewReader(bytes.NewReader(r.Data))
	case Zlib:
		reader, err = zlib.NewReader(bytes.NewReader(r.Data))
	case LZ4:
		return decodeLZ4(r.Data)
	default:
		return nil, fmt.Errorf("unsupported region compression %d", r.Compression)
	}
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// compressionBuffer 是池里复用的编码器与输出缓冲；Bytes 由 Compress 复制后交出去。
type compressionBuffer struct {
	bytes.Buffer
	writer *zlib.Writer
}

// 每次压缩独占一个编码器；复用其字典与工作数组，避免逐区块重新分配。
var compressors = sync.Pool{New: func() any {
	buffer := &compressionBuffer{}
	buffer.writer = zlib.NewWriter(&buffer.Buffer)
	return buffer
}}

// Compress 生成独立持有载荷的 zlib 记录，可并行调用；时间戳由调用方提供。
func Compress(data []byte, timestamp uint32) (Record, error) {
	buffer := compressors.Get().(*compressionBuffer)
	defer compressors.Put(buffer)
	buffer.Buffer.Reset()
	buffer.writer.Reset(&buffer.Buffer)
	if _, err := buffer.writer.Write(data); err != nil {
		return Record{}, err
	}
	if err := buffer.writer.Close(); err != nil {
		return Record{}, err
	}
	return Record{Compression: Zlib, Data: bytes.Clone(buffer.Bytes()), Timestamp: timestamp}, nil
}

// Java uses the LZ4Block stream framing, rather than the LZ4 frame format.
func decodeLZ4(data []byte) ([]byte, error) {
	var out []byte
	for {
		if len(data) < 21 {
			return nil, io.ErrUnexpectedEOF
		}
		if string(data[:8]) != "LZ4Block" {
			return nil, fmt.Errorf("invalid LZ4Block magic")
		}
		token := data[8]
		packed := int(binary.LittleEndian.Uint32(data[9:]))
		size := int(binary.LittleEndian.Uint32(data[13:]))
		data = data[21:]
		if packed == 0 && size == 0 {
			if len(data) != 0 {
				return nil, fmt.Errorf("trailing LZ4 data")
			}
			return out, nil
		}
		if size <= 0 || size > 1<<(10+(token&15)) || packed <= 0 || packed > len(data) {
			return nil, fmt.Errorf("invalid LZ4 block lengths")
		}
		block := data[:packed]
		data = data[packed:]
		switch token & 0xf0 {
		case 0x10:
			if packed != size {
				return nil, fmt.Errorf("invalid raw LZ4 block")
			}
			out = append(out, block...)
		case 0x20:
			decoded := make([]byte, size)
			n, err := lz4.UncompressBlock(block, decoded)
			if err != nil {
				return nil, err
			}
			if n != size {
				return nil, fmt.Errorf("LZ4 length mismatch")
			}
			out = append(out, decoded...)
		default:
			return nil, fmt.Errorf("invalid LZ4 block method")
		}
	}
}
