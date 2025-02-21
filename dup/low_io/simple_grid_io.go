package low_io

import (
	"bytes"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type SimpleGridIO struct {
	storage    map[GridPos][]byte
	writing    bool
	compressor Compressor
	// 当写入时，可以先从 cache 检查是否已经打开以供读取
	// 这对于无修改的 dump 很有帮助
	cache                        *simplelru.LRU[GridPos, *SimpleMemoryIOToBytes]
	SharedBuffer                 *bytes.Buffer
	WriterCounter, ReaderCounter int
}

func (gio SimpleGridIO) TotalSize() (s int) {
	for _, v := range gio.storage {
		s += len(v)
	}
	return
}

func NewSimpleGridIO(compress Compressor, maxCaches int) *SimpleGridIO {
	cache, _ := simplelru.NewLRU(maxCaches, func(key GridPos, value *SimpleMemoryIOToBytes) {})
	return &SimpleGridIO{
		storage:      make(map[GridPos][]byte),
		compressor:   compress,
		cache:        cache,
		SharedBuffer: bytes.NewBuffer(nil),
	}
}

func (gio *SimpleGridIO) IterGrid(fn func(GridPos) (next bool)) {
	for k := range gio.storage {
		if !fn(k) {
			return
		}
	}
}

func (gio *SimpleGridIO) DataWriter(key GridPos) SubChunkDataWriteCloser {
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
			gio.ReaderCounter += 1
			w = NewSimpleMemoryIOToBytes(gio.compressor.Decompress(data))
		}
	}
	w.SharedBuffer = gio.SharedBuffer
	w.SharedBuffer.Reset()
	gio.writing = true
	w.onClose = func(b []byte, change bool) {
		gio.writing = false
		if change {
			gio.WriterCounter += 1
			gio.storage[key] = bytes.Clone(gio.compressor.Compress(b))
		}
	}
	return w
}

func (gio *SimpleGridIO) DropGrid(key GridPos) {
	if gio.writing {
		panic("drop on write")
	}
	delete(gio.storage, key)
	gio.cache.Remove(key)
}

func (gio *SimpleGridIO) DataReader(key GridPos) SubChunkDataReadCloser {
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
		gio.ReaderCounter += 1
		w = NewSimpleMemoryIOToBytes(gio.compressor.Decompress(data))
		gio.cache.Add(key, w)
	}
	return w
}
