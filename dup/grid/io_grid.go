package grid

import (
	"celler"
	"celler/subchunk_storage/low_io"
	"celler/subchunk_storage/simple"
	"fmt"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

type GridSubChunkStorage[Data any] interface {
	celler.SubChunkStorage[Data]
	IsEmpty() bool
	LoadData(low_io.SubChunkDataReadCloser)
	DumpData(low_io.SubChunkDataWriteCloser)
}

// 不完全支持随机访问，可能导致读/写放大到 Grid 倍
type IOGrid[Data any, DataPtr simple.BasicSubchunk[Data]] struct {
	Data                               map[low_io.GridPos]GridSubChunkStorage[DataPtr]
	GridXLevel, GridYLevel, GridZLevel uint32
	newGrid                            func() GridSubChunkStorage[DataPtr]
	ActivateGrid                       *simplelru.LRU[low_io.GridPos, struct{}]
	Storage                            low_io.GridStorage
	hasLast                            bool
	lastPos                            low_io.GridPos
	lastPtr                            GridSubChunkStorage[DataPtr]
}

func NewIOGrid[Data any, DataPtr simple.BasicSubchunk[Data]](
	makeNew func() GridSubChunkStorage[DataPtr],
	gridXLevel uint32, gridYLevel uint32, gridZLevel uint32,
	maxActivateSubChunks int,
	storage low_io.GridStorage,
) *IOGrid[Data, DataPtr] {
	s := &IOGrid[Data, DataPtr]{
		Data:       make(map[low_io.GridPos]GridSubChunkStorage[DataPtr]),
		newGrid:    makeNew,
		GridXLevel: gridXLevel,
		GridYLevel: gridYLevel,
		GridZLevel: gridZLevel,
		Storage:    storage,
	}
	maxActivateGrids := maxActivateSubChunks >> (gridXLevel + gridYLevel + gridZLevel)
	if maxActivateGrids < 1 {
		panic(fmt.Errorf("max activate grids should > %v", 1<<(gridXLevel+gridYLevel+gridZLevel)))
	}
	activateGrid, _ := simplelru.NewLRU(maxActivateGrids, func(pos low_io.GridPos, value struct{}) {
		s.swapOutGrid(pos)
	})
	s.ActivateGrid = activateGrid
	return s
}

func (s *IOGrid[Data, DataPtr]) InsightFn(pos celler.SubChunkPos) (hash uint64) {
	// X|Z|Y
	X := uint32(pos.X) >> s.GridXLevel       // 29
	Z := uint32(pos.Z) >> s.GridZLevel       // 29
	Y := uint32(pos.Y) >> s.GridYLevel & 0xF // 6
	hash = uint64(Y) | (uint64(X) << 6) | (uint64(Z) << 35)
	return hash
}

func (s *IOGrid[Data, DataPtr]) tearPos(pos celler.SubChunkPos) (low_io.GridPos, celler.SubChunkPos) {
	return low_io.GridPos{
			X: int32(pos.X) >> s.GridXLevel,
			Y: int32(pos.Y) >> s.GridYLevel,
			Z: int32(pos.Z) >> s.GridZLevel,
		}, celler.SubChunkPos{
			ChunkPos: celler.ChunkPos{
				X: pos.X % celler.ChunkUnit(1<<s.GridXLevel),
				Z: pos.Z % celler.ChunkUnit(1<<s.GridZLevel),
			},
			Y: pos.Y % celler.ChunkUnit(1<<s.GridYLevel),
		}
}

func (s *IOGrid[Data, DataPtr]) assamblePos(gp low_io.GridPos, cp celler.SubChunkPos) celler.SubChunkPos {
	baseX := celler.ChunkUnit(gp.X) << s.GridXLevel
	baseY := celler.ChunkUnit(gp.Y) << s.GridYLevel
	baseZ := celler.ChunkUnit(gp.Z) << s.GridZLevel
	return celler.SubChunkPos{
		ChunkPos: celler.ChunkPos{X: baseX + cp.X, Z: baseZ + cp.Z},
		Y:        baseY + cp.Y,
	}
}

// 从底层存储中读取目标 grid
func (s *IOGrid[Data, DataPtr]) loadGrid(gp low_io.GridPos) GridSubChunkStorage[DataPtr] {
	reader := s.Storage.DataReader(gp)
	if reader == nil {
		return nil
	}
	emptyGrid := s.newGrid()
	emptyGrid.LoadData(reader)
	return emptyGrid
}

// 如果没有加载 grid， 则从底层存储中读取目标 grid, 如果底层也没有，则返回空
func (s *IOGrid[Data, DataPtr]) ensureGridLoaded(gp low_io.GridPos) (grid GridSubChunkStorage[DataPtr]) {
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
func (s *IOGrid[Data, DataPtr]) ensureGridForWrite(gp low_io.GridPos) (grid GridSubChunkStorage[DataPtr]) {
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
func (s *IOGrid[Data, DataPtr]) swapOutGrid(gd low_io.GridPos) {
	grid := s.Data[gd]
	s.ActivateGrid.Remove(gd)
	delete(s.Data, gd)
	if grid != nil {
		grid.DumpData(s.Storage.DataWriter(gd))
	} else {
		s.Storage.DropGrid(gd)
	}
	if s.lastPos == gd {
		s.hasLast = false
	}
}

// 将所有活跃的grid保存到底层但是保持激活状态
func (s *IOGrid[Data, DataPtr]) DumpAll() {
	for _, gd := range s.ActivateGrid.Keys() {
		grid := s.Data[gd]
		if grid != nil {
			grid.DumpData(s.Storage.DataWriter(gd))
		} else {
			s.Storage.DropGrid(gd)
		}
	}
}

func (s *IOGrid[Data, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
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
	for gp := range s.Storage.IterGrid {
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

func (s *IOGrid[Data, DataPtr]) Drop(pos celler.SubChunkPos) {
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		return
	}
	grid.Drop(cp)
	if grid.IsEmpty() {
		delete(s.Data, gp)
	}
	if s.lastPos == gp {
		s.lastPtr = nil
	}
}

func (s *IOGrid[Data, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	if data == nil {
		s.Drop(pos)
		return
	}
	gp, cp := s.tearPos(pos)
	s.ensureGridForWrite(gp).Set(cp, data)
}

func (s *IOGrid[Data, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
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

func (s *IOGrid[Data, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	gp, cp := s.tearPos(pos)
	grid := s.ensureGridLoaded(gp)
	if grid == nil {
		return 0
	}
	return grid.GetVersion(cp)
}

func (s *IOGrid[Data, DataPtr]) GetOrCreate(pos celler.SubChunkPos) DataPtr {
	gp, cp := s.tearPos(pos)
	current := s.ensureGridForWrite(gp)
	return current.GetOrCreate(cp)
}

// func (s *IOGrid[Data, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
// 	gp, cp := s.tearPos(pos)
// 	grid := s.ensureGridLoaded(gp)
// 	if grid == nil {
// 		if version == 0 {
// 			return celler.SubChunkChange[Change]{
// 				Valid: true,
// 			}
// 		} else {
// 			return celler.SubChunkChange[Change]{
// 				Valid: false,
// 				Drop:  true,
// 			}
// 		}
// 	}
// 	return grid.QueryChange(cp, version)
// }

// func (s *IOGrid[Data, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
// 	if change.Valid {
// 		return
// 	}
// 	gp, cp := s.tearPos(pos)
// 	if change.Drop {
// 		grid := s.ensureGridLoaded(gp)
// 		if grid != nil {
// 			grid.ApplyChange(cp, change)
// 			if grid.IsEmpty() {
// 				delete(s.Data, gp)
// 				s.hasLast = false
// 			}
// 		}

// 		return
// 	}
// 	s.ensureGridForWrite(gp).ApplyChange(cp, change)
// }
