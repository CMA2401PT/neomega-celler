package ldb_stor_mimc

import (
	"celler/kvdb/ldb_stor/mimc"
	"errors"
	"fmt"
	"os"

	"github.com/df-mc/goleveldb/leveldb/storage"
)

type MimcStorage struct {
	*FileHeader
	closed   bool
	readOnly bool
	mimc     *mimc.MimcCore
}

var errReadOnly = errors.New("storage is read-only")

func (fs *MimcStorage) Open(fd storage.FileDesc) (storage.Reader, error) {
	space := fs.FileHeader.GetFD(fd)
	if space == nil {
		return nil, os.ErrNotExist
	} else {
		return fs.mimc.Open(space), nil
	}
}

func (fs *MimcStorage) Create(fd storage.FileDesc) (storage.Writer, error) {
	if fs.readOnly {
		return nil, errReadOnly
	}
	space := fs.GetOrCreateFD(fd)
	return fs.mimc.Create(space), nil
}

func (fs *MimcStorage) Remove(fd storage.FileDesc) error {
	if fs.readOnly {
		return errReadOnly
	}
	space := fs.DeleteFD(fd)
	if space != nil {
		fs.mimc.Remove(space)
	}
	return nil
}

func (fs *MimcStorage) Rename(oldfd, newfd storage.FileDesc) error {
	if fs.readOnly {
		return errReadOnly
	}
	fs.RenameFD(oldfd, newfd)
	return nil
}

func (fs *MimcStorage) Close() error {
	fs.mimc.Close()
	return nil
}

func (fs *MimcStorage) Log(string) {}

func (fs *MimcStorage) shrink() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	allUsed := map[int64]struct{}{0: struct{}{}}
	for _, ci := range fs.mimc.Space0.Clusters {
		allUsed[ci] = struct{}{}
	}
	for _, s := range fs.FileHeader.datas {
		for _, ci := range s.Space.Clusters {
			allUsed[ci] = struct{}{}
		}
	}
	allCi := []int64{}
	for k := range allUsed {
		allCi = append(allCi, k)
	}
	mapping := fs.mimc.MapAndShrink(allCi)
	for i, ci := range fs.mimc.Space0.Clusters {
		if newCi, ok := mapping[ci]; ok {
			fs.mimc.Space0.Clusters[i] = newCi
		}
	}
	fs.mimc.Space0.Flush()
	for _, s := range fs.FileHeader.datas {
		for i, ci := range s.Space.Clusters {
			if newCi, ok := mapping[ci]; ok {
				s.Space.Clusters[i] = newCi
			}
		}
	}
	fs.FileHeader.dump()
	// fs.fullWriteFileHead(false)
	return nil
}

func Shrink(p string) error {
	stor, err := OpenOrCreate(p, false)
	if err != nil {
		return err
	}
	defer stor.Close()
	fmt.Println("before shrink")
	fmt.Println(stor.mimc)
	err = stor.shrink()
	if err != nil {
		return err
	}
	fmt.Println("after shrink")
	fmt.Println(stor.mimc)
	return nil
}

func OpenOrCreate(p string, readOnly bool) (*MimcStorage, error) {
	var mimcCore *mimc.MimcCore
	var err error
	if readOnly {
		mimcCore, err = mimc.NewReadOnlyMimcCore(p, 128*1024)
	} else {
		mimcCore, err = mimc.NewMimcCore(p, 128*1024)
	}
	if err != nil {
		return nil, err
	}
	ms := &MimcStorage{
		mimc:     mimcCore,
		readOnly: readOnly,
	}
	ms.FileHeader = &FileHeader{
		readSlot: func(i int64) ([]byte, error) {
			// fmt.Println("slot: ", i)
			reciver := make([]byte, 256)
			_, err := mimcCore.Open(mimcCore.Space0).ReadAt(reciver, i*256)
			return reciver, err
		},
		writeSlot: func(i int64, b []byte) error {
			if readOnly {
				panic("BUG: read on write only ")
			}
			if len(b) > 256 {
				panic("BUG: max 256 byte fd allowed ")
			}
			_, err := mimcCore.Create(mimcCore.Space0).WriteAt(b, i*256)
			return err
		},
	}
	err = ms.FileHeader.Load()
	if err != nil {
		mimcCore.Close()
		return nil, err
	}
	return ms, nil
}

func NewReadWrite(p string) (*MimcStorage, error) {
	return OpenOrCreate(p, false)
}

func NewReadOnly(p string) (*MimcStorage, error) {
	return OpenOrCreate(p, true)
}
