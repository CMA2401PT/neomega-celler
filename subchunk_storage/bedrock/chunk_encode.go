package bedrock

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"encoding/binary"
	"fmt"
	"io"
)

type ChunkEncoder[T ~uint32] struct {
	BlockEncode func(T) ([]byte, error)
}

type Writer interface {
	io.Writer
	io.ByteWriter
}

func (ce *ChunkEncoder[T]) EncodeSubChunkChunkBlocks(buf Writer, Y int32, stor *layer_storage.Palette[T]) {
	buf.WriteByte(9)              // version
	buf.WriteByte(1)              // layer count=1
	buf.WriteByte(uint8(int8(Y))) // y subchunk pos
	// Bedrock disk palettes support 1..6, 8 and 16 bits per index.
	level := stor.Level
	if level == 0 {
		level = 1
	} else if level == 7 {
		level = 8
	} else if level > 8 {
		level = 16
	}
	if level != stor.Level {
		converted := &layer_storage.Palette[T]{}
		converted.Fill(int(level), stor.Default())
		converted.SetPalette(stor.GetUsedPalette())
		for pos, index := range stor.IterNonDefaultPaletteI {
			converted.SetIndex(pos, index)
		}
		stor = converted
	}
	blockSize := uint8(stor.Level)
	blockSize <<= 1
	buf.WriteByte(blockSize)
	byteCount := len(stor.Data) * 4
	data := make([]byte, byteCount)
	for i, v := range stor.Data {
		// Explicitly don't use the binary package to greatly improve performance of writing the Uint32s.
		data[i*4], data[i*4+1], data[i*4+2], data[i*4+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	buf.Write(data)
	ce.encodePalette(buf, stor.GetUsedPalette())
}

func (ce *ChunkEncoder[T]) encodePalette(buf Writer, palette []T) {
	paletteCount := uint32(len(palette))
	binary.Write(buf, binary.LittleEndian, &paletteCount)
	for i := uint32(0); i < paletteCount; i++ {
		rawNbt, err := ce.BlockEncode(palette[i])
		if err != nil {
			fmt.Printf("encode bedrock level db palette error: %v", err)
			// ignore this and continue
		}
		buf.Write(rawNbt)
	}
}

func (p *BedrockStorageReadWrite[T]) saveSubChunkBlocks(position celler.SubChunkPos, chunkBlocks *layer_storage.Palette[T]) {
	key := p.index(position.ChunkPos)
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	p.EncodeSubChunkChunkBlocks(buf, int32(position.Y), chunkBlocks)
	p.RWDB.Put(append(key, keySubChunkData, uint8(int8(position.Y))), buf.Bytes())
}
