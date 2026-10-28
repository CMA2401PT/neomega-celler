package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/java/mca"

	"sync"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type ChunkWithChangeMark struct {
	Subchunks map[celler.ChunkUnit]*subchunk.Simple[
		uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.PlaceHolder[uint32],
		*layer_storage.Map[celler.SNBT],
	]
	Changed bool
}

type JavaStorageReadOnly struct {
	cache simplelru.LRUCache[
		celler.ChunkPos,
		*ChunkWithChangeMark,
	]
	rodb mca.ReadOnlyMCAChunkKV
	mu   sync.RWMutex
}

func NewReadOnlyJavaStorage(
	rodb mca.ReadOnlyMCAChunkKV,
) *JavaStorageReadOnly {
	cache, _ := simplelru.NewLRU(64, func(celler.ChunkPos, *ChunkWithChangeMark) {})
	return &JavaStorageReadOnly{
		cache: cache,
		rodb:  rodb,
		mu:    sync.RWMutex{},
	}
}

func (s *JavaStorageReadOnly) loadChunk(position celler.ChunkPos) (chunk *ChunkWithChangeMark) {
	var exist bool
	chunk, exist = s.cache.Get(position)
	if exist {
		return chunk
	}
	defer func() {
		s.cache.Add(position, chunk)
	}()
	data, _ := s.rodb.Get(position)
	if len(data) == 0 {
		return nil
	}

	chunk, _ = cd.DecodeChunkFromData(data)
	return chunk
}

func (s *JavaStorageReadOnly) loadOrCreateChunk(position celler.ChunkPos) (chunk *ChunkWithChangeMark) {
	var exist bool
	chunk, exist = s.cache.Get(position)
	if exist && chunk != nil {
		return chunk
	}
	defer func() {
		s.cache.Add(position, chunk)
	}()
	data, _ := s.rodb.Get(position)
	if len(data) == 0 {
		chunk = &ChunkWithChangeMark{
			Subchunks: map[celler.ChunkUnit]*subchunk.Simple[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.PlaceHolder[uint32], *layer_storage.Map[celler.SNBT]]{},
			Changed:   true,
		}
		return chunk
	}
	chunk, _ = cd.DecodeChunkFromData(data)
	return chunk
}

func (s *JavaStorageReadOnly) Get(position celler.SubChunkPos) *subchunk.Simple[
	uint32, uint32,
	*layer_storage.Palette[uint32],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
] {
	s.mu.Lock()
	defer s.mu.Unlock()
	chunk := s.loadChunk(position.ChunkPos)
	if chunk == nil {
		return nil
	}
	return chunk.Subchunks[position.Y]
}

func (s *JavaStorageReadOnly) Close() error {
	return nil
}

func (s *JavaStorageReadOnly) IterAll(cb func(celler.SubChunkPos) (next bool)) {
	for chunkPos := range s.rodb.IterAll {
		for y := celler.ChunkUnit(-4); y < celler.ChunkUnit(20); y++ {
			if !cb(celler.SubChunkPos{
				ChunkPos: chunkPos,
				Y:        y,
			}) {
				return
			}
		}
	}
}

func (s *JavaStorageReadOnly) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[uint32, uint32] {
	return s.Get(position).QueryChange(version)
}
