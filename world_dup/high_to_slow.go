package world

import "celler"

type SyncToAsyncWrapper[BlockIDT ~uint32, AttachT ~uint32] struct {
	syncWorld celler.StaticWorldServerSync[BlockIDT, AttachT]
}

func (w *SyncToAsyncWrapper[BlockIDT, AttachT]) QueryAsync(queries <-chan *celler.SubChunkQuery) <-chan *celler.SubChunkResponse[BlockIDT, AttachT] {
	batchSize := 32
	outChan := make(chan *celler.SubChunkResponse[BlockIDT, AttachT], batchSize)
	go func() {
		defer close(outChan)
		batchQuery := []celler.SubChunkQuery{}
		flush := func() {
			for _, _r := range w.syncWorld.QueryInstance(batchQuery) {
				r := _r
				outChan <- &r
			}
		}
		for _query := range queries {
			if _query == nil {
				flush()
				return
			} else {
				query := _query
				batchQuery = append(batchQuery, *query)
				if len(batchQuery) >= batchSize {
					flush()
					batchQuery = batchQuery[:0]
				}
			}
		}
	}()
	return outChan
}

func MakeSyncToAsyncWrapper[BlockIDT ~uint32, AttachT ~uint32](highSpeed celler.StaticWorldServerSync[BlockIDT, AttachT]) *SyncToAsyncWrapper[BlockIDT, AttachT] {
	return &SyncToAsyncWrapper[BlockIDT, AttachT]{
		syncWorld: highSpeed,
	}
}
