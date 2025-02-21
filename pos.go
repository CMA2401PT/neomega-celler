package celler

import "strconv"

type ChunkUnit int32

func (cu ChunkUnit) InRange(v int) bool {
	return (v >= (int(cu) << 4)) && (v < ((int(cu) + 1) << 4))
}

type ChunkPos struct {
	X ChunkUnit
	Z ChunkUnit
}

func (cp ChunkPos) String() string {
	return "ChunkPos:" + strconv.Itoa(int(cp.X)) + "," + strconv.Itoa(int(cp.Z))
}

func (p ChunkPos) Less(p2 ChunkPos) bool {
	if p.X == p2.X {
		return p.Z < p2.Z
	}
	return p.X < p2.X
}

type SubChunkPos struct {
	ChunkPos
	Y ChunkUnit
}

func (scp SubChunkPos) String() string {
	return "SubChunkPos:" + strconv.Itoa(int(scp.X)) + "," + strconv.Itoa(int(scp.Y)) + "," + strconv.Itoa(int(scp.Z))
}

func (p SubChunkPos) Less(p2 SubChunkPos) bool {
	if p.X == p2.X && p.Z == p2.Z {
		return p.Y < p2.Y
	}
	return p.ChunkPos.Less(p2.ChunkPos)
}

func (scp SubChunkPos) ToBytes() []byte {
	return []byte{
		byte(scp.X), byte(scp.X >> 8), byte(scp.X >> 16), byte(scp.X >> 24),
		byte(scp.Y), byte(scp.Y >> 8), byte(scp.Y >> 16), byte(scp.Y >> 24),
		byte(scp.Z), byte(scp.Z >> 8), byte(scp.Z >> 16), byte(scp.Z >> 24),
	}
}

func ToSubChunkPos[X, Y, Z Interger](x X, y Y, z Z) SubChunkPos {
	return SubChunkPos{ChunkPos: ChunkPos{X: ChunkUnit(x), Z: ChunkUnit(z)}, Y: ChunkUnit(y)}
}

type SubChunkBlockPos struct {
	X, Y, Z uint8 // 以 SubChunk 为基准的偏移
}

type ChunkBlockPos struct {
	X, Z uint8
	Y    int16
}

type Interger interface {
	~uint8 | ~int8 | ~uint16 | ~int16 | ~uint32 | ~int32 | ~uint64 | ~int64 | ~uint | ~int
}

func ToSubChunkBlockPos[X, Y, Z Interger](x X, y Y, z Z) SubChunkBlockPos {
	return SubChunkBlockPos{uint8(x & 0xf), uint8(y & 0xf), uint8(z & 0xf)}
}

type ChunkArea struct {
	Start ChunkPos
	Size  ChunkPos
}

type SubChunkArea struct {
	Start SubChunkPos
	Size  SubChunkPos
}

// type CenteredArea struct {
// 	ChunkPos
// 	ViewRange ChunkUnit
// }

// func (ca CenteredArea) Xmin() ChunkUnit {
// 	return ca.X - ca.ViewRange
// }

// func (ca CenteredArea) XmaxNotInclude() ChunkUnit {
// 	return ca.X + ca.ViewRange
// }
// func (ca CenteredArea) XmaxInclude() ChunkUnit {
// 	return ca.X + ca.ViewRange - 1
// }

// func (ca CenteredArea) Zmin() ChunkUnit {
// 	return ca.Z - ca.ViewRange
// }

// func (ca CenteredArea) ZmaxNotInclude() ChunkUnit {
// 	return ca.Z + ca.ViewRange
// }
// func (ca CenteredArea) ZmaxInclude() ChunkUnit {
// 	return ca.Z + ca.ViewRange - 1
// }

// func (ca CenteredArea) Has(X, Z ChunkUnit) bool {
// 	return X >= ca.Xmin() && X < ca.XmaxNotInclude() && Z >= ca.Zmin() && Z <= ca.ZmaxNotInclude()
// }

// func (ca CenteredArea) Iter(fn func(ChunkPos) (next bool)) {
// 	xStart := ca.Xmin()
// 	xEnd := ca.XmaxNotInclude()
// 	for x := xStart; x < xEnd; x++ {
// 		zStart := ca.Zmin()
// 		zEnd := ca.ZmaxNotInclude()
// 		for z := zStart; z < zEnd; z++ {
// 			if !fn(ChunkPos{X: x, Z: z}) {
// 				return
// 			}
// 		}
// 	}
// }
