package subchunk_storage

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func MakeRandomVersionedSubChunk[BlockIDT, AttachT ~uint32](r *rand.Rand) *subchunk.Versioned[BlockIDT, AttachT,
	*layer_storage.Palette[BlockIDT],
	*layer_storage.Palette[AttachT],
	*layer_storage.Map[celler.SNBT],
] {

	c := subchunk.MakeStandardVersioned[BlockIDT, AttachT](100, 200)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				bockID := BlockIDT(r.Int()) % 8
				attach := AttachT(r.Int()) % 4
				snbt := celler.SNBT("")
				if r.Int()%256 == 0 {
					snbt = celler.SNBT(fmt.Sprintf("%v", r.Int()))
				}
				c.ChangeBlock((celler.SubChunkBlockDelta[BlockIDT, AttachT]{}).
					SetPos(celler.ToSubChunkBlockPos(x, y, z)).
					SetID(bockID).
					SetAttach(attach).
					SetSNBT(snbt),
				)
			}
		}
	}
	return c
}

func testStorageEmpty[BlockIDT, AttachT ~uint32](
	t *testing.T,
	maker func() celler.SubChunkStorage[*subchunk.Versioned[BlockIDT, AttachT,
		*layer_storage.Palette[BlockIDT],
		*layer_storage.Palette[AttachT],
		*layer_storage.Map[celler.SNBT],
	], celler.SubChunkDataChange[BlockIDT, AttachT]]) {
	s := maker()
	for p, s := range s.IterSubChunks {
		t.Errorf("should get nothing but %v %v", p, s)
	}
	if s.Get(celler.SubChunkPos{}) != nil {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	tp := celler.ToSubChunkPos(1, 2, 3)
	r := rand.New(rand.NewSource(2401))
	s.Set(tp, MakeRandomVersionedSubChunk[BlockIDT, AttachT](r))
	if s.Get(tp) == nil {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}
	if s.Get(celler.SubChunkPos{}) != nil {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}

	if s.GetVersion(tp) == 0 {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}
	if s.GetVersion(celler.SubChunkPos{}) != 0 {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	count := 0
	for _, s := range s.IterSubChunks {
		count += 1
		if s.BlockAttachStorage == nil {
			t.Errorf("should have data but nothing")
		}
	}
	if count == 0 {
		t.Errorf("should get 1 but nothing")
	}

	d := maker()
	currentV := d.GetVersion(tp)
	if currentV != 0 {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	empty := NewSingle(BlockIDT(100), AttachT(200), subchunk.MakeStandardSimple)
	change := empty.QueryChange(tp, currentV)
	d.ApplyChange(tp, change)
	if currentV != 0 {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}

	change = s.QueryChange(tp, currentV)
	d.ApplyChange(tp, change)
	if d.Get(tp) == nil {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}

	currentV = d.GetVersion(tp)
	if currentV == 0 {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}
	change = s.QueryChange(tp, currentV)
	d.ApplyChange(tp, change)
	if d.Get(tp) == nil {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}

	currentV = d.GetVersion(tp)
	if currentV == 0 {
		t.Errorf("should get sth but: %v", s.Get(celler.SubChunkPos{}))
	}
	change = empty.QueryChange(tp, currentV)
	d.ApplyChange(tp, change)
	if d.Get(tp) != nil {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}

}

func testStorage[BlockIDT, AttachT ~uint32](
	t *testing.T,
	maker func() celler.SubChunkStorage[*subchunk.Versioned[BlockIDT, AttachT,
		*layer_storage.Palette[BlockIDT],
		*layer_storage.Palette[AttachT],
		*layer_storage.Map[celler.SNBT],
	], celler.SubChunkDataChange[BlockIDT, AttachT]]) {
	testStorageEmpty(t, maker)
	s := maker()
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
	chunks := make([]*subchunk.Versioned[BlockIDT, AttachT,
		*layer_storage.Palette[BlockIDT],
		*layer_storage.Palette[AttachT],
		*layer_storage.Map[celler.SNBT],
	], 0, elementCounts)
	st := time.Now()
	for i := 0; i < elementCounts; i++ {
		X := celler.ChunkUnit(r.Int() % 1000)
		Z := celler.ChunkUnit(r.Int() % 1000)
		Y := celler.ChunkUnit(r.Int() % 32)
		var chunk *subchunk.Versioned[BlockIDT, AttachT,
			*layer_storage.Palette[BlockIDT],
			*layer_storage.Palette[AttachT],
			*layer_storage.Map[celler.SNBT],
		]
		if i < 100 {
			chunk = MakeRandomVersionedSubChunk[BlockIDT, AttachT](r)
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
	}
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()
	for i := 0; i < elementCounts; i++ {
		pos := poses[i]
		chunk := chunks[i]
		s.Set(pos, chunk)
	}
	fmt.Println(time.Now().Sub(st).Seconds())
	st = time.Now()

	for pos, d := range testDatas {
		stor := s.Get(pos)
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
	fmt.Println(time.Now().Sub(st).Seconds())
}

func TestSimple(t *testing.T) {
	testStorage(t, func() celler.SubChunkStorage[*subchunk.Versioned[uint32, uint32, *layer_storage.Palette[uint32], *layer_storage.Palette[uint32], *layer_storage.Map[celler.SNBT]], celler.SubChunkDataChange[uint32, uint32]] {
		return NewSimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	})
}
