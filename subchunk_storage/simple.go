package subchunk_storage

import (
	"celler"
	"celler/subchunk_storage/low_io"
	"sort"
)

type Simple[Data any, Change any, DataPtr StandardSubChunk[Data, Change]] struct {
	Data     map[celler.SubChunkPos]DataPtr
	hasLast  bool
	lastPos  celler.SubChunkPos
	lastPtr  DataPtr
	NewEmpty func() DataPtr
}

func NewSimple[BlockIDT, AttachT ~uint32, Data any, DataPtr StandardSubChunk[Data, celler.SubChunkDataChange[BlockIDT, AttachT]]](
	bid BlockIDT, attach AttachT,
	makeNew func(BlockIDT, AttachT) DataPtr,
) *Simple[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr] {
	return &Simple[Data, celler.SubChunkDataChange[BlockIDT, AttachT], DataPtr]{
		NewEmpty: func() DataPtr { return makeNew(bid, attach) },
	}
}

func (s *Simple[Data, Change, DataPtr]) IsEmpty() bool {
	return len(s.Data) == 0
}

func (s *Simple[Data, Change, DataPtr]) LoadData(reader low_io.SubChunkDataReadCloser) {
	defer reader.Close()
	chunkVersions := reader.CurrentVersion()
	if len(chunkVersions) == 0 {
		s.Data = make(map[celler.SubChunkPos]DataPtr)
		return
	}
	nd := make(map[celler.SubChunkPos]DataPtr)
	for p, v := range chunkVersions {
		if current, found := s.Data[p]; found {
			if current.Version() == v {
				nd[p] = current
				continue
			}
		}
		d := s.NewEmpty()
		rd := reader.GetReader(p)
		d.Load(rd)
		rd.Close()
		nd[p] = d
	}
	s.Data = nd
}

func (s *Simple[Data, Change, DataPtr]) DumpData(writer low_io.SubChunkDataWriteCloser) {
	defer writer.Close()
	existVersions := writer.CurrentVersion()
	poses := make([]celler.SubChunkPos, 0, len(s.Data))
	for p := range s.Data {
		poses = append(poses, p)
	}
	sort.Slice(poses, func(i, j int) bool {
		return poses[i].Less(poses[j])
	})
	for _, p := range poses {
		d := s.Data[p]
		if d.Version() == existVersions[p] {
			continue
		}
		wt := writer.GetWriter(p, d.Version())
		d.Dump(wt)
		wt.Close()
	}

	for pos := range existVersions {
		if _, found := s.Data[pos]; !found {
			writer.Drop(pos)
		}
	}

}

func (s *Simple[Data, Change, DataPtr]) InsightFn() func(celler.ChunkPos) (hash int) {
	return nil
}

func (s *Simple[Data, Change, DataPtr]) IterSubChunks(cb func(celler.SubChunkPos, DataPtr) (next bool)) {
	if s == nil || s.Data == nil {
		return
	}
	for p, d := range s.Data {
		if !cb(p, d) {
			return
		}
	}
}

func (s *Simple[Data, Change, DataPtr]) Get(pos celler.SubChunkPos) DataPtr {
	if s == nil {
		return nil
	}
	if s.lastPos == pos && s.hasLast {
		return s.lastPtr
	}
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = s.Data[pos]
	return s.lastPtr
}

func (s *Simple[Data, Change, DataPtr]) Set(pos celler.SubChunkPos, data DataPtr) {
	s.hasLast = true
	s.lastPos = pos
	s.lastPtr = data
	if data == nil {
		if s.Data == nil {
			return
		} else {
			delete(s.Data, pos)
		}
	}
	if s.Data == nil {
		s.Data = make(map[celler.SubChunkPos]DataPtr)
	}
	s.Data[pos] = data
}

func (s *Simple[Data, Change, DataPtr]) QueryChange(pos celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[Change] {
	current := s.Get(pos)
	if current == nil {
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
	if current.Version() == version {
		return celler.SubChunkChange[Change]{
			Valid: true,
		}
	} else {
		change := current.QueryChange(version)
		return celler.SubChunkChange[Change]{
			Valid:  false,
			Drop:   false,
			Change: &change,
		}
	}
}

func (s *Simple[Data, Change, DataPtr]) GetVersion(pos celler.SubChunkPos) celler.ChunkVersion {
	current := s.Get(pos)
	if current == nil {
		return 0
	}
	return current.Version()
}

func (s *Simple[Data, Change, DataPtr]) ApplyChange(pos celler.SubChunkPos, change celler.SubChunkChange[Change]) {
	if change.Valid {
		return
	}
	if change.Drop {
		if s != nil && s.Data != nil {
			delete(s.Data, pos)
		}
		s.lastPtr = nil
		return
	}
	current := s.Get(pos)
	if current == nil {
		current = s.NewEmpty()
		if s.Data == nil {
			s.Data = make(map[celler.SubChunkPos]DataPtr)
		}
		s.Data[pos] = current
		s.lastPtr = current
	}
	current.ApplyChange(*change.Change)
}
