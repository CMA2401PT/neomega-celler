package celler

type CanvasRead[Block any] interface {
	// Block 的类型由具体实现定义，
	// 可能是 block id，
	// 也可能是 string 或者 同时有 block 和 nbt， attach 信息
	GetBlock(x, y, z int) Block
}

type CanvasWrite[Block any] interface {
	// Block 的类型由具体实现定义，
	// 可能是 block id，
	// 也可能是 string 或者 同时有 block 和 nbt， attach 信息
	SetBlock(x, y, z int, block Block)
}
