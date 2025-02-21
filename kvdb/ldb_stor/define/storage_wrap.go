package define

import "github.com/df-mc/goleveldb/leveldb/storage"

type VoidFL struct{}

func (VoidFL) Unlock() {
}

type StorageWrapper[S Storage] struct {
	Storage S
}

func (StorageWrapper[S]) Lock() (storage.Locker, error) {
	return VoidFL{}, nil
}

func (s StorageWrapper[S]) GetMeta() (storage.FileDesc, error) {
	return s.Storage.GetMeta()
}

func (s StorageWrapper[S]) List(ft storage.FileType) ([]storage.FileDesc, error) {
	return s.Storage.List(ft)
}
func (s StorageWrapper[S]) Open(fd storage.FileDesc) (storage.Reader, error) {
	return s.Storage.Open(fd)
}
func (s StorageWrapper[S]) Close() error {
	return s.Storage.Close()
}

func (s StorageWrapper[S]) Log(str string) {
	s.Storage.Log(str)
}

func (s StorageWrapper[S]) SetMeta(fd storage.FileDesc) error {
	return s.Storage.SetMeta(fd)
}

func (s StorageWrapper[S]) Create(fd storage.FileDesc) (storage.Writer, error) {
	return s.Storage.Create(fd)
}

func (s StorageWrapper[S]) Rename(oldfd, newfd storage.FileDesc) error {
	return s.Storage.Rename(oldfd, newfd)
}

func (s StorageWrapper[S]) Remove(fd storage.FileDesc) error {
	return s.Storage.Remove(fd)
}

type ReadOnlyStorageWrapper[S ReadOnlyStorage] struct {
	Storage S
}

func (ReadOnlyStorageWrapper[S]) Lock() (storage.Locker, error) {
	return VoidFL{}, nil
}

func (s ReadOnlyStorageWrapper[S]) GetMeta() (storage.FileDesc, error) {
	return s.Storage.GetMeta()
}

func (s ReadOnlyStorageWrapper[S]) List(ft storage.FileType) ([]storage.FileDesc, error) {
	return s.Storage.List(ft)
}
func (s ReadOnlyStorageWrapper[S]) Open(fd storage.FileDesc) (storage.Reader, error) {
	return s.Storage.Open(fd)
}
func (s ReadOnlyStorageWrapper[S]) Close() error {
	return s.Storage.Close()
}

func (s ReadOnlyStorageWrapper[S]) Log(str string) {
}

func (s ReadOnlyStorageWrapper[S]) SetMeta(fd storage.FileDesc) error {
	panic("not implement")
}

func (s ReadOnlyStorageWrapper[S]) Create(fd storage.FileDesc) (storage.Writer, error) {
	panic("not implement")
}

func (s ReadOnlyStorageWrapper[S]) Remove(fd storage.FileDesc) error {
	panic("not implement")
}

func (s ReadOnlyStorageWrapper[S]) Rename(oldfd, newfd storage.FileDesc) error {
	panic("not implement")
}
