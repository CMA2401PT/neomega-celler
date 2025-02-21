package test

import (
	"bytes"
	"celler/kvdb"
	"math/rand"
	"testing"
)

// Helper function to generate random data
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// 通用的测试函数
func testKVDB(t *testing.T, impl kvdb.KVDB) {
	// 创建一个 mockKVDB 用来同步验证
	mockKVDB := kvdb.NewMockKVDB()
	numTests := 10000
	t.Run("Random Put and Get", func(t *testing.T) {
		// Insert random data and verify retrieval
		for i := 0; i < numTests; i++ {
			key := randomString(20)
			value := randomString(20)

			// 对 mockKVDB 执行操作
			err := mockKVDB.Put([]byte(key), []byte(value))
			if err != nil {
				t.Errorf("Put failed for key %s in mockKVDB: %v", key, err)
			}

			// 对 impl 执行相同操作
			err = impl.Put([]byte(key), []byte(value))
			if err != nil {
				t.Errorf("Put failed for key %s in impl KVDB: %v", key, err)
			}
			back, err := impl.Get([]byte(key))
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal([]byte(value), back) {
				t.Errorf("not equal %v!=%v", string(value), string(back))
			}
		}
		// 检查 mockKVDB 和 impl KVDB 是否一致
		verifyConsistency(t, mockKVDB, impl)
	})

	t.Run("Random Delete", func(t *testing.T) {
		// Insert some data first
		for i := 0; i < numTests/10; i++ {
			key := randomString(20)
			value := randomString(20)
			mockKVDB.Put([]byte(key), []byte(value))
			impl.Put([]byte(key), []byte(value))
		}

		// Randomly delete keys from both
		for i := 0; i < 30; i++ {
			key := randomString(10)

			// 对 mockKVDB 和 impl 执行删除操作
			err := mockKVDB.Delete([]byte(key))
			if err != nil {
				t.Errorf("Delete failed for key %s in mockKVDB: %v", key, err)
			}

			err = impl.Delete([]byte(key))
			if err != nil {
				t.Errorf("Delete failed for key %s in impl KVDB: %v", key, err)
			}

			// 验证一致性
			verifyConsistency(t, mockKVDB, impl)
		}
	})

	t.Run("Random IterAll", func(t *testing.T) {
		// Insert random data first
		for i := 0; i < numTests; i++ {
			key := randomString(20)
			value := randomString(20)
			mockKVDB.Put([]byte(key), []byte(value))
			impl.Put([]byte(key), []byte(value))
		}

		// Iterate and verify both databases
		mockKeys := make(map[string][]byte)
		mockKVDB.IterAll(func(kv kvdb.CanKV) bool {
			mockKeys[string(kv.Key())] = kv.Value()
			return true
		})

		implKeys := make(map[string][]byte)
		impl.IterAll(func(kv kvdb.CanKV) bool {
			implKeys[string(kv.Key())] = kv.Value()
			return true
		})

		// 验证一致性
		if !mapEqual(mockKeys, implKeys) {
			t.Errorf("Iteration results are inconsistent between mockKVDB and impl KVDB")
		}
	})

	t.Run("Random Put and Overwrite", func(t *testing.T) {
		// Insert random data and overwrite
		for i := 0; i < numTests; i++ {
			key := randomString(20)
			value := randomString(20)

			// First insertion for mockKVDB
			err := mockKVDB.Put([]byte(key), []byte(value))
			if err != nil {
				t.Errorf("Put failed for key %s in mockKVDB: %v", key, err)
			}

			// First insertion for impl
			err = impl.Put([]byte(key), []byte(value))
			if err != nil {
				t.Errorf("Put failed for key %s in impl KVDB: %v", key, err)
			}

			// Overwrite with new value for mockKVDB
			newValue := randomString(20)
			err = mockKVDB.Put([]byte(key), []byte(newValue))
			if err != nil {
				t.Errorf("Put failed while overwriting key %s in mockKVDB: %v", key, err)
			}

			// Overwrite with new value for impl
			err = impl.Put([]byte(key), []byte(newValue))
			if err != nil {
				t.Errorf("Put failed while overwriting key %s in impl KVDB: %v", key, err)
			}
		}
		// Verify consistency
		verifyConsistency(t, mockKVDB, impl)
	})
}

// 验证 mockKVDB 和 impl KVDB 的一致性
func verifyConsistency(t *testing.T, mockKVDB, impl kvdb.KVDB) {
	// 检查所有 key 是否一致
	mockData := getAllData(mockKVDB)
	implData := getAllData(impl)

	if !mapEqual(mockData, implData) {
		t.Errorf("Databases are inconsistent. MockKVDB data: %+v, ImplKVDB data: %+v", mockData, implData)
	}
}

// 获取 KVDB 中的所有数据
func getAllData(db kvdb.KVDB) map[string][]byte {
	data := make(map[string][]byte)
	db.IterAll(func(kv kvdb.CanKV) bool {
		data[string(kv.Key())] = kv.Value()
		return true
	})
	return data
}

// 比较两个 map 是否相等
func mapEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valueA := range a {
		valueB, exists := b[key]
		if !exists || !bytes.Equal(valueA, valueB) {
			// fmt.Printf("not equal %v\n", key)
			return false
		}
	}
	return true
}
