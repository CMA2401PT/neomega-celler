package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/java/mca"
	"neomega_blocks"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type JavaStorageReadWrite struct {
	*JavaStorageReadOnly
	rwdb mca.ReadWriteMCAChunkKV
}

func (s *JavaStorageReadWrite) Close() error {
	ks := s.cache.Keys()
	for _, p := range ks {
		c, _ := s.cache.Get(p)
		if c == nil {
			continue
		}
		if !c.Changed {
			continue
		}
		data := chunkEncode.EncodeChunk(p, c)
		s.rwdb.Put(p, data)
	}
	return nil
}

func (s *JavaStorageReadWrite) Drop(position celler.SubChunkPos) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cm := s.loadOrCreateChunk(position.ChunkPos)
	cm.Changed = true
	delete(cm.Subchunks, position.Y)
}

func (s *JavaStorageReadWrite) Set(position celler.SubChunkPos, data *subchunk.Simple[
	neomega_blocks.UIBI, uint32,
	*layer_storage.Palette[neomega_blocks.UIBI],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
]) {
	if data.Version() == 0 {
		s.Drop(position)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cm := s.loadOrCreateChunk(position.ChunkPos)
	cm.Changed = true
	cm.Subchunks[position.Y] = data
}

func NewReadWriteJavaStorage(
	rwdb mca.ReadWriteMCAChunkKV,
) *JavaStorageReadWrite {
	cache, _ := simplelru.NewLRU(64, func(cp celler.ChunkPos, c *ChunkWithChangeMark) {
		if c == nil {
			return
		}
		if !c.Changed {
			return
		}
		data := chunkEncode.EncodeChunk(cp, c)
		rwdb.Put(cp, data)
	})
	s := &JavaStorageReadWrite{
		JavaStorageReadOnly: NewReadOnlyJavaStorage(rwdb),
		rwdb:                rwdb,
	}
	s.cache = cache
	return s
}
