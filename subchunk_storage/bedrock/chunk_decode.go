package bedrock

import (
	"bytes"
	"celler"
	"celler/internal/nbtutil"
	"celler/layer_storage"
	"encoding/binary"
	"fmt"
	"io"
	nbt "neomega-nbt/presets/v1"

	"github.com/df-mc/goleveldb/leveldb"
)

type ChunkDecoder[T ~uint32] struct {
	DefaultBlockID T
	BlockDecode    func(nbt []byte) (T, error)
}

type Reader interface {
	io.Reader
	io.ByteReader
}

func (cd *ChunkDecoder[T]) DecodeSubChunkBlocks(buf Reader, checkY int32) (p *layer_storage.Palette[T], err error) {
	ver, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("error reading version: %w", err)
	}
	switch ver {
	default:
		return nil, fmt.Errorf("unknown sub chunk version %v: can't decode", ver)
	case 1:
		// Version 1 only has one layer for each sub chunk, but uses the format with palettes.
		storage, err := cd.decodePalettedStorage(buf)
		if err != nil {
			return nil, err
		}
		return storage, nil
	case 8, 9:
		// Version 8 allows up to 256 layers for one sub chunk.
		storageCount, err := buf.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("error reading storage count: %w", err)
		}
		if ver == 9 {
			uIndex, err := buf.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("error reading subchunk index: %w", err)
			}
			// The index as written here isn't the actual index of the subchunk within the chunk. Rather, it is the Y
			// value of the subchunk. This means that we need to translate it to an index.
			if int8(uIndex) != int8(checkY) {
				return nil, fmt.Errorf("subchunk Y mismatch %v!=%v", int8(uIndex), int8(checkY))
			}
		}
		if storageCount == 0 {
			return nil, nil
		}
		p, err = cd.decodePalettedStorage(buf)
		if err != nil {
			return nil, err
		}
		// drop all non 0 layers
		for i := byte(1); i < storageCount; i++ {
			_, err = cd.decodePalettedStorage(buf)
			if err != nil {
				return nil, err
			}
		}
		return p, nil
	}
}

func (cd *ChunkDecoder[T]) decodePalettedStorage(buf Reader) (*layer_storage.Palette[T], error) {
	blockSize, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("error reading block size: %w", err)
	}
	if blockSize&1 != 0 {
		return nil, fmt.Errorf("runtime-ID palettes are not disk palettes")
	}
	blockSize >>= 1
	if blockSize == 0x7f {
		return nil, nil
	}

	switch blockSize {
	case 0, 1, 2, 3, 4, 5, 6, 8, 16:
	default:
		return nil, fmt.Errorf("unsupported palette bit width %d", blockSize)
	}
	stor := &layer_storage.Palette[T]{}
	stor.Fill(int(blockSize), cd.DefaultBlockID)

	byteCount := len(stor.Data) * 4
	data := make([]byte, byteCount)
	readCount, err := io.ReadFull(buf, data)
	if err != nil {
		return nil, err
	}
	if readCount != byteCount {
		return nil, fmt.Errorf("cannot read paletted storage (size=%v) not enough block data present: expected %v bytes, got %v", blockSize, byteCount, len(data))
	}
	for i := 0; i < len(stor.Data); i++ {
		// Explicitly don't use the binary package to greatly improve performance of reading the Uint32s.
		stor.Data[i] = uint32(data[i*4]) | uint32(data[i*4+1])<<8 | uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
	}
	if stor.Level == 0 {
		value, err := cd.decodePaletteBlock(buf)
		if err != nil {
			return nil, err
		}
		stor.SetPalette([]T{value})
	} else {
		palette, err := cd.decodePalette(buf)
		if err != nil {
			return nil, err
		}
		stor.SetPalette(palette)
	}

	return stor, nil
}

func (cd *ChunkDecoder[T]) decodePalette(buf Reader) ([]T, error) {
	paletteCount := uint32(1)
	if err := binary.Read(buf, binary.LittleEndian, &paletteCount); err != nil {
		return nil, fmt.Errorf("error reading palette entry count: %w", err)
	}
	var err error
	palette := make([]T, paletteCount)
	for i := uint32(0); i < paletteCount; i++ {
		palette[i], err = cd.decodePaletteBlock(buf)
		if err != nil {
			return nil, fmt.Errorf("palette entry %d: %w", i, err)
		}
	}
	return palette, nil
}

type ByPassReader struct {
	in Reader
	r  *bytes.Buffer
}

func (r *ByPassReader) Read(p []byte) (n int, err error) {
	n, err = r.in.Read(p)
	r.r.Write(p[:n])
	return n, err
}

func (r *ByPassReader) ReadByte() (b byte, err error) {
	b, err = r.in.ReadByte()
	if err != nil {
		return b, err
	}
	r.r.WriteByte(b)
	return b, err
}

func (cd *ChunkDecoder[T]) decodePaletteBlock(buf Reader) (rtid T, err error) {
	bp := &ByPassReader{in: buf, r: bytes.NewBuffer(nil)}
	_, err = nbtutil.Read(bp, nbt.EncodingLittleEndian)
	if err != nil {
		return 0, fmt.Errorf("error decoding block palette entry: %w", err)
	}
	return cd.BlockDecode(bp.r.Bytes())
}

func (p *BedrockStorageReadOnly[T]) loadSubChunkBlocks(position celler.SubChunkPos) (chunkBlocks *layer_storage.Palette[T], err error) {
	key := p.index(position.ChunkPos)
	data, err := p.ReadOnlyDB.Get(append(key, keySubChunkData, uint8(position.Y)))

	if err == leveldb.ErrNotFound {
		return chunkBlocks, nil
	}
	if err != nil {
		return chunkBlocks, err
	}
	return p.DecodeSubChunkChunkBlocks(bytes.NewReader(data), int32(position.Y))
}
