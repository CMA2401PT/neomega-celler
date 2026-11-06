package bedrock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"celler"
	native "celler/bedrock/leveldb"
	"celler/storage"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/df-mc/goleveldb/leveldb/util"
	nbt "neomega-nbt/presets/v1"
)

type Options struct{ ReadOnly bool }

// World owns its filesystem LevelDB engine. DB exposes native batches,
// snapshots and non-chunk records; Files contains level.dat and auxiliary files.
type World struct {
	DB        *leveldb.DB
	Files     fs.FS
	directory *storage.Directory
	readOnly  bool
	closed    bool
}

func Open(directory string, options Options) (*World, error) {
	w := &World{readOnly: options.ReadOnly}
	if options.ReadOnly {
		w.Files = os.DirFS(directory)
	} else {
		var err error
		w.directory, err = storage.OpenDirectory(directory)
		if err != nil {
			return nil, err
		}
		w.Files = w.directory
	}
	db, err := native.Open(filepath.Join(directory, "db"), options.ReadOnly)
	if err != nil {
		return nil, err
	}
	w.DB = db
	return w, nil
}

// NewReadOnly 用已经打开的数据库和世界文件组成只读世界。
// mcworld 的 LevelDB 不在目录里，调用方负责在 Close 之后释放自己的文件。
func NewReadOnly(db *leveldb.DB, files fs.FS) *World {
	return &World{DB: db, Files: files, readOnly: true}
}

func (w *World) ReadChunk(dimension int32, pos celler.ChunkPos) (*Chunk, error) {
	if w.closed {
		return nil, fs.ErrClosed
	}
	c := NewChunk(pos, dimension)
	iter := w.DB.NewIterator(util.BytesPrefix(ChunkPrefix(pos, dimension)), nil)
	defer iter.Release()
	for iter.Next() {
		p, d, suffix, ok := ParseChunkKey(iter.Key())
		if ok && p == pos && d == dimension {
			c.Records[suffix] = bytes.Clone(iter.Value())
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	if len(c.Records) == 0 {
		return nil, fs.ErrNotExist
	}
	return c, nil
}

// WriteChunk atomically replaces this chunk's recognized record set, preserving
// all other database keys. WriteOptions.Sync selects per-batch durability.
// ReadChunkActors 读取一个柱子的实体。Bedrock 用 digp 记录保存每柱的实体 id，
// 每个 id 对应一条 actorprefix 记录；缺失的 actor 记录直接跳过。
func (w *World) ReadChunkActors(dimension int32, pos celler.ChunkPos) ([]nbt.Document, error) {
	if w.closed {
		return nil, fs.ErrClosed
	}
	key := append([]byte("digp"), ChunkPrefix(pos, dimension)...)
	data, err := w.DB.Get(key, nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data)%8 != 0 {
		return nil, fmt.Errorf("digp record length %d is not a multiple of 8", len(data))
	}
	documents := make([]nbt.Document, 0, len(data)/8)
	for offset := 0; offset < len(data); offset += 8 {
		actorKey := append([]byte("actorprefix"), data[offset:offset+8]...)
		value, err := w.DB.Get(actorKey, nil)
		if errors.Is(err, leveldb.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		document, err := nbt.DecodeDocument(value, nbt.EncodingLittleEndian)
		if err != nil {
			// 单条实体损坏不应让整柱读取失败。
			continue
		}
		documents = append(documents, document)
	}
	return documents, nil
}

// Actor 是一条待写入的 Bedrock 实体记录。ID 是 digp 里的 8 字节实体键。
type Actor struct {
	ID  [8]byte
	NBT nbt.Document
}

// WriteChunkActors 写入一个柱子的 digp 索引和每条 actorprefix 记录。
// actors 为空时删除该柱的 digp 记录。
func (w *World) WriteChunkActors(dimension int32, pos celler.ChunkPos, actors []Actor) error {
	if w.closed {
		return fs.ErrClosed
	}
	digp := append([]byte("digp"), ChunkPrefix(pos, dimension)...)
	ids := make([]byte, 0, len(actors)*8)
	for _, actor := range actors {
		data, err := nbt.EncodeDocument(actor.NBT, nbt.EncodingLittleEndian)
		if err != nil {
			return err
		}
		key := append([]byte("actorprefix"), actor.ID[:]...)
		if err := w.DB.Put(key, data, nil); err != nil {
			return err
		}
		ids = append(ids, actor.ID[:]...)
	}
	if len(ids) == 0 {
		return w.DB.Delete(digp, nil)
	}
	return w.DB.Put(digp, ids, nil)
}

func (w *World) WriteChunk(c *Chunk, options *opt.WriteOptions) error {
	if w.closed {
		return fs.ErrClosed
	}
	if w.readOnly {
		return fs.ErrPermission
	}
	prefix := ChunkPrefix(c.Pos, c.Dimension)
	batch := new(leveldb.Batch)
	for suffix, value := range c.Records {
		key := append(bytes.Clone(prefix), []byte(suffix)...)
		p, d, s, ok := ParseChunkKey(key)
		if !ok || p != c.Pos || d != c.Dimension || s != suffix {
			return fmt.Errorf("invalid chunk record suffix %x", suffix)
		}
		batch.Put(key, value)
	}
	iter := w.DB.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		p, d, suffix, ok := ParseChunkKey(iter.Key())
		if ok && p == c.Pos && d == c.Dimension {
			if _, keep := c.Records[suffix]; !keep {
				batch.Delete(iter.Key())
			}
		}
	}
	err := iter.Error()
	iter.Release()
	if err != nil {
		return err
	}
	return w.DB.Write(batch, options)
}
func (w *World) DeleteChunk(dimension int32, pos celler.ChunkPos, options *opt.WriteOptions) error {
	return w.WriteChunk(NewChunk(pos, dimension), options)
}

// ScanChunks reads a consistent database snapshot. It buffers one X/Z group's
// dimensions, cloning record bytes before the underlying iterator advances.
func (w *World) ScanChunks(fn func(*Chunk) error) error {
	if w.closed {
		return fs.ErrClosed
	}
	snapshot, err := w.DB.GetSnapshot()
	if err != nil {
		return err
	}
	defer snapshot.Release()
	iter := snapshot.NewIterator(nil, nil)
	defer iter.Release()
	var current celler.ChunkPos
	have := false
	group := map[int32]*Chunk{}
	flush := func() error {
		dims := make([]int32, 0, len(group))
		for d := range group {
			dims = append(dims, d)
		}
		sort.Slice(dims, func(i, j int) bool { return dims[i] < dims[j] })
		for _, d := range dims {
			if err := fn(group[d]); err != nil {
				return err
			}
		}
		clear(group)
		return nil
	}
	for iter.Next() {
		pos, dim, suffix, ok := ParseChunkKey(iter.Key())
		if !ok {
			continue
		}
		if have && pos != current {
			if err := flush(); err != nil {
				return err
			}
		}
		current = pos
		have = true
		c := group[dim]
		if c == nil {
			c = NewChunk(pos, dim)
			group[dim] = c
		}
		c.Records[suffix] = bytes.Clone(iter.Value())
	}
	if err := iter.Error(); err != nil {
		return err
	}
	return flush()
}

// Level retains the Bedrock level.dat header version and little-endian root.
type Level struct {
	Version uint32
	NBT     nbt.Document
}

func (w *World) ReadLevel() (Level, error) {
	if w.closed {
		return Level{}, fs.ErrClosed
	}
	data, err := fs.ReadFile(w.Files, "level.dat")
	if err != nil {
		return Level{}, err
	}
	if len(data) < 8 || uint64(binary.LittleEndian.Uint32(data[4:])) != uint64(len(data)-8) {
		return Level{}, fmt.Errorf("invalid level.dat length")
	}
	doc, err := nbt.DecodeDocument(data[8:], nbt.EncodingLittleEndian)
	return Level{Version: binary.LittleEndian.Uint32(data), NBT: doc}, err
}
func (w *World) WriteLevel(level Level) error {
	if w.closed {
		return fs.ErrClosed
	}
	if w.readOnly {
		return fs.ErrPermission
	}
	data, err := nbt.EncodeDocument(level.NBT, nbt.EncodingLittleEndian)
	if err != nil {
		return err
	}
	header := binary.LittleEndian.AppendUint32(nil, level.Version)
	header = binary.LittleEndian.AppendUint32(header, uint32(len(data)))
	if err = storage.WriteFile(w.directory, "level.dat.tmp", append(header, data...)); err != nil {
		return err
	}
	if err = w.directory.Rename("level.dat.tmp", "level.dat"); err != nil {
		return err
	}
	return w.directory.Commit()
}
func (w *World) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	err := w.DB.Close()
	if w.directory != nil {
		err = errors.Join(err, w.directory.Commit())
	}
	return err
}
