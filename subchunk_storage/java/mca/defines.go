package mca

import (
	"celler"
	"fmt"
	"strconv"
	"strings"
)

type RegionPos struct {
	Rx, Rz int32
}

type InRegionPos struct {
	Cx, Cz uint8
}

func ChunkPosToRegionPos(cp celler.ChunkPos) (RegionPos, InRegionPos) {
	return RegionPos{
			Rx: int32(cp.X) >> 5, Rz: int32(cp.Z) >> 5,
		}, InRegionPos{
			Cx: uint8(cp.X & 31), Cz: uint8(cp.Z & 31),
		}
}

func RegionPosToChunkPos(rp RegionPos, irp InRegionPos) (cp celler.ChunkPos) {
	return celler.ChunkPos{
		X: celler.ChunkUnit(rp.Rx)<<5 | celler.ChunkUnit(irp.Cx&31),
		Z: celler.ChunkUnit(rp.Rz)<<5 | celler.ChunkUnit(irp.Cz&31),
	}
}

func FileNameToRegionPos(name string) (rp RegionPos, err error) {
	ss := strings.Split(name, ".")
	if len(ss) != 4 || ss[0] != "r" || ss[3] != "mca" {
		return rp, fmt.Errorf("%v invalid", name)
	}
	x, err := strconv.Atoi(ss[1])
	if err != nil {
		return rp, fmt.Errorf("%v invalid: %v not int", name, ss[1])
	}
	z, err := strconv.Atoi(ss[2])
	if err != nil {
		return rp, fmt.Errorf("%v invalid: %v not int", name, ss[2])
	}
	return RegionPos{Rx: int32(x), Rz: int32(z)}, nil
}

func RegionPosToFileName(rp RegionPos) (name string) {
	return fmt.Sprintf("r.%v.%v.mca", rp.Rx, rp.Rz)
}

type ReadOnlyMCAChunkKV interface {
	Get(celler.ChunkPos) (value []byte, err error)
	IterAll(interFn func(celler.ChunkPos) (continueIter bool))
	Close() error
}

type ReadWriteMCAChunkKV interface {
	ReadOnlyMCAChunkKV
	Put(celler.ChunkPos, []byte) error
}
