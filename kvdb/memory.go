package kvdb

type MemoryKVDB struct {
	data map[string][]byte
}

func NewMockKVDB() *MemoryKVDB {
	return &MemoryKVDB{
		data: make(map[string][]byte),
	}
}

func (db *MemoryKVDB) Close() error { return nil }

func (db *MemoryKVDB) Put(key []byte, value []byte) error {
	db.data[string(key)] = value
	return nil
}

func (db *MemoryKVDB) Get(key []byte) (value []byte, err error) {
	value, exists := db.data[string(key)]
	if !exists {
		return nil, nil // simulate key not found
	}
	return value, nil
}

func (db *MemoryKVDB) Delete(key []byte) error {
	delete(db.data, string(key))
	return nil
}

func (db *MemoryKVDB) IterAll(interFn func(CanKV) bool) {
	for key, value := range db.data {
		if !interFn(&memoryCanKV{key: []byte(key), value: value}) {
			break
		}
	}
}

type memoryCanKV struct {
	key   []byte
	value []byte
}

func (kv *memoryCanKV) Key() []byte {
	return kv.key
}

func (kv *memoryCanKV) Value() []byte {
	return kv.value
}
