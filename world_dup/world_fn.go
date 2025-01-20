package world

import (
	"celler"
	"celler/storage"
	"celler/subchunk"
)

type WorldFunction[BlockIDT ~uint32, AttachT ~uint32] func(pos celler.SubChunkPos) celler.SubChunkData[BlockIDT, AttachT]
type VersionFunction func(pos celler.SubChunkPos) celler.ChunkVersion

func MakeDefaultWorldFn[BlockIDT ~uint32, AttachT ~uint32](
	airID BlockIDT, stoneID BlockIDT,
	defaultAttach AttachT,
) (WorldFunction[BlockIDT, AttachT], VersionFunction) {
	worldFn :=
		func(pos celler.SubChunkPos) celler.SubChunkData[BlockIDT, AttachT] {
			if pos.Y < 0 {
				return subchunk.MakeSubChunk(1, nil,
					storage.MakePlaceHolder[BlockIDT](stoneID),
					storage.MakePlaceHolder[AttachT](0),
					storage.MakePlaceHolder[celler.SNBT](""),
				)
			}
			if pos.Y == 0 {
				bx := int(pos.X * 16)
				bz := int(pos.Z * 16)
				abs := func(v int) int {
					if v < 0 {
						return -v
					}
					return v
				}
				blockID := &storage.PaletteStorageAny[BlockIDT]{}
				blockID.InitWithDefault(0)
				for x := 0; x < 16; x++ {
					for z := 0; z < 16; z++ {
						RTID := abs((bx+x)%256) + 256*abs((bz+z)%256)
						blockID.Set(x, 0, z, 1)
						blockID.Set(x, 1, z, BlockIDT(RTID%10000+1))
					}
				}
				return subchunk.MakeSubChunk(1, nil,
					blockID,
					storage.MakePlaceHolder[AttachT](0),
					storage.MakePlaceHolder[celler.SNBT](""),
				)
			}
			{
				return nil
			}
		}
	versionFn := func(pos celler.SubChunkPos) celler.ChunkVersion {
		if pos.Y > 0 {
			return 0
		}
		return 1
	}
	return worldFn, versionFn
}
