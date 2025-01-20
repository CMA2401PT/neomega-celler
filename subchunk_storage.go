package celler

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

type SubChunkStorageRead[Data any] interface {
	IterSubChunks(func(SubChunkPos, Data) (next bool))
}

type SubChunkStorageRandomRead[Data any] interface {
	InsightFn() func(ChunkPos) (hash int)
	Get(SubChunkPos) Data
}

type SubChunkStorageRandomWrite[Data any] interface {
	InsightFn() func(ChunkPos) (hash int)
	Set(SubChunkPos, Data)
}

type SubChunkStorageSyncProvider[Change any] interface {
	// 用于同步数据
	QueryChange(SubChunkPos, ChunkVersion) SubChunkChange[Change]
}

type SubChunkStorageSyncReciver[Change any] interface {
	// 用于同步数据
	GetVersion(SubChunkPos) ChunkVersion
	ApplyChange(SubChunkPos, SubChunkChange[Change])
}

type SubChunkStorage[Data any, Change any] interface {
	SubChunkStorageRead[Data]
	SubChunkStorageRandomRead[Data]
	SubChunkStorageRandomWrite[Data]
	SubChunkStorageSyncProvider[Change]
	SubChunkStorageSyncReciver[Change]
}
