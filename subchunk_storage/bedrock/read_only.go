package bedrock

import (
	"celler"
	"celler/kvdb"
	"celler/layer_storage"
	"celler/subchunk"
	"encoding/binary"
	"sync"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type BedrockStorageReadOnly[T ~uint32] struct {
	ReadOnlyDB                kvdb.ReadOnlyKVDB
	Dim                       int32
	SNBTCache                 *simplelru.LRU[celler.ChunkPos, *chunkSnbtWithChangeMark]
	DecodeSubChunkChunkBlocks func(buf Reader, checkY int32) (*layer_storage.Palette[T], error)
	DefaultBlock              T
	mu                        sync.Mutex
}

func NewReadOnlyBedrockStorage[T ~uint32](
	DB kvdb.ReadOnlyKVDB,
	Dim int32,
	DefaultBlockID T,
	BlockDecode func(nbt []byte) (T, error),
) *BedrockStorageReadOnly[T] {
	lru, _ := simplelru.NewLRU(16, func(key celler.ChunkPos, value *chunkSnbtWithChangeMark) {})
	chunkDecoder := &ChunkDecoder[T]{
		DefaultBlockID: DefaultBlockID,
		BlockDecode:    BlockDecode,
	}
	p := &BedrockStorageReadOnly[T]{
		ReadOnlyDB:                DB,
		Dim:                       Dim,
		DecodeSubChunkChunkBlocks: chunkDecoder.DecodeSubChunkBlocks,
		DefaultBlock:              DefaultBlockID,
		SNBTCache:                 lru,
	}
	return p
}

func (p *BedrockStorageReadOnly[T]) Get(position celler.SubChunkPos) *subchunk.Simple[
	T, uint32,
	*layer_storage.Palette[T],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
] {
	p.mu.Lock()
	defer p.mu.Unlock()
	blocks, err := p.loadSubChunkBlocks(position)
	if err != nil {
		return nil
	}
	if blocks == nil {
		return nil
	}
	snbt := p.loadSubChunkNBT(position)
	return subchunk.MakeSimpleSubChunk(
		1,
		nil,
		blocks,
		layer_storage.MakePlaceHolder(uint32(0)),
		snbt,
	)
}

func (s *BedrockStorageReadOnly[T]) GetOrEmpty(pos celler.SubChunkPos) *subchunk.Simple[
	T, uint32,
	*layer_storage.Palette[T],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
] {
	current := s.Get(pos)
	if current.Version() == 0 {
		current = &subchunk.Simple[
			T, uint32,
			*layer_storage.Palette[T],
			*layer_storage.PlaceHolder[uint32],
			*layer_storage.Map[celler.SNBT]]{
			CurrentVersion: 0, MetaData: nil,
			BlockIDStorage:     layer_storage.New(s.DefaultBlock).Palette(),
			BlockAttachStorage: layer_storage.New(uint32(0)).PlaceHolder(),
			BlockSNBTStorage:   layer_storage.New(celler.SNBT("")).Map(),
		}
	}
	return current
}

func (p *BedrockStorageReadOnly[T]) IterAll(cb func(celler.SubChunkPos) (next bool)) {
	for kv := range p.ReadOnlyDB.IterAll {
		key := kv.Key()
		if p.Dim == 0 {
			if len(key) != (8+2) || key[8] != '/' {
				continue
			}
		} else {
			if len(key) != (12+2) || key[12] != '/' {
				continue
			}
			dm := binary.LittleEndian.Uint32(key[8:12])
			if dm != uint32(p.Dim) {
				continue
			}
		}
		x := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key)))
		z := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key[4:])))
		var y celler.ChunkUnit
		if p.Dim == 0 {
			y = celler.ChunkUnit(int8(key[9]))
		} else {
			y = celler.ChunkUnit(int8(key[13]))
		}
		pos := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: x, Z: z}, Y: y}
		if !cb(pos) {
			break
		}
	}
}

func (p *BedrockStorageReadOnly[T]) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[T, uint32] {
	return p.Get(position).QueryChange(version)
}
