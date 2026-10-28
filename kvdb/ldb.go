package kvdb

import (
	"bytes"
	"celler/kvdb/ldb_stor/define"
	"celler/kvdb/ldb_stor/ldb_stor_dir"
	"celler/kvdb/ldb_stor/ldb_stor_memory"
	"celler/kvdb/ldb_stor/ldb_stor_mimc"

	"path/filepath"
	"strings"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/iterator"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

type simplifiedLDB struct {
	*leveldb.DB
}

func (db simplifiedLDB) Put(key []byte, value []byte) error {
	return db.DB.Put(key, value, nil)
}

func (db simplifiedLDB) Delete(key []byte) error {
	return db.DB.Delete(key, nil)
}

type IterWithCopy struct {
	iterator.Iterator
}

func (ic IterWithCopy) Value() []byte {
	return bytes.Clone((iterator.Iterator)(ic.Iterator).Value())
}

func (ic IterWithCopy) Key() []byte {
	return bytes.Clone((iterator.Iterator)(ic.Iterator).Key())
}

func (db simplifiedLDB) IterAll(interFn func(CanKV) bool) {
	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		if !interFn(IterWithCopy{iter}) {
			return
		}
	}
}

func (db simplifiedLDB) Get(key []byte) (value []byte, err error) {
	return db.DB.Get(key, nil)
}

func NewReadOnlyLevelDB(path string) (ReadOnlyKVDB, error) {
	if path == "" {
		s := ldb_stor_memory.NewReadOnly()
		db, err := leveldb.Open(define.ReadOnlyStorageWrapper[*ldb_stor_memory.MemoryStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
			ReadOnly:    true,
		})
		return simplifiedLDB{db}, err

	}
	if isMimcPath(path) {
		s, err := ldb_stor_mimc.NewReadOnly(path)
		if err != nil {
			return nil, err
		}
		db, err := leveldb.Open(define.ReadOnlyStorageWrapper[*ldb_stor_mimc.MimcStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
			ReadOnly:    true,
		})
		return simplifiedLDB{db}, err
	}
	s, err := ldb_stor_dir.NewReadOnly(path)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.Open(define.ReadOnlyStorageWrapper[*ldb_stor_dir.DirStorage]{Storage: s}, &opt.Options{
		Compression: opt.FlateCompression,
		BlockSize:   16 * opt.KiB,
		ReadOnly:    true,
	})
	return simplifiedLDB{db}, err
}

func NewReadWriteLevelDB(path string) (KVDB, error) {
	if path == "" {
		s := ldb_stor_memory.NewReadWrite()
		db, err := leveldb.Open(define.StorageWrapper[*ldb_stor_memory.MemoryStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
		})
		return simplifiedLDB{db}, err

	}
	if isMimcPath(path) {
		s, err := ldb_stor_mimc.NewReadWrite(path)
		if err != nil {
			return nil, err
		}
		db, err := leveldb.Open(define.StorageWrapper[*ldb_stor_mimc.MimcStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
		})
		return simplifiedLDB{db}, err
	}
	s, err := ldb_stor_dir.NewReadWrite(path)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.Open(define.StorageWrapper[*ldb_stor_dir.DirStorage]{Storage: s}, &opt.Options{
		Compression: opt.FlateCompression,
		BlockSize:   16 * opt.KiB,
	})
	return simplifiedLDB{db}, err
}

func NewReadWriteLevelDBWithFix(path string) (KVDB, error) {
	if path == "" {
		s := ldb_stor_memory.NewReadWrite()
		db, err := leveldb.Recover(define.StorageWrapper[*ldb_stor_memory.MemoryStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
		})
		return simplifiedLDB{db}, err

	}
	if isMimcPath(path) {
		s, err := ldb_stor_mimc.NewReadWrite(path)
		if err != nil {
			return nil, err
		}
		db, err := leveldb.Recover(define.StorageWrapper[*ldb_stor_mimc.MimcStorage]{Storage: s}, &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
		})
		return simplifiedLDB{db}, err
	}
	s, err := ldb_stor_dir.NewReadWrite(path)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.Recover(define.StorageWrapper[*ldb_stor_dir.DirStorage]{Storage: s}, &opt.Options{
		Compression: opt.FlateCompression,
		BlockSize:   16 * opt.KiB,
	})
	return simplifiedLDB{db}, err
}

func ShrinkMimcLevelDB(path string) error {
	return ldb_stor_mimc.Shrink(path)
}

// Both names have been used for the same single-file LevelDB container.
func isMimcPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mimc" || ext == ".nomg"
}
