package enhance

import (
	"celler"
	"celler/layer_storage"
	"celler/subchunk"
	"celler/subchunk_storage"
	"fmt"
	"math/rand"
	"testing"
)

func TestSyncer(t *testing.T) {
	basic := subchunk_storage.NewSimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
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

	basic2 := subchunk_storage.NewSimple(uint32(100), uint32(200), subchunk.MakeStandardVersioned)
	d := MakeBiDirSyncer[celler.SubChunkDataChange[uint32, uint32]](basic2)
	currentV := d.GetVersion(tp)
	if currentV != 0 {
		t.Errorf("should get nothing but: %v", s.Get(celler.SubChunkPos{}))
	}
	empty := subchunk_storage.NewSingle(uint32(100), uint32(200), subchunk.MakeStandardSimple)
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
