package enhance

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage/grid"
	"celler/subchunk_storage/low_io"
	"celler/subchunk_storage/simple"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	newGrid := func() grid.GridSubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]]] {
		return simple.NewLazySimple(subchunk.StandardVersionedFactory(uint32(100), uint32(200)))
	}
	storage := low_io.NewSimpleGridIO(low_io.NoCompress{}, (64 * 64))
	base := grid.NewIOGrid(newGrid, 2, 2, 2, (64 * 64 * 32), storage)
	chunk_storage := NewBatchOperator(base)

	elementCounts := 32 * 32 * 32
	poses := make([]celler.SubChunkPos, 0, elementCounts)
	chunks := make([]*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	], 0, elementCounts)
	r := rand.New(rand.NewSource(2401))
	testDatas := map[celler.SubChunkPos][]byte{}
	writeOperations := map[celler.SubChunkPos]*subchunk.Versioned[uint32, uint32,
		*layer_storage.Palette[uint32],
		*layer_storage.Palette[uint32],
		*layer_storage.Map[celler.SNBT],
	]{}
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
	for i := 0; i < elementCounts; i++ {
		pos := poses[i]
		chunk := chunks[i]
		buf := bytes.NewBuffer(nil)
		chunk.Dump(buf)
		testDatas[pos] = buf.Bytes()
		writeOperations[pos] = chunk
	}

	st := time.Now()
	chunk_storage.BatchSet(writeOperations)
	fmt.Println("set time:", time.Since(st).Seconds())

	targetPoses := make([]celler.SubChunkPos, 0, len(testDatas))
	for p := range testDatas {
		targetPoses = append(targetPoses, p)
	}
	st = time.Now()
	back := chunk_storage.BatchGet(targetPoses)
	fmt.Println("get time:", time.Since(st).Seconds())
	if len(back) != len(testDatas) {
		t.Errorf("items not equal: %v != %v", len(back), len(testDatas))
	}
	for _, v := range back {
		std := testDatas[v.Pos]
		stor := v.Data
		buf := bytes.NewBuffer(nil)
		stor.Dump(buf)
		ret := buf.Bytes()
		if bytes.Compare(ret, std) != 0 {
			t.Errorf("should euqal but not @ %v", v.Pos)
		}
	}
}
