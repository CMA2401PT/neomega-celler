package world

import "celler"

type SlowStaticWrapper[BlockIDT ~uint32, AttachT ~uint32] struct {
	worldFn   WorldFunction[BlockIDT, AttachT]
	versionFn VersionFunction
}

func (s *SlowStaticWrapper[BlockIDT, AttachT]) SetWatchRange(celler.CenteredArea, celler.WorldProcessor[BlockIDT, AttachT]) {
}

func (s *SlowStaticWrapper[BlockIDT, AttachT]) QueryAsyncQueryAsync(queries <-chan *celler.SubChunkQuery) <-chan *celler.SubChunkResponse[BlockIDT, AttachT] {
	outChan := make(chan *celler.SubChunkResponse[BlockIDT, AttachT], 32*4)
	go func() {
		defer close(outChan)
		for query := range queries {
			currentV := s.versionFn(query.Pos)
			if currentV == query.Version {
				outChan <- &celler.SubChunkResponse[BlockIDT, AttachT]{
					Pos:     query.Pos,
					Version: query.Version,
				}
			} else {
				data := s.worldFn(query.Pos)
				if data == nil {
					outChan <- &celler.SubChunkResponse[BlockIDT, AttachT]{
						Pos:            query.Pos,
						Version:        currentV,
						SubChunkChange: nil,
					}
				} else {
					outChan <- &celler.SubChunkResponse[BlockIDT, AttachT]{
						Pos:     query.Pos,
						Version: data.Version(),
						SubChunkChange: &celler.SubChunkDataChange[BlockIDT, AttachT]{
							Base: data,
						},
					}
				}
			}
		}

	}()
	return outChan
}

func WrapWorldFnAsFastStaticServer[BlockIDT ~uint32, AttachT ~uint32](
	worldFn WorldFunction[BlockIDT, AttachT],
	versionFn VersionFunction,
) *SlowStaticWrapper[BlockIDT, AttachT] {
	return &SlowStaticWrapper[BlockIDT, AttachT]{
		worldFn:   worldFn,
		versionFn: versionFn,
	}
}

type FastStaticWrapper[BlockIDT ~uint32, AttachT ~uint32] struct {
	SlowStaticWrapper[BlockIDT, AttachT]
}

func (s *FastStaticWrapper[BlockIDT, AttachT]) QueryInstance(queries []celler.SubChunkQuery) []celler.SubChunkResponse[BlockIDT, AttachT] {
	out := make([]celler.SubChunkResponse[BlockIDT, AttachT], 0, 32*4)
	for _, query := range queries {
		currentV := s.versionFn(query.Pos)
		if currentV == query.Version {
			out = append(out, celler.SubChunkResponse[BlockIDT, AttachT]{
				Pos:     query.Pos,
				Version: query.Version,
			})
		} else {
			data := s.worldFn(query.Pos)
			if data == nil {
				out = append(out, celler.SubChunkResponse[BlockIDT, AttachT]{
					Pos:            query.Pos,
					Version:        currentV,
					SubChunkChange: nil,
				})
			} else {
				out = append(out, celler.SubChunkResponse[BlockIDT, AttachT]{
					Pos:     query.Pos,
					Version: data.Version(),
					SubChunkChange: &celler.SubChunkDataChange[BlockIDT, AttachT]{
						Base: data,
					},
				})
			}
		}
	}
	return out
}

func WrapWorldFnAsSlowStaticServer[BlockIDT ~uint32, AttachT ~uint32](
	worldFn WorldFunction[BlockIDT, AttachT],
	versionFn VersionFunction,
) *FastStaticWrapper[BlockIDT, AttachT] {
	sw := &FastStaticWrapper[BlockIDT, AttachT]{
		SlowStaticWrapper: SlowStaticWrapper[BlockIDT, AttachT]{
			worldFn:   worldFn,
			versionFn: versionFn,
		},
	}
	return sw
}
