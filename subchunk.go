package celler

type ChunkVersion int32

const (
	ChunkVersionNoData = 0
)

// SubChunkData 完全描述了一个SubChunk内的方块信息
// SubChunkData 本身并不能表示两种特殊情况:
// 1. 这个区域是未知的
// 2. 这个区域是已知没有数据(全空气的)
// 但是可以表示，这个区域是已知的，有数据的，只是刚好数据全是空气
type SubChunkData[BlockIDT ~uint32, AttachT ~uint32] interface {
	Version() ChunkVersion
	Meta() map[string]string
	BlockID() LayerStorageRandomRead[BlockIDT]
	BlockAttach() LayerStorageRandomRead[AttachT]
	BlockSNBT() LayerStorageRandomRead[SNBT]
}

// SubChunkRead 描述了一个可以通过 QueryChanges 方式获得缓存的 Subchunk 与当前subchunk差距的接口
type SubChunkRead[BlockIDT ~uint32, AttachT ~uint32] interface {
	Version() ChunkVersion
	QueryChange(cacheVersion ChunkVersion) SubChunkDataChange[BlockIDT, AttachT]
}

// SubChunkWite 描述了一个可以通过 ApplyChange 来同步源数据的接口
type SubChunkWrite[BlockIDT ~uint32, AttachT ~uint32] interface {
	Version() ChunkVersion
	ApplyChange(change SubChunkDataChange[BlockIDT, AttachT])
	ChangeBlock(delta SubChunkBlockDelta[BlockIDT, AttachT])
	ChangeBlockWithVersion(delta SubChunkBlockDelta[BlockIDT, AttachT], newVersion ChunkVersion)
}

// 描述一个方块发生的变化
type BlockDeltaType byte

const (
	BlockDeltaChangeBlockID = BlockDeltaType(1)
	BlockDeltaChangeAttach  = BlockDeltaType(1 << 1)
	BlockDeltaChangeSNBT    = BlockDeltaType(1 << 2)
)

// SubChunkBlockDeltaNoSNBT 具有固定的尺寸且内部不存在指针结构
type SubChunkBlockDeltaNoSNBT[BlockIDT ~uint32, AttachT ~uint32] struct {
	// 考虑到一个方块具有多个分离的属性，可能一次变更一个或者多个，使用 Mark 来标记其变更
	Type BlockDeltaType
	// 变更的方块相对于子区块位置
	Pos     SubChunkBlockPos
	BlockID BlockIDT
	Attach  AttachT
}

type SubChunkBlockDelta[BlockIDT ~uint32, AttachT ~uint32] struct {
	SubChunkBlockDeltaNoSNBT[BlockIDT, AttachT]
	// SNBT 是一个 string，所以其内部存在一个指针
	SNBT SNBT
}

type SubChunkBlockDeltaWithVersion[BlockIDT ~uint32, AttachT ~uint32] struct {
	SubChunkBlockDelta[BlockIDT, AttachT]
	// 变更前子区块版本
	BeforeVersion ChunkVersion
	// 变更后子区块版本
	AfterVersion ChunkVersion
}

type SubChunkDataChange[BlockIDT ~uint32, AttachT ~uint32] struct {
	// 当 Base 存在时，意味着要先应用 Base 替换掉当前 Cache 的base
	Base SubChunkData[BlockIDT, AttachT]
	// BlockDeltas 代表基于Base 发生的方块变化
	BlockDeltas []SubChunkBlockDeltaWithVersion[BlockIDT, AttachT]
}

func (d SubChunkBlockDelta[BlockIDT, AttachT]) SetPos(pos SubChunkBlockPos) SubChunkBlockDelta[BlockIDT, AttachT] {
	d.Pos = pos
	return d
}

func (d SubChunkBlockDelta[BlockIDT, AttachT]) SetID(id BlockIDT) SubChunkBlockDelta[BlockIDT, AttachT] {
	d.BlockID = id
	d.Type |= BlockDeltaChangeBlockID
	return d
}

func (d SubChunkBlockDelta[BlockIDT, AttachT]) SetAttach(attach AttachT) SubChunkBlockDelta[BlockIDT, AttachT] {
	d.Attach = attach
	d.Type |= BlockDeltaChangeAttach
	return d
}

func (d SubChunkBlockDelta[BlockIDT, AttachT]) SetSNBT(snbt SNBT) SubChunkBlockDelta[BlockIDT, AttachT] {
	d.SNBT = snbt
	d.Type |= BlockDeltaChangeSNBT
	return d
}
