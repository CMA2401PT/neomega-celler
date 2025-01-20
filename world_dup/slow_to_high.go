package world

import (
	"celler"
	"celler/storage"
	"celler/subchunk"
	"fmt"
)

type SyncSubChunkT[BlockIDT ~uint32, AttachT ~uint32] subchunk.Versioned[BlockIDT, AttachT,
	*storage.PaletteStorageAny[BlockIDT],
	*storage.PaletteStorageAny[AttachT],
	*storage.MapStorageAny[celler.SNBT],
]

type event struct {
	center  celler.CenteredArea
	queries []celler.SubChunkQuery
	cb      chan []celler.SubChunkResponse[BlockIDT, AttachT]
}

// 显然，将低速转为高速读取不可能没有代价:
// 读取必需集中在一个小区块且事先指明此小区块
// Block 代表当此区块数据尚未就绪时访问此区块会造成等待
type AsyncToSyncWrapperBlock[BlockIDT ~uint32, AttachT ~uint32] struct {
	asyncWorld celler.StaticWorldServerAsync[BlockIDT, AttachT]
	// 不可访问 currentRange 外的区域
	currentRange celler.CenteredArea
	queryChan    chan event
}

func (s *AsyncToSyncWrapperBlock[BlockIDT, AttachT]) routine() {
	syncedEmpty := SubChunkStorage[celler.ChunkVersion]{}
	syncedSubChunk := SubChunkStorage[SyncSubChunkT[BlockIDT, AttachT]]{}
	var syncRange celler.CenteredArea
	allResponded := true
	var response <-chan *celler.SubChunkResponse[BlockIDT, AttachT]
	onResp := func(resp *celler.SubChunkResponse[BlockIDT, AttachT]) {

	}
	onEvent := func(event event) {
		if len(event.queries) == 0 {
			syncRange = event.center
			syncedEmpty.Filter(syncRange)
			syncedSubChunk.Filter(syncRange)
		} else {

		}
	}

	for {
		if allResponded {
			onEvent(<-s.queryChan)
			continue
		}
		select {
		case event := <-s.queryChan:
			onEvent(event)
		case resp := <-response:
			onResp(resp)
		}
	}
}

func (s *AsyncToSyncWrapperBlock[BlockIDT, AttachT]) SetWatchRange(newCenter celler.CenteredArea) {
	if newCenter == s.currentRange {
		// 没有发生更改
		return
	} else {
		// 发生更改，送入更新队列处理
		s.queryChan <- event{
			center: newCenter,
		}
		s.currentRange = newCenter
	}
}

func (s *AsyncToSyncWrapperBlock[BlockIDT, AttachT]) SetChangeReciver(changeReciver celler.WorldProcessor[BlockIDT, AttachT]) {
	panic("use AsyncToSyncWrapperNonBlock instead")
}
func (s *AsyncToSyncWrapperBlock[BlockIDT, AttachT]) QueryInstance(queries []celler.SubChunkQuery) []celler.SubChunkResponse[BlockIDT, AttachT] {
	if len(queries) == 0 {
		return []celler.SubChunkResponse[BlockIDT, AttachT]{}
	}
	for _, q := range queries {
		if !s.currentRange.Has(q.Pos.X, q.Pos.Z) {
			panic(fmt.Errorf("query %v out of range %v", q.Pos, s.currentRange))
		}
	}
	ret := make(chan []celler.SubChunkResponse[BlockIDT, AttachT])
	s.queryChan <- event{
		queries: queries,
		cb:      ret,
	}
	return <-ret
}
