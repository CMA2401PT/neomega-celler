package subchunk_storage

import (
	"celler"
	"celler/subchunk_storage/low_io"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type GridSubChunkStorage[Data any, Change any] interface {
	celler.SubChunkStorage[Data, Change]
	IsEmpty() bool
	LoadData(low_io.SubChunkDataReadCloser)
	DumpData(low_io.SubChunkDataWriteCloser)
}

// 不完全支持随机访问，可能导致读/写放大到 Grid 倍
type IOGrid[Data any, Change any, DataPtr StandardSubChunk[Data, Change]] struct {
	Data                   map[GridPos]GridSubChunkStorage[DataPtr, Change]
	GridXLevel, GridZLevel uint32
	newGrid                func() GridSubChunkStorage[DataPtr, Change]
	ActivateGrid           *simplelru.LRU[GridPos, struct{}]
	Storage                low_io.GridStorage
	hasLast                bool
	lastPos                GridPos
	lastPtr                GridSubChunkStorage[DataPtr, Change]
}

func NewIOGrid[Data any, Change any, DataPtr StandardSubChunk[Data, Change]](
	makeNew func() GridSubChunkStorage[DataPtr, Change],
	gridXLevel uint32, gridZLevel uint32,
	maxActivateChunks int,
	storage low_io.GridStorage,
) *IOGrid[Data, Change, DataPtr] {
	s := &IOGrid[Data, Change, DataPtr]{
		Data:       make(map[GridPos]GridSubChunkStorage[DataPtr, Change]),
		newGrid:    makeNew,
		GridXLevel: gridXLevel,
		GridZLevel: gridZLevel,
		Storage:    storage,
	}
	maxActivateGrids := maxActivateChunks >> gridXLevel
	maxActivateGrids = maxActivateGrids >> gridZLevel
	activateGrid, _ := simplelru.NewLRU[GridPos, struct{}](maxActivateGrids, func(pos GridPos, value struct{}) {
		s.swapOutGrid(pos)
	})
	s.ActivateGrid = activateGrid
	return s
}

func (s *IOGrid[Data, Change, DataPtr]) InsightFn() func(celler.ChunkPos) (hash int) {
	return func(pos celler.ChunkPos) (hash int) {
		return (int(pos.Z)>>s.GridZLevel)<<s.GridXLevel + (int(pos.X) >> s.GridXLevel)
	}
}

func (s *IOGrid[Data, Change, DataPtr]) tearPos(pos celler.SubChunkPos) (GridPos, celler.SubChunkPos) {
	return GridPos{
			X: int32(pos.X) >> s.GridXLevel,
			Z: int32(pos.Z) >> s.GridZLevel,
		}, celler.SubChunkPos{
			ChunkPos: celler.ChunkPos{
				X: pos.X % celler.ChunkUnit(1<<s.GridXLevel),
				Z: pos.Z % celler.ChunkUnit(1<<s.GridZLevel),
			},
			Y: pos.Y,
		}
}

func (s *IOGrid[Data, Change, DataPtr]) assamblePos(gp GridPos, cp celler.SubChunkPos) celler.SubChunkPos {
	baseX := celler.ChunkUnit(gp.X) << s.GridXLevel
	baseZ := celler.ChunkUnit(gp.Z) << s.GridZLevel
	return celler.SubChunkPos{
		ChunkPos: celler.ChunkPos{X: baseX + cp.X, Z: baseZ + cp.Z},
		Y:        cp.Y,
	}
}

func (s *IOGrid[Data, Change, DataPtr]) toStorageKey(gp GridPos) uint64 {
	return uint64(gp.X)<<32 | uint64(gp.Z)
}

func (s *IOGrid[Data, Change, DataPtr]) fromStorageKey(key uint64) GridPos {
	return GridPos{X: int32(key >> 32), Z: int32(key)}
}

// 从底层存储中读取目标 grid
func (s *IOGrid[Data, Change, DataPtr]) loadGrid(gp GridPos) GridSubChunkStorage[DataPtr, Change] {
	reader := s.Storage.DataReader(s.toStorageKey(gp))
	if reader == nil {
		return nil
	}
	emptyGrid := s.newGrid()
	emptyGrid.LoadData(reader)
	return emptyGrid
}

// 如果没有加载 grid， 则从底层存储中读取目标 grid, 如果底层也没有，则返回空
func (s *IOGrid[Data, Change, DataPtr]) ensureGridLoaded(gp GridPos) (grid GridSubChunkStorage[DataPtr, Change]) {
	if s.hasLast && s.lastPos == gp && s.lastPtr != nil {
		return s.lastPtr
	}
	defer func() {
		s.hasLast = true
		s.lastPos = gp
		s.lastPtr = grid
	}()
	if _, found := s.ActivateGrid.Get(gp); found {
		grid = s.Data[gp]
		return
	}
	grid = s.loadGrid(gp)
	s.ActivateGrid.Add(gp, struct{}{})
	if grid == nil {
		return nil
	}
	s.Data[gp] = grid

	return grid
}

// 确保 grid 存在以供写入，若不活跃且底层存储也没有这个数据，新建一个空 grid
func (s *IOGrid[Data, Change, DataPtr]) ensureGridForWrite(gp GridPos) (grid GridSubChunkStorage[DataPtr, Change]) {
	if s.hasLast && s.lastPos == gp && s.lastPtr != nil {
		return s.lastPtr
	}
	defer func() {
		s.hasLast = true
		s.lastPos = gp
		s.lastPtr = grid
	}()
	if _, found := s.ActivateGrid.Get(gp); found {
		grid = s.Data[gp]
		if grid == nil {
			grid = s.newGrid()
			s.Data[gp] = grid
		}
		return
	}
	s.ActivateGrid.Add(gp, struct{}{})
	grid = s.loadGrid(gp)
	if grid != nil {
		s.Data[gp] = grid
		return
	}
	grid = s.newGrid()
	s.Data[gp] = grid
	return
}

// 将一个 grid 变为非活跃状态并 swap out
func (s *IOGrid[Data, Change, DataPtr]) swapOutGrid(gd GridPos) {
	grid := s.Data[gd]
	s.ActivateGrid.Remove(gd)
	delete(s.Data, gd)
	if grid != nil {
		grid.DumpData(s.Storage.DataWriter(s.toStorageKey(gd)))
	} else {
		s.Storage.DropGrid(s.toStorageKey(gd))
	}
	if s.lastPos == gd {
		s.hasLast = false
	}
}

// 将所有活跃的grid保存到底层但是保持激活状态
func (s *IOGrid[Data, Change, DataPtr]) DumpAll() {
	for _, gd := range s.ActivateGrid.Keys() {
		grid := s.Data[gd]
		if grid != nil {
			grid.DumpData(s.Storage.DataWriter(s.toStorageKey(gd)))
		} else {
			s.Storage.DropGrid(s.toStorageKey(gd))
		}
	}
}

func (s *IOGrid[Data, Change, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil {
		return
	}
	if s.Data != nil {
		for gp, gd := range s.Data {
			for sp, d := range gd.IterSubChunks {
				finalPos := s.assamblePos(gp, sp)
				if !cb(finalPos, d) {
					return
				}
			}
		}
	}
	for key := range s.Storage.IterGrid {
		gp := s.fromStorageKey(key)
		if _, found := s.ActivateGrid.Get(gp); found {
			continue
		}
		tmpGrid := s.loadGrid(gp)
		for sp, d := range tmpGrid.IterSubChunks {
			finalPos := s.assamblePos(gp, sp)
			if !cb(finalPos, d) {
				return
			}
		}
	}
}

func (s *IOGrid[Data, Change, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		if data == nil {
			return
		} else {
			s.ensureGridForWrite(gp).Set(cp, data)
			return
		}
	} else {
		if data == nil {
			grid.Set(cp, data)
			if grid.IsEmpty() {
				delete(s.Data, gp)
				s.hasLast = false
			}
		} else {
			grid.Set(cp, data)
		}
	}
}

func (s *IOGrid[Data, Change, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		return nil
	}
	return grid.Get(cp)
}

func (s *IOGrid[Data, Change, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		if version == 0 {
			return celler.SubChunkChange[Change]{
				Valid: true,
			}
		} else {
			return celler.SubChunkChange[Change]{
				Valid: false,
				Drop:  true,
			}
		}
	}
	return grid.QueryChange(cp, version)
}

func (s *IOGrid[Data, Change, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		return 0
	}
	return grid.GetVersion(cp)
}

func (s *IOGrid[Data, Change, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	gp, cp := s.tearPos(pos)
	if change.Drop {
		grid := s.ensureGridLoaded(gp)
		if grid != nil {
			grid.ApplyChange(cp, change)
			if grid.IsEmpty() {
				delete(s.Data, gp)
				s.hasLast = false
			}
		}

		return
	}
	s.ensureGridForWrite(gp).ApplyChange(cp, change)
}
