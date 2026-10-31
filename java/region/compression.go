package region

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

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

// Compress produces a zlib record. Its timestamp is supplied by the caller.
func Compress(data []byte, timestamp uint32) (Record, error) {
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		return Record{}, err
	}
	if err := writer.Close(); err != nil {
		return Record{}, err
	}
	return Record{Compression: Zlib, Data: buffer.Bytes(), Timestamp: timestamp}, nil
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
