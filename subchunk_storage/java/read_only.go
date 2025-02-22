package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/java/mca"
	"neomega_blocks"
	"sync"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type JavaStorageReadOnly struct {
	cache simplelru.LRUCache[celler.ChunkPos, map[celler.ChunkUnit]*subchunk.Simple[
		neomega_blocks.UIBI, uint32,
		*layer_storage.Palette[neomega_blocks.UIBI],
		*layer_storage.PlaceHolder[uint32],
		*layer_storage.Map[celler.SNBT],
	]]
	rodb mca.ReadOnlyMCAChunkKV
	mu   sync.RWMutex
}

func NewReadOnlyJavaStorage(
	rodb mca.ReadOnlyMCAChunkKV,
) *JavaStorageReadOnly {
	cache, _ := simplelru.NewLRU(64, func(celler.ChunkPos, map[celler.ChunkUnit]*subchunk.Simple[
		neomega_blocks.UIBI, uint32,
		*layer_storage.Palette[neomega_blocks.UIBI],
		*layer_storage.PlaceHolder[uint32],
		*layer_storage.Map[celler.SNBT],
	]) {
	})
	return &JavaStorageReadOnly{
		cache: cache,
		rodb:  rodb,
		mu:    sync.RWMutex{},
	}
}

func (s *JavaStorageReadOnly) Get(position celler.SubChunkPos) *subchunk.Simple[
	neomega_blocks.UIBI, uint32,
	*layer_storage.Palette[neomega_blocks.UIBI],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
] {
	s.mu.Lock()
	defer s.mu.Unlock()
	var exist bool
	chunk, exist := s.cache.Get(position.ChunkPos)
	if !exist {
		data, _ := s.rodb.Get(position.ChunkPos)
		if len(data) == 0 {
			return nil
		}

		chunk, _ = cd.DecodeChunkFromData(data)
		s.cache.Add(position.ChunkPos, chunk)
	}
	if chunk == nil {
		return nil
	}
	return chunk[position.Y]
}

func (s *JavaStorageReadOnly) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[neomega_blocks.UIBI, uint32] {
	return s.Get(position).QueryChange(version)
}
