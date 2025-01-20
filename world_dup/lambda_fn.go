package main

import (
	"celler"
	"time"
)

type WorldFn[B ~uint32] func(chunkX, chunkY, chunkZ celler.ChunkUnit) (U32SubchunkData[B], ChunkVersion)

func DefaultWorldFn[B ~uint32](cx, cy, cz ChunkUnit) (U32SubchunkData[B], ChunkVersion) {
	if cy < 0 {
		return makeStorageByLevel[B](0, 1), 1
	} else if cy == 0 {
		bx := int(cx * 16)
		bz := int(cz * 16)
		abs := func(v int) int {
			if v < 0 {
				return -v
			}
			return v
		}
		blocksData := makeStorageByLevel[B](0, 0)
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				RTID := abs((bx+x)%256) + 256*abs((bz+z)%256)
				blocksData = blocksData.setBlock(x, 0, z, 1)
				blocksData = blocksData.setBlock(x, 1, z, B(RTID%10000+1))
			}
		}
		return blocksData, 1
	} else {
		return nil, 1
	}
}

func (w WorldFn[B]) ToStaticLambdaServer(ratio time.Duration) *BasicStaticLambdaServer[B] {
	return NewLambdaServer(w, ratio)
}
