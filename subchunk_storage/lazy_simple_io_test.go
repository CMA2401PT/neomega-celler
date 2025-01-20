package subchunk_storage

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/low_io"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestLazySimpleIO(t *testing.T) {
	s := NewLazySimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	for p, s := range s.IterSubChunks {
		t.Errorf("should get nothing but %v %v", p, s)
	}
	if s.Get(celler.SubChunkPos{}) != nil {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	testDatas := map[celler.SubChunkPos][]byte{}
	r := rand.New(rand.NewSource(2401))
	elementCounts := 32 * 32 * 32
	poses := make([]celler.SubChunkPos, 0, elementCounts)
	chunks := make([]*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	], 0, elementCounts)
	st := time.Now()
	for i := 0; i < elementCounts; i++ {
		X := celler.ChunkUnit(r.Int() % 1000)
		Z := celler.ChunkUnit(r.Int() % 1000)
		Y := celler.ChunkUnit(r.Int() % 32)
		var chunk *subchunk.Versioned[uint32, uint32,
			*layer_storage.Palette[uint32],
			*layer_storage.Palette[uint32],
			*layer_storage.Map[celler.SNBT],
		]
		if i < 100 {
			chunk = MakeRandomVersionedSubChunk[uint32, uint32](r)
		} else {
			chunk = chunks[i%(100)]
		}

		pos := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: X, Z: Z}, Y: Y}
		chunks = append(chunks, chunk)
		poses = append(poses, pos)
	}
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()
	for i := 0; i < elementCounts; i++ {
		pos := poses[i]
		chunk := chunks[i]
		buf := bytes.NewBuffer(nil)
		chunk.Dump(buf)
		testDatas[pos] = buf.Bytes()
		s.Set(pos, chunk)
	}

	storage := low_io.NewSimpleMemoryIOToBytes(nil)
	s.DumpData(storage)
	storage.Close()
	storageBack := low_io.NewSimpleMemoryIOToBytes(storage.Bytes())
	back := NewLazySimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()
	back.LoadData(storageBack)
	count := 0
	for pos := range s.IterSubChunks {
		if _, found := testDatas[pos]; !found {
			t.Errorf("should get nth but nil @ %v", pos)
		}
		count += 1
	}
	if count != len(testDatas) {
		t.Error("count mismatch")
	}
	for pos, d := range testDatas {
		stor := back.Get(pos)
		if stor == nil {
			t.Errorf("should get sth but nil @ %v", pos)
		}
		buf := bytes.NewBuffer(nil)
		stor.Dump(buf)
		ret := buf.Bytes()
		if bytes.Compare(ret, d) != 0 {
			t.Errorf("should euqal but not @ %v", pos)
		}
	}

	fmt.Println(time.Now().Sub(st).Seconds())
}
