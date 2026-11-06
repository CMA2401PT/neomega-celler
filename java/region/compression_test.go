package region

import (
	"bytes"
	"sync"
	"testing"
)

// 压缩器归还池后，旧记录仍必须持有自己的字节，且并发任务不能串用字典。
func TestCompressConcurrentRecordsRemainIndependent(t *testing.T) {
	var group sync.WaitGroup
	for worker := range 8 {
		group.Go(func() {
			var records []Record
			var inputs [][]byte
			for i := range 20 {
				data := bytes.Repeat([]byte{byte(worker), byte(i), 0xff}, 1024+i)
				record, err := Compress(data, uint32(i))
				if err != nil {
					t.Error(err)
					return
				}
				records = append(records, record)
				inputs = append(inputs, data)
			}
			for i, record := range records {
				got, err := record.Decode()
				if err != nil || !bytes.Equal(got, inputs[i]) || record.Timestamp != uint32(i) {
					t.Errorf("worker=%d record=%d: %v", worker, i, err)
				}
			}
		})
	}
	group.Wait()
}
