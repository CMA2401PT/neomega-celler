package mcworld

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/df-mc/goleveldb/leveldb/storage"
)

// zip 里的 LevelDB 不能先解成目录再打开。
// Store 条目按原文件偏移读；Deflate 条目没有随机偏移，第一次打开时才展开到内存。
type zipStorage struct {
	base io.ReaderAt

	mu     sync.Mutex
	closed bool
	files  map[storage.FileDesc]*zip.File
	raw    map[string]*zip.File
	cache  map[string][]byte
}

func newZipStorage(base io.ReaderAt, entries map[string]*zip.File) *zipStorage {
	s := &zipStorage{
		base:  base,
		files: map[storage.FileDesc]*zip.File{},
		raw:   entries,
		cache: map[string][]byte{},
	}
	for name, file := range entries {
		fd, ok := parseName(name)
		if !ok {
			continue
		}
		// 同一个编号若同时有 .ldb 和 .sst，LevelDB 按 .ldb 打开。
		if existing := s.files[fd]; existing != nil && strings.HasSuffix(path.Base(existing.Name), ".ldb") {
			continue
		}
		s.files[fd] = file
	}
	return s
}

func (s *zipStorage) Lock() (storage.Locker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, storage.ErrClosed
	}
	return nopLock{}, nil
}

type nopLock struct{}

func (nopLock) Unlock() {}

func (s *zipStorage) Log(string) {}

func (s *zipStorage) SetMeta(storage.FileDesc) error { return s.readOnly() }
func (s *zipStorage) Create(storage.FileDesc) (storage.Writer, error) {
	return nil, s.readOnly()
}
func (s *zipStorage) Remove(storage.FileDesc) error { return s.readOnly() }
func (s *zipStorage) Rename(storage.FileDesc, storage.FileDesc) error {
	return s.readOnly()
}

func (s *zipStorage) readOnly() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return storage.ErrClosed
	}
	return os.ErrPermission
}

func (s *zipStorage) GetMeta() (storage.FileDesc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return storage.FileDesc{}, storage.ErrClosed
	}
	var last error = os.ErrNotExist
	for _, name := range []string{"CURRENT", "CURRENT.bak"} {
		file := s.raw[name]
		if file == nil {
			continue
		}
		data, err := s.bytes(file)
		if err != nil {
			return storage.FileDesc{}, err
		}
		text := string(bytes.TrimRight(data, "\r\n"))
		fd, ok := parseName(text)
		if !ok || s.files[fd] == nil {
			last = &storage.ErrCorrupted{Err: fmt.Errorf("CURRENT %q", text)}
			continue
		}
		return fd, nil
	}
	return storage.FileDesc{}, last
}

func (s *zipStorage) List(ft storage.FileType) ([]storage.FileDesc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, storage.ErrClosed
	}
	out := make([]storage.FileDesc, 0, len(s.files))
	for fd := range s.files {
		if fd.Type&ft != 0 {
			out = append(out, fd)
		}
	}
	return out, nil
}

func (s *zipStorage) Open(fd storage.FileDesc) (storage.Reader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, storage.ErrClosed
	}
	file := s.files[fd]
	if file == nil {
		return nil, os.ErrNotExist
	}
	if file.Method == zip.Store {
		offset, err := file.DataOffset()
		if err != nil {
			return nil, err
		}
		return sectionReader{SectionReader: io.NewSectionReader(s.base, offset, int64(file.UncompressedSize64))}, nil
	}
	data, err := s.bytes(file)
	if err != nil {
		return nil, err
	}
	return bytesReader{Reader: bytes.NewReader(data)}, nil
}

// bytes 在已经持有 s.mu 时读取一个条目。Deflate 的结果留在缓存里。
func (s *zipStorage) bytes(file *zip.File) ([]byte, error) {
	if data, ok := s.cache[file.Name]; ok {
		return data, nil
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	s.cache[file.Name] = data
	return data, nil
}

func (s *zipStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.cache = nil
	return nil
}

type sectionReader struct{ *io.SectionReader }

func (sectionReader) Close() error { return nil }

type bytesReader struct{ *bytes.Reader }

func (bytesReader) Close() error { return nil }

func parseName(name string) (storage.FileDesc, bool) {
	var fd storage.FileDesc
	var tail string
	if _, err := fmt.Sscanf(name, "%d.%s", &fd.Num, &tail); err == nil {
		switch tail {
		case "log":
			fd.Type = storage.TypeJournal
		case "ldb", "sst":
			fd.Type = storage.TypeTable
		case "tmp":
			fd.Type = storage.TypeTemp
		default:
			return storage.FileDesc{}, false
		}
		return fd, fd.Num >= 0
	}
	if n, _ := fmt.Sscanf(name, "MANIFEST-%d%s", &fd.Num, &tail); n == 1 {
		fd.Type = storage.TypeManifest
		return fd, fd.Num >= 0
	}
	return storage.FileDesc{}, false
}
