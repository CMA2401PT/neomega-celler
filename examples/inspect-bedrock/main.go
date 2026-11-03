// 从工作区根目录运行：go run ./neomega-celler/examples/inspect-bedrock -world /path/to/world -x 0 -z 0
// 读取已有存档；x、z 是 16×16 区块柱子的坐标，输出 Y 是 16 格高的 section 编号。
// LevelDB 即使只读打开也可能写辅助文件，请将 -world 指向存档副本。
package main

import (
	"flag"
	"fmt"
	"log"

	"celler"
	"celler/bedrock"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run 将错误交给 main 报告，让打开的句柄能先通过 defer 释放。
func run() error {
	// 1. 选择已有世界与区块；只读模式不会写回区块。
	worldPath := flag.String("world", "", "包含 db 的 Bedrock 世界目录")
	dimension := flag.Int("dimension", 0, "维度 ID：主世界 0、下界 1、末地 2")
	x := flag.Int("x", 0, "区块 X")
	z := flag.Int("z", 0, "区块 Z")
	flag.Parse()
	if *worldPath == "" {
		return fmt.Errorf("必须指定 -world")
	}
	// 2. 世界句柄管理底层文件；读取的区块归调用方所有。
	var world *bedrock.World
	var err error
	world, err = bedrock.Open(*worldPath, bedrock.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer world.Close()
	// 3. ReadChunk 读取整根柱子，Section 按需解码每个高度。
	var chunk *bedrock.Chunk
	chunk, err = world.ReadChunk(int32(*dimension), celler.ChunkPos{X: int32(*x), Z: int32(*z)})
	if err != nil {
		return err
	}
	for _, y := range chunk.SectionYs() {
		var section *bedrock.Section
		section, err = chunk.Section(y)
		if err != nil {
			return err
		}
		fmt.Printf("Y=%d version=%d layers=%d\n", y, section.Version, len(section.Layers))
		for i, layer := range section.Layers {
			fmt.Printf("  layer=%d palette=%d\n", i, len(layer.Blocks.GetUsedPalette()))
		}
	}
	// 成功路径检查关闭错误；defer 负责中途失败时清理，Close 可重复调用。
	return world.Close()
}
