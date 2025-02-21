package celler

type SubChunkStorageRead[Data any] interface {
	IterAll(func(SubChunkPos) (next bool))
	Get(SubChunkPos) Data
	GetOrEmpty(SubChunkPos) Data
}

type SubChunkStorageWrite[Data any] interface {
	Set(SubChunkPos, Data)
	Drop(SubChunkPos)
}

type SubChunkStorage[Data any] interface {
	SubChunkStorageRead[Data]
	SubChunkStorageWrite[Data]
}

type SubChunkStorageSyncProvider[Change any] interface {
	// 用于同步数据
	QueryChange(SubChunkPos, ChunkVersion) Change
}

type SubChunkStorageSyncReciver[Change any] interface {
	// 用于同步数据
	ApplyChange(SubChunkPos, Change)
}

// type SubChunkBatchOperator[Data any] interface {
// 	BatchGet([]SubChunkPos) []struct {
// 		Pos  SubChunkPos
// 		Data Data
// 	}
// 	BatchSet(map[SubChunkPos]Data)
// }

// type SubChunkChange[Change any] struct {
// 	// case 1. 通常更新: Valid=false, Drop=false
// 	// 从 空 或现有 Cache 更新到目标
// 	// case 2. 移除数据: Valid=false,Drop=true
// 	// 删除现有 Cache
// 	// case 3. 不变更: Valid=true
// 	Valid  bool
// 	Drop   bool
// 	Change *Change
// }

// type SubChunkStorageSyncer[Change any] interface {
// 	SubChunkStorageSyncProvider[Change]
// 	SubChunkStorageSyncReciver[Change]
// }
