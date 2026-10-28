package enhance

import (
	"celler"
	"sort"
)

type BaseStorage[Data any] interface {
	InsightFn(celler.SubChunkPos) (hash uint64)
	celler.SubChunkStorageRandomRead[Data]
	celler.SubChunkStorageRandomWrite[Data]
}

type BatchOperator[Data any, Base BaseStorage[Data]] struct {
	Base      Base
	InsightFn func(celler.SubChunkPos) (hash uint64)
}

func NewBatchOperator[Data any, Base BaseStorage[Data]](base Base) BatchOperator[Data, Base] {
	return BatchOperator[Data, Base]{
		Base:      base,
		InsightFn: base.InsightFn,
	}
}

func (b BatchOperator[Data, Base]) BatchGet(poses []celler.SubChunkPos) (out []struct {
	Pos  celler.SubChunkPos
	Data Data
}) {
	if b.InsightFn != nil {
		sort.Slice(poses, func(i, j int) bool {
			return b.InsightFn(poses[i]) < b.InsightFn(poses[j])
		})
	}
	out = make([]struct {
		Pos  celler.SubChunkPos
		Data Data
	}, 0, len(poses))
	for _, pos := range poses {
		out = append(out, struct {
			Pos  celler.SubChunkPos
			Data Data
		}{
			Pos:  pos,
			Data: b.Base.Get(pos),
		})
	}
	return out
}

func (b BatchOperator[Data, Base]) BatchSet(datas map[celler.SubChunkPos]Data) {
	if b.InsightFn == nil {
		for k, v := range datas {
			b.Base.Set(k, v)
		}
	} else {
		order := make([]struct {
			Pos  celler.SubChunkPos
			Data Data
		}, 0, len(datas))
		for k, v := range datas {
			order = append(order, struct {
				Pos  celler.SubChunkPos
				Data Data
			}{
				Pos:  k,
				Data: v,
			})
		}
		sort.Slice(order, func(i, j int) bool {
			return b.InsightFn(order[i].Pos) < b.InsightFn(order[j].Pos)
		})
		for _, r := range order {
			b.Base.Set(r.Pos, r.Data)
		}
	}
}
