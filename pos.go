// Package celler provides coordinates shared by native world storage packages.
package celler

// ChunkPos addresses a vertical column of 16×16 blocks.
type ChunkPos struct{ X, Z int32 }

// BlockPos is an absolute block position within a dimension.
type BlockPos struct{ X, Y, Z int32 }

func (p BlockPos) Chunk() ChunkPos { return ChunkPos{X: p.X >> 4, Z: p.Z >> 4} }
