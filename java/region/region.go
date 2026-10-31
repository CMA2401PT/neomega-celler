// Package region reads and edits Java region containers and their external
// chunk files. Records contain compressed bytes, independent of NBT schemas.
package region

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"celler"
	"celler/storage"
)

const sectorSize = 4096

// Record owns its compressed payload. Timestamp is the region header's Unix time.
type Record struct {
	Compression byte
	Data        []byte
	Timestamp   uint32
}

// Region has a single owner. Flush writes its header; the backend owns durability.
type Region struct {
	fsys            fs.FS
	writer          storage.Writer
	file            fs.File
	read            io.ReaderAt
	write           storage.File
	dir             string
	pos             celler.ChunkPos
	locations       [1024]uint32
	timestamps      [1024]uint32
	used            []bool
	retired         []uint32
	externalDeletes map[string]bool
	dirty           bool
	closed          bool
}

func Name(pos celler.ChunkPos) string { return fmt.Sprintf("r.%d.%d.mca", pos.X, pos.Z) }
func ParseName(name string) (celler.ChunkPos, bool) {
	var p celler.ChunkPos
	n, err := fmt.Sscanf(name, "r.%d.%d.mca", &p.X, &p.Z)
	return p, err == nil && n == 2 && Name(p) == name
}

// Open uses region coordinates (chunk coordinates shifted right by five).
// Writable opens create missing files; existing truncated headers are errors.
func Open(fsys fs.FS, directory string, pos celler.ChunkPos, writable bool) (*Region, error) {
	r := &Region{fsys: fsys, dir: directory, pos: pos, externalDeletes: map[string]bool{}}
	var err error
	if writable {
		var ok bool
		r.writer, ok = fsys.(storage.Writer)
		if !ok {
			return nil, fs.ErrPermission
		}
		if err = r.writer.MkdirAll(directory); err != nil {
			return nil, err
		}
		r.write, err = r.writer.OpenFile(path.Join(directory, Name(pos)), os.O_RDWR|os.O_CREATE)
		r.file = r.write
	} else {
		r.file, err = fsys.Open(path.Join(directory, Name(pos)))
	}
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Region, error) { r.file.Close(); return nil, err }
	var ok bool
	r.read, ok = r.file.(io.ReaderAt)
	if !ok {
		return fail(fmt.Errorf("region files require io.ReaderAt"))
	}
	info, err := r.file.Stat()
	if err != nil {
		return fail(err)
	}
	size := info.Size()
	if size == 0 && writable {
		if err = writeAt(r.write, make([]byte, 2*sectorSize), 0); err != nil {
			return fail(err)
		}
		size = 2 * sectorSize
	}
	if size < 2*sectorSize {
		return fail(fmt.Errorf("truncated region header"))
	}
	var header [2 * sectorSize]byte
	if _, err = r.read.ReadAt(header[:], 0); err != nil {
		return fail(err)
	}
	r.used = make([]bool, (size+sectorSize-1)/sectorSize)
	r.used[0], r.used[1] = true, true
	for i := range r.locations {
		loc := binary.BigEndian.Uint32(header[i*4:])
		r.locations[i] = loc
		r.timestamps[i] = binary.BigEndian.Uint32(header[sectorSize+i*4:])
		if loc == 0 {
			continue
		}
		start, count := int(loc>>8), int(loc&255)
		if start < 2 || count == 0 || int64(start+count)*sectorSize > size {
			return fail(fmt.Errorf("invalid region location at slot %d", i))
		}
		for j := start; j < start+count; j++ {
			if r.used[j] {
				return fail(fmt.Errorf("overlapping region sectors"))
			}
			r.used[j] = true
		}
	}
	return r, nil
}

func (r *Region) slot(pos celler.ChunkPos) (int, error) {
	if r.closed {
		return 0, fs.ErrClosed
	}
	if pos.X>>5 != r.pos.X || pos.Z>>5 != r.pos.Z {
		return 0, fmt.Errorf("chunk %v outside region %v", pos, r.pos)
	}
	return int(pos.X&31) + int(pos.Z&31)*32, nil
}
func (r *Region) external(pos celler.ChunkPos) string {
	return path.Join(r.dir, fmt.Sprintf("c.%d.%d.mcc", pos.X, pos.Z))
}

func (r *Region) Read(pos celler.ChunkPos) (Record, error) {
	i, err := r.slot(pos)
	if err != nil {
		return Record{}, err
	}
	loc := r.locations[i]
	if loc == 0 {
		return Record{}, fs.ErrNotExist
	}
	var head [5]byte
	if _, err = r.read.ReadAt(head[:], int64(loc>>8)*sectorSize); err != nil {
		return Record{}, err
	}
	length := binary.BigEndian.Uint32(head[:])
	if length < 1 || uint64(length)+4 > uint64(loc&255)*sectorSize {
		return Record{}, fmt.Errorf("invalid chunk record length %d", length)
	}
	record := Record{Compression: head[4] & 127, Timestamp: r.timestamps[i]}
	if head[4]&128 != 0 {
		if length != 1 {
			return Record{}, fmt.Errorf("invalid external chunk stub")
		}
		record.Data, err = fs.ReadFile(r.fsys, r.external(pos))
	} else {
		record.Data = make([]byte, int(length)-1)
		if len(record.Data) > 0 {
			_, err = r.read.ReadAt(record.Data, int64(loc>>8)*sectorSize+5)
		}
	}
	return record, err
}

func (r *Region) allocate(count int) int {
	start, run := 2, 0
	for i := 2; i < len(r.used); i++ {
		if r.used[i] {
			start = i + 1
			run = 0
		} else {
			run++
		}
		if run == count {
			return start
		}
	}
	end := start + count
	if end > len(r.used) {
		r.used = append(r.used, make([]bool, end-len(r.used))...)
	}
	return start
}

// Write stages an already-compressed record. Large records use .mcc files.
// Compression may run before this call, independently of the file owner.
func (r *Region) Write(pos celler.ChunkPos, record Record) error {
	i, err := r.slot(pos)
	if err != nil {
		return err
	}
	if r.write == nil {
		return fs.ErrPermission
	}
	if record.Compression == 0 || record.Compression >= 128 {
		return fmt.Errorf("invalid compression ID %d", record.Compression)
	}
	count := (len(record.Data) + 5 + sectorSize - 1) / sectorSize
	external := count > 255
	if external {
		count = 1
	}
	start := r.allocate(count)
	if start > 0xffffff {
		return fmt.Errorf("region sector address exhausted")
	}
	data := make([]byte, count*sectorSize)
	data[4] = record.Compression
	extName := r.external(pos)
	if external {
		// Replacement happens through a separate file so an unsuccessful write
		// cannot truncate the previous external payload.
		if err = storage.WriteFile(r.writer, extName+".tmp", record.Data); err != nil {
			return err
		}
		if err = r.writer.Rename(extName+".tmp", extName); err != nil {
			return err
		}
		data[4] |= 128
		binary.BigEndian.PutUint32(data, 1)
	} else {
		binary.BigEndian.PutUint32(data, uint32(len(record.Data)+1))
		copy(data[5:], record.Data)
	}
	if err = writeAt(r.write, data, int64(start)*sectorSize); err != nil {
		return err
	}
	for j := start; j < start+count; j++ {
		r.used[j] = true
	}
	if r.locations[i] != 0 {
		r.retired = append(r.retired, r.locations[i])
	}
	r.locations[i] = uint32(start)<<8 | uint32(count)
	r.timestamps[i] = record.Timestamp
	r.dirty = true
	if external {
		delete(r.externalDeletes, extName)
	} else {
		r.externalDeletes[extName] = true
	}
	return nil
}

func (r *Region) Delete(pos celler.ChunkPos) error {
	i, err := r.slot(pos)
	if err != nil {
		return err
	}
	if r.write == nil {
		return fs.ErrPermission
	}
	if r.locations[i] == 0 {
		return nil
	}
	r.retired = append(r.retired, r.locations[i])
	r.locations[i] = 0
	r.timestamps[i] = 0
	r.dirty = true
	r.externalDeletes[r.external(pos)] = true
	return nil
}

// Scan visits only occupied slots, in Z-X order.
func (r *Region) Scan(fn func(celler.ChunkPos) error) error {
	if r.closed {
		return fs.ErrClosed
	}
	for i, loc := range r.locations {
		if loc == 0 {
			continue
		}
		p := celler.ChunkPos{X: r.pos.X*32 + int32(i&31), Z: r.pos.Z*32 + int32(i>>5)}
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (r *Region) Flush() error {
	if r.closed {
		return fs.ErrClosed
	}
	if !r.dirty {
		return nil
	}
	var header [2 * sectorSize]byte
	for i, loc := range r.locations {
		binary.BigEndian.PutUint32(header[i*4:], loc)
		binary.BigEndian.PutUint32(header[sectorSize+i*4:], r.timestamps[i])
	}
	if err := writeAt(r.write, header[:], 0); err != nil {
		return err
	}
	for name := range r.externalDeletes {
		if err := r.writer.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	clear(r.externalDeletes)
	for _, loc := range r.retired {
		for j := loc >> 8; j < (loc>>8)+(loc&255); j++ {
			r.used[j] = false
		}
	}
	r.retired = nil
	end := len(r.used)
	for end > 2 && !r.used[end-1] {
		end--
	}
	if err := r.write.Truncate(int64(end) * sectorSize); err != nil {
		return err
	}
	r.used = r.used[:end]
	r.dirty = false
	return nil
}

// Close flushes the header and releases this file, without committing its backend.
func (r *Region) Close() error {
	if r.closed {
		return nil
	}
	err := r.Flush()
	closeErr := r.file.Close()
	r.closed = true
	return errors.Join(err, closeErr)
}

func writeAt(w io.WriterAt, data []byte, off int64) error {
	n, err := w.WriteAt(data, off)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}
