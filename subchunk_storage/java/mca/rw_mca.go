package mca

import (
	"celler"
	"fmt"
	"sync"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

var _ ReadWriteMCAChunkKV = &MCARwKV{}

type MCARwKV struct {
	RwStor      ReadWriteStor
	regionCache *simplelru.LRU[RegionPos, *RwRegion]
	mu          sync.RWMutex
}

func NewReadWriteMCA(rwStor ReadWriteStor) *MCARwKV {
	cache, err := simplelru.NewLRU(16, func(rp RegionPos, rr *RwRegion) {
		rr.Close()
	})
	if err != nil {
		panic(err)
	}
	return &MCARwKV{
		RwStor:      rwStor,
		mu:          sync.RWMutex{},
		regionCache: cache,
	}
}

func (db *MCARwKV) getRegion(rp RegionPos) (r *RwRegion) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if r, exist := db.regionCache.Get(rp); exist {
		return r
	}
	defer func() {
		db.regionCache.Add(rp, r)
	}()
	fmt.Println("open region", rp)
	rd, err := db.RwStor.OpenRegion(rp)
	if err != nil {
		return
	}
	r, err = newRwRegion(rd)
	if err != nil {
		return
	}
	return r
}

func (db *MCARwKV) Close() error {
	for _, rg := range db.regionCache.Values() {
		rg.Close()
	}
	return db.RwStor.Close()
}

func (db *MCARwKV) IterAll(interFn func(celler.ChunkPos) (continueIter bool)) {
	for rp := range db.RwStor.IterRegions {
		rd, err := db.RwStor.OpenRegion(rp)
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

func (db *MCARwKV) Get(chunkPos celler.ChunkPos) (value []byte, err error) {
	rp, irp := ChunkPosToRegionPos(chunkPos)
	rg := db.getRegion(rp)
	if rg == nil {
		return nil, ErrNoData
	}
	return rg.getChunkData(irp)
}

func (db *MCARwKV) Put(chunkPos celler.ChunkPos, value []byte) (err error) {
	rp, irp := ChunkPosToRegionPos(chunkPos)
	rg := db.getRegion(rp)
	if rg == nil {
		return ErrNoData
	}
	return rg.setChunkData(irp, value)
}

func NewReadWriteMCAKvFromDir(dirPath string) (ReadWriteMCAChunkKV, error) {
	stor, err := OpenDirReadWrite(dirPath)
	if err != nil {
		return nil, err
	}
	cache, _ := simplelru.NewLRU(16, func(rp RegionPos, rg *RwRegion) {
		rg.Close()
	})
	return &MCARwKV{
		RwStor:      stor,
		regionCache: cache,
		mu:          sync.RWMutex{},
	}, nil
}
