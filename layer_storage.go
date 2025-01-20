package celler

type LayerStorageRead[D any] interface {
	// 默认的值
	Default() D
	// 所有非默认值
	IterNonDefault(fn func(SubChunkBlockPos, D) (next bool))
}

type LayerStorageWrite[D any] interface {
	// 初始化并填充所有位置
	InitWithDefault(data D)
	// 核心实现，考虑A <-> B 两个系统，它们可能使用不同的存储方式
	// 例如出于内存考虑，A的存贮方式更省内存，且是只读的，但是B的存储方式访问和操作更简便
	// 我们使用这个方法快速、轻松的完成两个系统存贮方式的转换
	CopyFrom(LayerStorageRead[D])
}

type LayerStorageRandomRead[D any] interface {
	LayerStorageRead[D]
	Get(SubChunkBlockPos) D
}

type LayerStorageRandomWrite[D any] interface {
	LayerStorageWrite[D]
	// 设置某个特定位置
	Set(SubChunkBlockPos, D)
}

type Storage[D any] interface {
	LayerStorageRandomRead[D]
	LayerStorageRandomWrite[D]
}

type StorageMaker[D any, StorageT Storage[D]] interface {
	// 工厂函数，对于范型有帮助
	Make() StorageT
}
