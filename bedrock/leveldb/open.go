// Package leveldb opens Bedrock databases on the host filesystem.
package leveldb

import (
	db "github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

// Open uses the engine's native iterators, snapshots and batches. An existing
// database's compression tags are read by the engine; new tables use raw deflate.
func Open(directory string, readOnly bool) (*db.DB, error) {
	return db.OpenFile(directory, &opt.Options{
		ReadOnly:       readOnly,
		ErrorIfMissing: readOnly,
		Compression:    opt.FlateCompression,
		BlockSize:      16 * opt.KiB,
	})
}
