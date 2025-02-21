package ldb_stor_memory

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/df-mc/goleveldb/leveldb/storage"
)

// DirStorage is a file-system backed storage.
type MemoryStorage struct {
	datas    map[storage.FileDesc][]byte
	rootFd   storage.FileDesc
	readOnly bool
	mu       sync.Mutex
}

var errReadOnly = errors.New("storage is read-only")

func (fs *MemoryStorage) SetMeta(fd storage.FileDesc) error {
	if fs.readOnly {
		return errReadOnly
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.rootFd = fd
	return nil
}

func (fs *MemoryStorage) GetMeta() (storage.FileDesc, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.rootFd, nil
}

func (fs *MemoryStorage) List(ft storage.FileType) (fds []storage.FileDesc, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fds = make([]storage.FileDesc, 0, len(fs.datas)+1)
	if fs.rootFd.Type&ft != 0 {
		fds = append(fds, fs.rootFd)
	}

	for k := range fs.datas {
		if k.Type&ft == 0 {
			continue
		}
		fds = append(fds, k)
	}
	return fds, nil
}

type CloseWrap struct{ bytes.Reader }

func (CloseWrap) Close() error {
	return nil
}

func (fs *MemoryStorage) Open(fd storage.FileDesc) (storage.Reader, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if data, ok := fs.datas[fd]; !ok {
		return nil, os.ErrNotExist
	} else {

		return &CloseWrap{*bytes.NewReader(data)}, nil
	}
}

type WriteWrap struct {
	bytes.Buffer
	flush func()
}

func (w *WriteWrap) Close() error {
	w.flush()
	return nil
}

func (w *WriteWrap) Sync() error {
	w.flush()
	return nil
}

func (fs *MemoryStorage) Create(fd storage.FileDesc) (storage.Writer, error) {
	fmt.Printf("create: %v\n", fd)
	if fs.readOnly {
		return nil, errReadOnly
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.datas, fd)
	buf := bytes.NewBuffer(nil)
	flush := func() {
		fs.mu.Lock()
		fs.datas[fd] = buf.Bytes()
		fs.mu.Unlock()
	}
	return &WriteWrap{
		Buffer: *buf,
		flush:  flush,
	}, nil
}

func (fs *MemoryStorage) Remove(fd storage.FileDesc) error {
	if fs.readOnly {
		return errReadOnly
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.datas, fd)
	return nil
}

func (fs *MemoryStorage) Rename(oldfd, newfd storage.FileDesc) error {
	if fs.readOnly {
		return errReadOnly
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.datas[newfd] = fs.datas[oldfd]
	delete(fs.datas, oldfd)
	return nil
}

func (fs *MemoryStorage) Close() error {
	return nil
}

func (fs *MemoryStorage) Log(string) {}

func NewReadWrite() *MemoryStorage {
	return &MemoryStorage{
		datas: make(map[storage.FileDesc][]byte),
		mu:    sync.Mutex{},
	}
}

func NewReadOnly() *MemoryStorage {
	return &MemoryStorage{
		datas:    make(map[storage.FileDesc][]byte),
		mu:       sync.Mutex{},
		readOnly: true,
	}
}
