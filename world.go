package celler

type DataWithPos[Data any] struct {
	Pos SubChunkPos
}

type WorldStaticServer[Change any] interface {
	QueryChange(SubChunkPos, ChunkVersion) SubChunkChange[Change]
	QueryChanges(<-chan *DataWithPos[ChunkVersion])
	Changes() <-chan *DataWithPos[SubChunkChange[Change]]
}

type WorldDynamcServer[Change any] interface {
	WorldStaticServer[Change]
	Watch(SubChunkArea)
}

type WorldProcessor[Change any] interface {
	ApplyChanges(<-chan *DataWithPos[SubChunkChange[Change]])
}

// type SubChunkQuery SubChunkPosWithVersion
// type SubChunkResponse[BlockIDT ~uint32, AttachT ~uint32] SubChunkChanges[BlockIDT, AttachT]

// // 模式: 数据提供者只能以特定顺序提供数据
// type BatchWorldProvider[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	IterSubChunk(fn func(subchunk SubChunkChanges[BlockIDT, AttachT]) (next bool))
// }

// // 模式: 客户/数据需求者知道自己需要什么/知道自己需要的范围，向提供者请求数据
// type StaticWorldServerAsync[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	// 当提供者是缓慢的，例如基于网络、文件读取等，可以提供异步实现
// 	QueryAsync(<-chan *SubChunkQuery) <-chan *SubChunkResponse[BlockIDT, AttachT]
// }
// type StaticWorldServerSync[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	// 当提供者几乎是即时的，例如使用 map 保存的，可以提供同步实现
// 	QueryInstance([]SubChunkQuery) []SubChunkResponse[BlockIDT, AttachT]
// }

// // 客户/数据需求者 要求监听服务器指定范围的数据变化
// // 或者指定未来一段时间集中访问的区域
// type DynamicWorld[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	// 客户/数据需求者 指明自己当前要监听的范围
// 	SetWatchRange(watchRange CenteredArea)
// 	// 客户/数据需求者 指明当数据发生变化时的处理函数
// 	SetChangeReciver(changeReciver WorldProcessor[BlockIDT, AttachT])
// }

// 模式: 客户/数据处理者不知道自己需要什么/不知道数据有哪些，数据提供者主动将数据推送
// type SubChunksProcessor[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	// 数据提供者提供基于缓存的数据变更
// 	PushChanges([]SubChunkChanges[BlockIDT, AttachT])
// }

// type WorldProcessorWithCacher[BlockIDT ~uint32, AttachT ~uint32] interface {
// 	WorldProcessor[BlockIDT, AttachT]
// 	// 数据提供者查询数据处理者当前缓存
// 	QueryCacheVersion([]SubChunkPos) []SubChunkPosWithVersion
// }
