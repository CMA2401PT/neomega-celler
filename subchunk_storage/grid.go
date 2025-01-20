package subchunk_storage

import "celler"

type GridPos struct {
	X int32
	Z int32
}

type Grid[Data any, Change any, DataPtr StandardSubChunk[Data, Change]] struct {
	Data         map[GridPos]*Simple[Data, Change, DataPtr]
	GridX, GridZ uint32
	hasLast      bool
	lastPos      GridPos
	lastPtr      *Simple[Data, Change, DataPtr]
	NewEmpty     func() DataPtr
}

func NewGrid[BlockIDT, AttachT ~uint32, Data any, DataPtr StandardSubChunk[Data, celler.SubChunkDataChange[BlockIDT, AttachT]]](
	bid BlockIDT, attach AttachT,
	makeNew func(BlockIDT, AttachT) DataPtr,
	gridX uint32, gridZ uint32,
) *Grid[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr] {
	return &Grid[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
		GridX:    gridX,
		GridZ:    gridZ,
	}
}

func (s *Grid[Data, Change, DataPtr]) InsightFn() func(celler.ChunkPos) (hash int) {
	return func(pos celler.ChunkPos) (hash int) {
		return (int(pos.Z)/int(s.GridZ))*int(s.GridX) + (int(pos.X) / int(s.GridX))
	}
}

func (s *Grid[Data, Change, DataPtr]) tearPos(pos celler.SubChunkPos) (GridPos, celler.SubChunkPos) {
	return GridPos{
			X: int32(pos.X) / int32(s.GridX),
			Z: int32(pos.Z) / int32(s.GridZ),
		}, celler.SubChunkPos{
			ChunkPos: celler.ChunkPos{
				X: pos.X % celler.ChunkUnit(s.GridX),
				Z: pos.Z % celler.ChunkUnit(s.GridZ),
			},
			Y: pos.Y,
		}
}

func (s *Grid[Data, Change, DataPtr]) assamblePos(gp GridPos, cp celler.SubChunkPos) celler.SubChunkPos {
	baseX := celler.ChunkUnit(gp.X) * celler.ChunkUnit(s.GridX)
	baseZ := celler.ChunkUnit(gp.Z) * celler.ChunkUnit(s.GridZ)
	return celler.SubChunkPos{
		ChunkPos: celler.ChunkPos{X: baseX + cp.X, Z: baseZ + cp.Z},
		Y:        cp.Y,
	}
}

func (s *Grid[Data, Change, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil || s.Data == nil {
		return
	}
	for gp, gd := range s.Data {
		for sp, d := range gd.IterSubChunks {
			finalPos := s.assamblePos(gp, sp)
			if !cb(finalPos, d) {
				return
			}
		}
	}
}

func (s *Grid[Data, Change, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	gp, cp := s.tearPos(pos)
	if s.lastPos == gp && s.hasLast {
		if s.lastPtr == nil {
			return nil
		}
		return s.lastPtr.Get(cp)
	}
	grid := s.Data[gp]
	s.hasLast = true
	s.lastPos = gp
	s.lastPtr = grid
	if s.lastPtr == nil {
		return nil
	}
	return s.lastPtr.Get(cp)
}

func (s *Grid[Data, Change, DataPtr]) newGrid() *Simple[Data, Change, DataPtr] {
	return &Simple[Data, Change, DataPtr]{
		Data:     make(map[celler.SubChunkPos]DataPtr),
		NewEmpty: s.NewEmpty,
	}
}

func (s *Grid[Data, Change, DataPtr]) ensureGrid(gp GridPos) *Simple[Data, Change, DataPtr] {
	if s.hasLast && s.lastPos == gp && s.lastPtr != nil {
		return s.lastPtr
	}
	if s.Data == nil {
		s.Data = make(map[GridPos]*Simple[Data, Change, DataPtr])
	}
	grid := s.Data[gp]
	if grid == nil {
		grid = s.newGrid()
		s.Data[gp] = grid
	}
	s.hasLast = true
	s.lastPos = gp
	s.lastPtr = grid
	return grid
}

func (s *Grid[Data, Change, DataPtr]) getGrid(gp GridPos) *Simple[Data, Change, DataPtr] {
	if s.hasLast && s.lastPos == gp {
		return s.lastPtr
	}
	grid := s.Data[gp]
	s.hasLast = true
	s.lastPos = gp
	s.lastPtr = grid
	return grid
}

func (s *Grid[Data, Change, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	gp, cp := s.tearPos(pos)
	if data == nil {
		if s.Data[gp] == nil {
			return
		}

	}
	s.ensureGrid(gp).Set(cp, data)
}

func (s *Grid[Data, Change, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
	gp, cp := s.tearPos(pos)
	grid := s.getGrid(gp)
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

func (s *Grid[Data, Change, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	gp, cp := s.tearPos(pos)
	grid := s.getGrid(gp)
	if grid == nil {
		return 0
	}
	return grid.GetVersion(cp)
}

func (s *Grid[Data, Change, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	gp, cp := s.tearPos(pos)
	if change.Drop {
		if s != nil && s.Data != nil {
			grid := s.getGrid(gp)
			if grid != nil {
				grid.ApplyChange(cp, change)
				if len(grid.Data) == 0 {
					delete(s.Data, gp)
					s.lastPtr = nil
				}
			}
		}
		return
	}
	s.ensureGrid(gp).ApplyChange(cp, change)
}
