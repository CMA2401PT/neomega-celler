package celler

// 描述一个方块发生的变化
type BlockChangeType byte

const (
	BlockDeltaChangeBlockID = BlockChangeType(1)
	BlockDeltaChangeAttach  = BlockChangeType(1 << 1)
	BlockDeltaChangeSNBT    = BlockChangeType(1 << 2)
)

type SubChunkChange[BlockIDT ~uint32, AttachT ~uint32] struct {
	// case 1. 通常更新: Valid=false, Drop=false
	// 从 空 或现有 Cache 更新到目标
	// case 2. 移除数据: Valid=false,Drop=true
	// 删除现有 Cache
	// case 3. 不变更: Valid=true
	Valid bool
	Drop  bool
	// 当 Base 存在时，意味着要先应用 Base 替换掉当前 Cache 的base
	Base SubChunk[BlockIDT, AttachT]
	// BlockChanges 代表基于Base 发生的方块变化
	BlockChanges []VersionedBlockChange[BlockIDT, AttachT]
}

type BlockChange[BlockIDT ~uint32, AttachT ~uint32] struct {
	// 考虑到一个方块具有多个分离的属性，可能一次变更一个或者多个，使用 Mark 来标记其变更
	Type BlockChangeType
	// 变更的方块相对于子区块位置
	Pos     SubChunkBlockPos
	BlockID BlockIDT
	Attach  AttachT
	// SNBT 是一个 string，所以其内部存在一个指针
	SNBT SNBT
}

type VersionedBlockChange[BlockIDT ~uint32, AttachT ~uint32] struct {
	BlockChange[BlockIDT, AttachT]
	// 变更前子区块版本
	BeforeVersion ChunkVersion
	// 变更后子区块版本
	AfterVersion ChunkVersion
}

func (d BlockChange[BlockIDT, AttachT]) SetPos(pos SubChunkBlockPos) BlockChange[BlockIDT, AttachT] {
	d.Pos = pos
	return d
}

func (d BlockChange[BlockIDT, AttachT]) SetID(id BlockIDT) BlockChange[BlockIDT, AttachT] {
	d.BlockID = id
	d.Type |= BlockDeltaChangeBlockID
	return d
}

func (d BlockChange[BlockIDT, AttachT]) SetAttach(attach AttachT) BlockChange[BlockIDT, AttachT] {
	d.Attach = attach
	d.Type |= BlockDeltaChangeAttach
	return d
}

func (d BlockChange[BlockIDT, AttachT]) SetSNBT(snbt SNBT) BlockChange[BlockIDT, AttachT] {
	d.SNBT = snbt
	d.Type |= BlockDeltaChangeSNBT
	return d
}
