package java

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"time"

	"celler"
	"celler/java/region"
	"celler/storage"
	nbt "neomega-nbt/presets/v1"
)

// Dimension is the world's relative directory for a dimension. Custom dimensions
// use their native dimensions/<namespace>/<path> directory.
type Dimension string

const (
	Overworld Dimension = "."
	Nether    Dimension = "DIM-1"
	End       Dimension = "DIM1"
)

type RegionKind string

const (
	Terrain  RegionKind = "region"
	Entities RegionKind = "entities"
	POI      RegionKind = "poi"
)

type Options struct {
	ReadOnly bool
	// MaxOpenRegions bounds file handles and cached headers. Zero selects 16.
	MaxOpenRegions int
}

// World borrows Files. Close releases region handles; Commit flushes all regions
// before committing the backend. The caller owns a borrowed PackFS container.
type World struct {
	Files   fs.FS
	options Options
	regions map[string]*region.Region
	order   []string
	closed  bool
}

func Open(files fs.FS, options Options) (*World, error) {
	if !options.ReadOnly {
		if _, ok := files.(storage.Writer); !ok {
			return nil, fs.ErrPermission
		}
	}
	if options.MaxOpenRegions <= 0 {
		options.MaxOpenRegions = 16
	}
	if _, err := fs.Stat(files, "."); err != nil {
		return nil, err
	}
	return &World{Files: files, options: options, regions: map[string]*region.Region{}}, nil
}
func OpenDirectory(directory string, options Options) (*World, error) {
	if options.ReadOnly {
		return Open(os.DirFS(directory), options)
	}
	files, err := storage.OpenDirectory(directory)
	if err != nil {
		return nil, err
	}
	return Open(files, options)
}

func regionDir(dim Dimension, kind RegionKind) (string, error) {
	if !fs.ValidPath(string(dim)) {
		return "", fs.ErrInvalid
	}
	switch kind {
	case Terrain, Entities, POI:
	default:
		return "", fmt.Errorf("invalid region kind %q", kind)
	}
	return path.Join(string(dim), string(kind)), nil
}
func (w *World) getRegion(dim Dimension, kind RegionKind, pos celler.ChunkPos, create bool) (*region.Region, error) {
	if w.closed {
		return nil, fs.ErrClosed
	}
	dir, err := regionDir(dim, kind)
	if err != nil {
		return nil, err
	}
	rp := celler.ChunkPos{X: pos.X >> 5, Z: pos.Z >> 5}
	name := path.Join(dir, region.Name(rp))
	if r, ok := w.regions[name]; ok {
		w.touch(name)
		return r, nil
	}
	if !create {
		if _, err := fs.Stat(w.Files, name); err != nil {
			return nil, err
		}
	}
	if len(w.regions) >= w.options.MaxOpenRegions {
		oldest := w.order[0]
		if err := w.regions[oldest].Close(); err != nil {
			return nil, err
		}
		delete(w.regions, oldest)
		w.order = w.order[1:]
	}
	r, err := region.Open(w.Files, dir, rp, !w.options.ReadOnly)
	if err != nil {
		return nil, err
	}
	w.regions[name] = r
	w.order = append(w.order, name)
	return r, nil
}
func (w *World) touch(name string) {
	i := slices.Index(w.order, name)
	w.order = append(slices.Delete(w.order, i, i+1), name)
}
func (w *World) ReadRaw(dim Dimension, kind RegionKind, pos celler.ChunkPos) (region.Record, error) {
	r, err := w.getRegion(dim, kind, pos, false)
	if err != nil {
		return region.Record{}, err
	}
	return r.Read(pos)
}
func (w *World) WriteRaw(dim Dimension, kind RegionKind, pos celler.ChunkPos, record region.Record) error {
	if w.options.ReadOnly {
		return fs.ErrPermission
	}
	r, err := w.getRegion(dim, kind, pos, true)
	if err != nil {
		return err
	}
	return r.Write(pos, record)
}
func (w *World) Delete(dim Dimension, kind RegionKind, pos celler.ChunkPos) error {
	if w.options.ReadOnly {
		return fs.ErrPermission
	}
	r, err := w.getRegion(dim, kind, pos, false)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.Delete(pos)
}
func (w *World) ReadChunk(dim Dimension, pos celler.ChunkPos) (*Chunk, error) {
	raw, err := w.ReadRaw(dim, Terrain, pos)
	if err != nil {
		return nil, err
	}
	data, err := raw.Decode()
	if err != nil {
		return nil, err
	}
	c, err := DecodeChunk(data)
	if err != nil {
		return nil, err
	}
	cp, err := c.Position()
	if err != nil {
		return nil, err
	}
	if cp != pos {
		return nil, fmt.Errorf("chunk position %v does not match slot %v", cp, pos)
	}
	return c, nil
}
func (w *World) WriteChunk(dim Dimension, c *Chunk) error {
	pos, err := c.Position()
	if err != nil {
		return err
	}
	data, err := c.Encode()
	if err != nil {
		return err
	}
	raw, err := region.Compress(data, uint32(time.Now().Unix()))
	if err != nil {
		return err
	}
	return w.WriteRaw(dim, Terrain, pos, raw)
}

// Scan visits existing records. Callbacks may read or edit the world; slot
// positions for each region are captured before invoking the callback.
func (w *World) Scan(dim Dimension, kind RegionKind, fn func(celler.ChunkPos) error) error {
	if w.closed {
		return fs.ErrClosed
	}
	dir, err := regionDir(dim, kind)
	if err != nil {
		return err
	}
	entries, err := fs.ReadDir(w.Files, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		rp, ok := region.ParseName(entry.Name())
		if !ok {
			continue
		}
		r, err := w.getRegion(dim, kind, celler.ChunkPos{X: rp.X * 32, Z: rp.Z * 32}, false)
		if err != nil {
			return err
		}
		var positions []celler.ChunkPos
		if err = r.Scan(func(p celler.ChunkPos) error { positions = append(positions, p); return nil }); err != nil {
			return err
		}
		for _, p := range positions {
			if err = fn(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// ReadLevel reads the gzip-compressed level.dat document, including its root name.
func (w *World) ReadLevel() (nbt.Document, error) {
	if w.closed {
		return nbt.Document{}, fs.ErrClosed
	}
	f, err := w.Files.Open("level.dat")
	if err != nil {
		return nbt.Document{}, err
	}
	defer f.Close()
	reader, err := gzip.NewReader(f)
	if err != nil {
		return nbt.Document{}, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nbt.Document{}, err
	}
	return nbt.DecodeDocument(data, nbt.EncodingBigEndian)
}
func (w *World) WriteLevel(doc nbt.Document) error {
	if w.closed {
		return fs.ErrClosed
	}
	if w.options.ReadOnly {
		return fs.ErrPermission
	}
	data, err := nbt.EncodeDocument(doc, nbt.EncodingBigEndian)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err = writer.Write(data); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	backend := w.Files.(storage.Writer)
	if err = storage.WriteFile(backend, "level.dat.tmp", buffer.Bytes()); err != nil {
		return err
	}
	return backend.Rename("level.dat.tmp", "level.dat")
}
func (w *World) Flush() error {
	if w.closed {
		return fs.ErrClosed
	}
	for _, name := range w.order {
		if err := w.regions[name].Flush(); err != nil {
			return err
		}
	}
	return nil
}
func (w *World) Commit() error {
	if w.options.ReadOnly {
		return fs.ErrPermission
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return w.Files.(storage.Writer).Commit()
}
func (w *World) Close() error {
	if w.closed {
		return nil
	}
	var result error
	for _, name := range w.order {
		result = errors.Join(result, w.regions[name].Close())
	}
	w.closed = true
	clear(w.regions)
	w.order = nil
	return result
}
