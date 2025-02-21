package bedrock

import (
	"celler"
	"encoding/binary"
)

// Keys on a per-sub chunk basis. These are prefixed by the chunk coordinates and subchunk ID.
const (
	keySubChunkData = '/' // 2f
)

// Keys on a per-chunk basis. These are prefixed by only the chunk coordinates.
const (
	// keyVersion holds a single byte of data with the version of the chunk.
	keyVersion = ',' // 2c
	// keyBlockEntities holds n amount of NBT compound tags appended to each other (not a TAG_List, just appended). The
	// compound tags contain the position of the block entities.
	keyBlockEntities = '1' // 31
	// keyEntities holds n amount of NBT compound tags appended to each other (not a TAG_List, just appended). The
	// compound tags contain the position of the entities.
	keyEntities = '2' // 32
	// keyFinalisation contains a single LE int32 that indicates the state of generation of the chunk. If 0, the chunk
	// needs to be ticked. If 1, the chunk needs to be populated and if 2 (which is the state generally found in world
	// saves from vanilla), the chunk is fully finalised.
	keyFinalisation = '6' // 36
	// key3DData holds 3-dimensional biomes for the entire chunk.
	key3DData = '+' // 2b
	// key2DData is no longer used in worlds with world height change. It was replaced by key3DData in newer worlds
	// which has 3-dimensional biomes.
	key2DData = '-' // 2d
	// keyChecksum holds a list of checksums of some sort. It's not clear of what data this checksum is composed or what
	// these checksums are used for.
	keyChecksums = ';' // 3b
)

// index returns a byte buffer holding the written index of the chunk position passed. If the dimension passed to New
// is not world.Overworld, the length of the index returned is 12. It is 8 otherwise.
func (p *BedrockStorageReadOnly[T]) index(position celler.ChunkPos) []byte {
	x, z, dim := uint32(position.X), uint32(position.Z), uint32(p.Dim)
	b := make([]byte, 12)

	binary.LittleEndian.PutUint32(b, x)
	binary.LittleEndian.PutUint32(b[4:], z)
	if dim == 0 {
		return b[:8]
	}
	binary.LittleEndian.PutUint32(b[8:], dim)
	return b
}
