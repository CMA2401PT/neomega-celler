package mca

import (
	"celler"
	"fmt"
	"sync"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

var _ ReadOnlyMCAChunkKV = &MCARoKV{}

type MCARoKV struct {
	RoStor      ReadOnlyStor
	regionCache *simplelru.LRU[RegionPos, *RoRegion]
	mu          sync.RWMutex
}

func NewReadOnlyMCA(roStor ReadOnlyStor) *MCARoKV {
	cache, err := simplelru.NewLRU(16, func(rp RegionPos, rr *RoRegion) {
		rr.Close()
	})
	if err != nil {
		panic(err)
	}
	return &MCARoKV{
		RoStor:      roStor,
		mu:          sync.RWMutex{},
		regionCache: cache,
	}
}

func (db *MCARoKV) getRegion(rp RegionPos) (r *RoRegion) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if r, exist := db.regionCache.Get(rp); exist {
		return r
	}
	defer func() {
		db.regionCache.Add(rp, r)
	}()
	fmt.Println("open region", rp)
	rd, err := db.RoStor.OpenRegion(rp)
	if err != nil {
		return
	}
	r, err = newRoRegion(rd)
	if err != nil {
		return
	}
	return r
}

func (db *MCARoKV) Close() error {
	for _, rg := range db.regionCache.Values() {
		rg.Close()
	}
	return db.RoStor.Close()
}

func (db *MCARoKV) IterAll(interFn func(celler.ChunkPos) (continueIter bool)) {
	for rp := range db.RoStor.IterRegions {
		rd, err := db.RoStor.OpenRegion(rp)
		if rd == nil || err != nil {
			continue
		}
		rg, err := newRoRegion(rd)
		if rg == nil || err != nil {
			continue
		}
		for x := uint8(0); x < 32; x++ {
			for z := uint8(0); z < 32; z++ {
				space := rg.chunkDataLocation[x][z]
				start, count := rg.decodeSpace(space)
				if start == 0 || count == 0 {
					continue
				}
				irp := InRegionPos{
					Cx: x, Cz: z,
				}
				chunkPos := RegionPosToChunkPos(rp, irp)
				if !interFn(chunkPos) {
					return
				}
			}
		}
	}
}

func (db *MCARoKV) Get(chunkPos celler.ChunkPos) (value []byte, err error) {
	rp, irp := ChunkPosToRegionPos(chunkPos)
	rg := db.getRegion(rp)
	if rg == nil {
		return nil, ErrNoData
	}
	return rg.getChunkData(irp)
}

func NewReadOnlyMCAKvFromDir(dirPath string) (ReadOnlyMCAChunkKV, error) {
	stor, err := OpenDirReadOnly(dirPath)
	if err != nil {
		return nil, err
	}
	cache, _ := simplelru.NewLRU(16, func(rp RegionPos, rg *RoRegion) {
		rg.Close()
	})
	return &MCARoKV{
		RoStor:      stor,
		regionCache: cache,
		mu:          sync.RWMutex{},
	}, nil
}
