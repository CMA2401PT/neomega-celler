package define

import "github.com/df-mc/goleveldb/leveldb/storage"

type ReadOnlyStorage interface {
	// GetMeta returns 'file descriptor' stored in meta. The 'file descriptor'
	// can be updated using SetMeta method.
	// Returns os.ErrNotExist if meta doesn't store any 'file descriptor', or
	// 'file descriptor' point to nonexistent file.
	GetMeta() (storage.FileDesc, error)

	// List returns file descriptors that match the given file types.
	// The file types may be OR'ed together.
	List(ft storage.FileType) ([]storage.FileDesc, error)

	// Open opens file with the given 'file descriptor' read-only.
	// Returns os.ErrNotExist error if the file does not exist.
	// Returns ErrClosed if the underlying storage is closed.
	Open(fd storage.FileDesc) (storage.Reader, error)

	// Close closes the storage.
	// It is valid to call Close multiple times. Other methods should not be
	// called after the storage has been closed.
	Close() error
}

type Storage interface {
	ReadOnlyStorage

	// Log logs a string. This is used for logging.
	// An implementation may write to a file, stdout or simply do nothing.
	Log(str string)

	// SetMeta store 'file descriptor' that can later be acquired using GetMeta
	// method. The 'file descriptor' should point to a valid file.
	// SetMeta should be implemented in such way that changes should happen
	// atomically.
	SetMeta(fd storage.FileDesc) error

	// Create creates file with the given 'file descriptor', truncate if already
	// exist and opens write-only.
	// Returns ErrClosed if the underlying storage is closed.
	Create(fd storage.FileDesc) (storage.Writer, error)

	// Remove removes file with the given 'file descriptor'.
	// Returns ErrClosed if the underlying storage is closed.
	Remove(fd storage.FileDesc) error

	// Rename renames file from oldfd to newfd.
	// Returns ErrClosed if the underlying storage is closed.
	Rename(oldfd, newfd storage.FileDesc) error
}
