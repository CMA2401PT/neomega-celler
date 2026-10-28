package java

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/java/mca"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type JavaStorageReadWrite struct {
	*JavaStorageReadOnly
	rwdb     mca.ReadWriteMCAChunkKV
	writeErr error
}

func (s *JavaStorageReadWrite) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeErr != nil {
		return s.writeErr
	}
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
		if err := s.rwdb.Put(p, data); err != nil {
			return err
		}
		c.Changed = false
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
	uint32, uint32,
	*layer_storage.Palette[uint32],
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
	s := &JavaStorageReadWrite{
		JavaStorageReadOnly: NewReadOnlyJavaStorage(rwdb),
		rwdb:                rwdb,
	}
	cache, _ := simplelru.NewLRU(64, func(cp celler.ChunkPos, c *ChunkWithChangeMark) {
		if c == nil || !c.Changed {
			return
		}
		data := chunkEncode.EncodeChunk(cp, c)
		if err := rwdb.Put(cp, data); err != nil {
			if s.writeErr == nil {
				s.writeErr = err
			}
		}
	})
	s.cache = cache
	return s
}
