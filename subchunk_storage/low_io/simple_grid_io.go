package low_io

import (
	"bytes"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type SimpleGridIO struct {
	storage    map[uint64][]byte
	writing    bool
	compressor Compressor
	// 当写入时，可以先从 cache 检查是否已经打开以供读取
	// 这对于无修改的 dump 很有帮助
	cache        *simplelru.LRU[uint64, *SimpleMemoryIOToBytes]
	SharedBuffer *bytes.Buffer
}

func (gio SimpleGridIO) TotalSize() (s int) {
	for _, v := range gio.storage {
		s += len(v)
	}
	return
}

func NewSimpleGridIO(compress Compressor, maxCaches int) *SimpleGridIO {
	cache, _ := simplelru.NewLRU[uint64, *SimpleMemoryIOToBytes](maxCaches, func(key uint64, value *SimpleMemoryIOToBytes) {})
	return &SimpleGridIO{
		storage:      make(map[uint64][]byte),
		compressor:   compress,
		cache:        cache,
		SharedBuffer: bytes.NewBuffer(nil),
	}
}

func (gio SimpleGridIO) IterGrid(fn func(uint64) (next bool)) {
	for k := range gio.storage {
		if !fn(k) {
			return
		}
	}
}

func (gio SimpleGridIO) DataWriter(key uint64) SubChunkDataWriteCloser {
	if gio.writing {
		panic("reopen writer on write")
	}
	w, _ := gio.cache.Get(key)
	if w != nil {
		gio.cache.Remove(key)
	} else {
		if data, found := gio.storage[key]; !found {
			w = NewSimpleMemoryIOToBytes(nil)
		} else {
			w = NewSimpleMemoryIOToBytes(gio.compressor.Decompress(data))
		}
	}
	w.SharedBuffer = gio.SharedBuffer
	w.SharedBuffer.Reset()
	gio.writing = true
	w.onClose = func(b []byte, change bool) {
		gio.writing = false
		if change {
			gio.storage[key] = bytes.Clone(gio.compressor.Compress(b))
		}
	}
	return w
}

func (gio SimpleGridIO) DropGrid(key uint64) {
	if gio.writing {
		panic("drop on write")
	}
	delete(gio.storage, key)
	gio.cache.Remove(key)
}

func (gio SimpleGridIO) DataReader(key uint64) SubChunkDataReadCloser {
	if gio.writing {
		panic("open reader on write")
	}
	w, _ := gio.cache.Get(key)
	if w != nil {
		return w
	}
	if data, found := gio.storage[key]; !found {
		return nil
	} else {
		w = NewSimpleMemoryIOToBytes(gio.compressor.Decompress(data))
		gio.cache.Add(key, w)
	}
	return w
}
