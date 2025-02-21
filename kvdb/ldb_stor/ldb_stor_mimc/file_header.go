package ldb_stor_mimc

import (
	"bytes"
	"celler/kvdb/ldb_stor/mimc"
	"encoding/binary"
	"io"
	"sync"

	"github.com/df-mc/goleveldb/leveldb/storage"
)

type SpaceWithSlot struct {
	Space *mimc.Space
	Slot  int64
}

type FileHeader struct {
	mu        sync.RWMutex
	datas     map[storage.FileDesc]*SpaceWithSlot
	root      storage.FileDesc
	slots     int64
	reclaims  []int64
	readSlot  func(int64) ([]byte, error)
	writeSlot func(int64, []byte) error
}

func dumpFD(fd storage.FileDesc, w func(b byte) error) (err error) {
	err = mimc.WriteVarInt64(w, fd.Num)
	if err != nil {
		return err
	}
	err = mimc.WriteVarInt64(w, int64(fd.Type))
	if err != nil {
		return err
	}
	return nil
}

func loadFD(r func() (byte, error)) (fd storage.FileDesc, err error) {
	fd.Num, err = mimc.ReadVarInt64(r)
	if err != nil {
		return fd, err
	}
	t, err := mimc.ReadVarInt64(r)
	if err != nil {
		return fd, err
	}
	fd.Type = storage.FileType(t)
	return fd, nil
}

func (fh *FileHeader) dump() {
	fh.updateRoot()
	for fd, ss := range fh.datas {
		fh.updateSlot(ss.Slot, fd, ss.Space)
	}
}

func (fh *FileHeader) Load() (err error) {
	fh.mu = sync.RWMutex{}
	fh.datas = make(map[storage.FileDesc]*SpaceWithSlot)
	fh.slots = 0
	fh.reclaims = make([]int64, 0)

	bs, err := fh.readSlot(0)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	fh.slots = int64(binary.LittleEndian.Uint64(bs[:8]))
	buf := bytes.NewBuffer(bs[8:])
	fh.root, err = loadFD(buf.ReadByte)
	fh.mu = sync.RWMutex{}
	if err != nil {
		return err
	}
	for i := int64(0); i < int64(fh.slots); i++ {
		slotI := int64(i) + 1
		bs, err = fh.readSlot(slotI)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return err
		}
		if bs[0] == 0 {
			fh.reclaims = append(fh.reclaims, slotI)
			continue
		}
		buf := bytes.NewBuffer(bs[4:])
		fd, err := loadFD(buf.ReadByte)
		if err != nil {
			return err
		}
		space, err := mimc.LoadSpace(buf.ReadByte)
		if err != nil {
			return err
		}
		space.SetChangeHook(func(s *mimc.Space) {
			fh.mu.Lock()
			defer fh.mu.Unlock()
			if fh.datas[fd].Slot != slotI {
				panic("BUG: slot move!")
			}
			if fh.datas[fd].Space != s {
				panic("BUG: space move!")
			}
			fh.updateSlot(slotI, fd, s)
		})
		fh.datas[fd] = &SpaceWithSlot{
			Space: space,
			Slot:  slotI,
		}
	}
	return nil
}

func (fh *FileHeader) DeleteFD(fd storage.FileDesc) *mimc.Space {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if ss, ok := fh.datas[fd]; ok {
		fh.writeSlot(ss.Slot, []byte{0})
		delete(fh.datas, fd)
		fh.reclaims = append(fh.reclaims, ss.Slot)
		return ss.Space
	}
	return nil
}

func (fh *FileHeader) RenameFD(oldFd, newFd storage.FileDesc) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if ss, ok := fh.datas[oldFd]; ok {
		delete(fh.datas, oldFd)
		slot := ss.Slot
		space := ss.Space
		fh.datas[newFd] = &SpaceWithSlot{
			Slot:  slot,
			Space: space,
		}
		ss.Space.SetChangeHook(func(s *mimc.Space) {
			fh.mu.Lock()
			defer fh.mu.Unlock()
			if fh.datas[newFd].Slot != slot {
				panic("BUG: slot move!")
			}
			if fh.datas[newFd].Space != s {
				panic("BUG: space move!")
			}
			fh.updateSlot(slot, newFd, s)
		})
		fh.updateSlot(slot, newFd, space)
	}
}

func (fh *FileHeader) updateSlot(slotI int64, fd storage.FileDesc, space *mimc.Space) {
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte{1, 0, 0, 0})
	dumpFD(fd, buf.WriteByte)
	space.Dump(buf.WriteByte)
	fh.writeSlot(slotI, buf.Bytes())
}

func (fh *FileHeader) GetFD(fd storage.FileDesc) *mimc.Space {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if ss, ok := fh.datas[fd]; ok {
		return ss.Space
	}
	return nil
}

func (fh *FileHeader) GetOrCreateFD(fd storage.FileDesc) *mimc.Space {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if ss, ok := fh.datas[fd]; ok {
		return ss.Space
	}
	slotI := int64(0)
	if len(fh.reclaims) > 0 {
		slotI = fh.reclaims[0]
		fh.reclaims = fh.reclaims[1:]
	} else {
		slotI = fh.slots + 1
		fh.slots += 1
		fh.updateRoot()
	}
	space := &mimc.Space{}
	space.SetChangeHook(func(s *mimc.Space) {
		fh.mu.Lock()
		defer fh.mu.Unlock()
		if fh.datas[fd].Slot != slotI {
			panic("BUG: slot move!")
		}
		if fh.datas[fd].Space != s {
			panic("BUG: space move!")
		}
		fh.updateSlot(slotI, fd, s)
	})
	fh.updateSlot(slotI, fd, space)
	ss := &SpaceWithSlot{
		Slot:  slotI,
		Space: space,
	}
	fh.datas[fd] = ss
	return space
}

func (fh *FileHeader) List(ft storage.FileType) (fds []storage.FileDesc, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fds = make([]storage.FileDesc, 0, len(fh.datas)+1)
	if fh.root.Type&ft != 0 {
		fds = append(fds, fh.root)
	}

	for k := range fh.datas {
		if k.Type&ft == 0 {
			continue
		}
		fds = append(fds, k)
	}
	return fds, nil
}

func (fh *FileHeader) GetMeta() (storage.FileDesc, error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.root, nil
}

func (fh *FileHeader) updateRoot() {
	buf := bytes.NewBuffer(nil)
	fl := make([]byte, 8)
	binary.LittleEndian.PutUint64(fl, uint64(fh.slots))
	buf.Write(fl)
	dumpFD(fh.root, buf.WriteByte)
	fh.writeSlot(0, buf.Bytes())
}

func (fh *FileHeader) SetMeta(fd storage.FileDesc) error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.root = fd
	fh.updateRoot()
	return nil
}
