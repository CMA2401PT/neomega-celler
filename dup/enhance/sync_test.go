package enhance

import (
	"celler"
	"celler/subchunk"
	"celler/subchunk_storage/simple"
	"math/rand"
	"testing"
)

func TestSyncer(t *testing.T) {
	basic := simple.NewSimpleMemory(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	s := MakeBiDirSyncer[celler.SubChunkDataChange[uint32, uint32]](basic)

	for p, s := range s.IterSubChunks {
		t.Errorf("should get nothing but %v %v", p, s)
	}
	if s.Get(celler.SubChunkPos{}) != nil {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	tp := celler.ToSubChunkPos(1, 2, 3)
	r := rand.New(rand.NewSource(2401))
	s.Set(tp, MakeRandomVersionedSubChunk[uint32, uint32](r))
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

	basic2 := simple.NewSimpleMemory(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	d := MakeBiDirSyncer[celler.SubChunkDataChange[uint32, uint32]](basic2)
	currentV := d.GetVersion(tp)
	if currentV != 0 {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	empty := simple.NewSingle(uint32(100), uint32(200), subchunk.MakeStandardSimple)
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
