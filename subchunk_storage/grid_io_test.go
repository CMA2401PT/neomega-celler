package subchunk_storage

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/low_io"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestGridIO(t *testing.T) {
	// 随机数是难以压缩的，所以我们这边先不考虑压缩
	newGrid := func() GridSubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]], celler.SubChunkDataChange[uint32, uint32]] {
		return NewLazySimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	}
	storage := low_io.NewSimpleGridIO(low_io.NoCompress{}, 2*(64*64)/(4*4))
	s := NewIOGrid(newGrid, 2, 2, (64 * 64), storage)
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
		X := celler.ChunkUnit(r.Int() % 250) // 4096=250*16
		Z := celler.ChunkUnit(r.Int() % 250)
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
	for i := 0; i < elementCounts; i++ {
		pos := poses[i]
		chunk := chunks[i]
		buf := bytes.NewBuffer(nil)
		chunk.Dump(buf)
		orig := buf.Bytes()
		testDatas[pos] = orig
	}
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()
	for i := 0; i < elementCounts; i++ {
		pos := poses[i]
		chunk := chunks[i]
		s.Set(pos, chunk)
	}
	s.DumpAll()
	fmt.Printf("size: %v\n", storage.TotalSize()/(1024*1024))

	poses = make([]celler.SubChunkPos, 0)
	for p := range testDatas {
		poses = append(poses, p)
	}
	sort.Slice(poses, func(i, j int) bool {
		return poses[i].Less(poses[j])
	})
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()

	// storage := low_io.NewSimpleMemoryIOToBytes(nil)
	// s.DumpData(storage)
	// storage.Close()
	// storageBack := low_io.NewSimpleMemoryIOToBytes(storage.Bytes())
	// back := NewSimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	// back.LoadData(storageBack)
	count := 0
	for _, pos := range poses {
		d := testDatas[pos]
		count += 1
		stor := s.Get(pos)
		if stor == nil {
			t.Errorf("should get sth but nil @ %v, %v", pos, count)
			continue
		}
		buf := bytes.NewBuffer(nil)
		stor.Dump(buf)
		ret := buf.Bytes()
		if bytes.Compare(ret, d) != 0 {
			t.Errorf("should euqal but not @ %v, %v", pos, count)
		}
	}
	count = 0
	for pos := range s.IterSubChunks {
		if _, found := testDatas[pos]; !found {
			t.Errorf("should get nth but nil @ %v", pos)
		}
		count += 1
	}
	if count != len(testDatas) {
		t.Error("count mismatch")
	}
	fmt.Println(time.Now().Sub(st).Seconds())
}
