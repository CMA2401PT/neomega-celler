package celler

type SubChunkStorageRead[Data any] interface {
	IterSubChunks(func(SubChunkPos, Data) (next bool))
}

type SubChunkStorageRandomRead[Data any] interface {
	Get(SubChunkPos) Data
	GetVersion(SubChunkPos) ChunkVersion
}

type SubChunkStorageRandomWrite[Data any] interface {
	Set(SubChunkPos, Data)
	Drop(SubChunkPos)
}

type SubChunkStorage[Data any] interface {
	SubChunkStorageRead[Data]
	SubChunkStorageRandomRead[Data]
	SubChunkStorageRandomWrite[Data]
	GetOrCreate(SubChunkPos) Data
}

type SubChunkBatchOperator[Data any] interface {
	BatchGet([]SubChunkPos) []struct {
		Pos  SubChunkPos
		Data Data
	}
	BatchSet(map[SubChunkPos]Data)
}

type SubChunkChange[Change any] struct {
	// case 1. 通常更新: Valid=false, Drop=false
	// 从 空 或现有 Cache 更新到目标
	// case 2. 移除数据: Valid=false,Drop=true
	// 删除现有 Cache
	// case 3. 不变更: Valid=true
	Valid  bool
	Drop   bool
	Change *Change
}

type SubChunkStorageSyncProvider[Change any] interface {
	// 用于同步数据
	QueryChange(SubChunkPos, ChunkVersion) SubChunkChange[Change]
}

type SubChunkStorageSyncReciver[Change any] interface {
	// 用于同步数据
	ApplyChange(SubChunkPos, SubChunkChange[Change])
}

type SubChunkStorageSyncer[Change any] interface {
	SubChunkStorageSyncProvider[Change]
	SubChunkStorageSyncReciver[Change]
}
