package celler

type ChunkVersion int32

const (
	ChunkVersionNoData = 0
)

// SubChunk 完全描述了一个SubChunk内的方块信息
// SubChunk 本身并不能表示两种特殊情况:
// 1. 这个区域是未知的
// 2. 这个区域是已知没有数据(全空气的)
// 但是可以表示，这个区域是已知的，有数据的，只是刚好数据全是空气
type SubChunk[BlockIDT ~uint32, AttachT ~uint32] interface {
	// Version() 用来指示方块是否更新和新旧信息
	Version() ChunkVersion
	// Meta() 用来指示附加信息
	Meta() map[string]string
	// 方块 ID - 基础数据块
	BlockID() LayerStorageRandomRead[BlockIDT]
	// 方块附加信息 - 基础数据块
	BlockAttach() LayerStorageRandomRead[AttachT]
	// 方块 NBT 信息(SNBT形式) - 基础数据块
	BlockSNBT() LayerStorageRandomRead[SNBT]
	// 获得区块从 ChunkVersion 开始发生的变化
	QueryChange(ChunkVersion) SubChunkChange[BlockIDT, AttachT]
	// 在区块中应用变化 (Drop 必须为 false，这应该由上一级处理)
	ApplyChange(change SubChunkChange[BlockIDT, AttachT])
	// 改变区块中的方块
	ChangeBlock(delta BlockChange[BlockIDT, AttachT])
	// 改变区块中的方块并更新版本信息
	ChangeBlockWithVersion(delta BlockChange[BlockIDT, AttachT], newVersion ChunkVersion)
}
