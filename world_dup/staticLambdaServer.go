package main

import (
	"time"
)

type SubchunkResponse[B ~uint32] struct {
	SubChunkPos
	Version    ChunkVersion
	BlocksData U32SubchunkData[B]
}

func (r SubchunkResponse[B]) Dump(d []byte) []byte {
	d = ValueDumpFnUint32(r.X, d)
	d = ValueDumpFnUint32(r.Y, d)
	d = ValueDumpFnUint32(r.Z, d)
	d = ValueDumpFnUint32(r.Version, d)
	d = r.BlocksData.Dump(d)
	return d
}

func LoadSubchunkResponse[B ~uint32](data []byte, i int) (SubchunkResponse[B], int) {
	r := SubchunkResponse[B]{}
	r.X, i = ValueLoadFnUint32[ChunkUnit](data, i)
	r.Y, i = ValueLoadFnUint32[ChunkUnit](data, i)
	r.Z, i = ValueLoadFnUint32[ChunkUnit](data, i)
	r.Version, i = ValueLoadFnUint32[ChunkVersion](data, i)
	r.BlocksData, i = LoadU32SubchunkData[B](data, i)
	return r, i
}

type BasicStaticLambdaServer[B ~uint32] struct {
	sended             ChunkedStorage[struct{}]
	lambdaFn           WorldFn[B]
	centerChangeWaitor chan CenteredArea
	Changes            chan []SubchunkResponse[B]
}

func NewLambdaServer[B ~uint32](
	lambdaFn WorldFn[B],
	ratio time.Duration,
) *BasicStaticLambdaServer[B] {
	server := &BasicStaticLambdaServer[B]{
		sended:             ChunkedStorage[struct{}]{},
		lambdaFn:           lambdaFn,
		centerChangeWaitor: make(chan CenteredArea, 16),
		Changes:            make(chan []SubchunkResponse[B], 16*16),
	}
	go server.routine(ratio)
	return server
}

func (s *BasicStaticLambdaServer[B]) UpdateCenter(center CenteredArea) {
	s.centerChangeWaitor <- center
}

func (s *BasicStaticLambdaServer[B]) routine(ratio time.Duration) {
	raitoLimiter := time.NewTimer(ratio)
	center := <-s.centerChangeWaitor
	for {
		<-raitoLimiter.C
		raitoLimiter = time.NewTimer(ratio)
		for len(s.centerChangeWaitor) > 0 {
			s.sended.Filter(center)
			center = <-s.centerChangeWaitor
		}
		s.sended.Filter(center)
		earlyQuit := false
		toSend := []ChunkPos{}
		for pos := range center.Iter {
			if s.sended.Has(pos) {
				continue
			}
			toSend = append(toSend, pos)
		}
		for _, pos := range toSend {
			select {
			case center = <-s.centerChangeWaitor:
				earlyQuit = true
			default:
			}
			if earlyQuit {
				break
			}
			changes := make([]SubchunkResponse[B], 0, 24)
			for y := -8; y < 24; y++ {
				cd, v := s.lambdaFn(pos.X, ChunkUnit(y), pos.Z)
				changes = append(changes, SubchunkResponse[B]{
					SubChunkPos{ChunkPos: pos, Y: ChunkUnit(y)},
					ChunkVersion(v),
					cd,
				})
			}
			s.Changes <- changes
			s.sended.Set(pos, struct{}{})
		}
		if !earlyQuit {
			center = <-s.centerChangeWaitor
		}
	}
}
