package kvdb

type CanKV interface {
	Key() []byte
	Value() []byte
}

type ReadOnlyKVDB interface {
	Get(key []byte) (value []byte, err error)
	IterAll(interFn func(CanKV) (continueIter bool))
	Close() error
}

type KVDB interface {
	ReadOnlyKVDB
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

type readOnlyKVWrapper[RODB ReadOnlyKVDB] struct {
	DB RODB
}

func (w readOnlyKVWrapper[RODB]) Put(key []byte, value []byte) error {
	return nil
}

func (w readOnlyKVWrapper[RODB]) Delete(key []byte) error {
	return nil
}

func (w readOnlyKVWrapper[RODB]) Get(key []byte) (value []byte, err error) {
	return w.DB.Get(key)
}

func (w readOnlyKVWrapper[RODB]) IterAll(interFn func(CanKV) (continueIter bool)) {
	w.DB.IterAll(interFn)
}

func (w readOnlyKVWrapper[RODB]) Close() error {
	return w.DB.Close()
}

func ReadOnlyAsReadWrite[RODB ReadOnlyKVDB](rodb RODB) KVDB {
	return &readOnlyKVWrapper[RODB]{
		DB: rodb,
	}
}
