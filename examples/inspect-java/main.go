// 从工作区根目录运行：go run ./neomega-celler/examples/inspect-java -world /path/to/world -x 0 -z 0
// 读取已有存档；x、z 是 16×16 区块柱子的坐标，输出 Y 是 16 格高的 section 编号。
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"celler"
	"celler/java"
	packfs "neomega-packfs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run 将错误交给 main 报告，让打开的句柄能先通过 defer 释放。
func run() error {
	// 1. 选择已有世界与区块；只读模式不会写回区块。
	directory := flag.String("world", "", "Java 世界目录")
	container := flag.String("pack", "", "以 Java 世界目录为根的 PackFS 文件")
	dimension := flag.String("dimension", ".", "维度相对目录：主世界 .、下界 DIM-1、末地 DIM1")
	x := flag.Int("x", 0, "区块 X")
	z := flag.Int("z", 0, "区块 Z")
	flag.Parse()
	if (*directory == "") == (*container == "") {
		return fmt.Errorf("-world 与 -pack 必须且只能指定一个")
	}
	// Java 世界借用文件后端；PackFS 必须在 world 之后关闭。
	var files fs.FS
	var pack *packfs.PackFS
	if *container != "" {
		if _, err := os.Stat(*container); err != nil {
			return err
		}
		var err error
		pack, err = packfs.Open(*container)
		if err != nil {
			return err
		}
		defer pack.Close()
		files = pack
	} else {
		files = os.DirFS(*directory)
	}
	// 2. 世界句柄管理底层文件；读取的区块归调用方所有。
	var world *java.World
	var err error
	world, err = java.Open(files, java.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer world.Close()
	// 3. ReadChunk 读取整根柱子，Section 按需解码每个高度。
	var chunk *java.Chunk
	chunk, err = world.ReadChunk(java.Dimension(*dimension), celler.ChunkPos{X: int32(*x), Z: int32(*z)})
	if err != nil {
		return err
	}
	var ys []int32
	ys, err = chunk.SectionYs()
	if err != nil {
		return err
	}
	for _, y := range ys {
		var section *java.Section
		section, err = chunk.Section(y)
		if err != nil {
			return err
		}
		count := 0
		// 仅含光照的 section 可以没有方块字段；nil 与全空气层不同。
		if section.Blocks != nil {
			count = len(section.Blocks.GetUsedPalette())
		}
		fmt.Printf("Y=%d palette=%d biomes=%t\n", y, count, section.Biomes != nil)
	}
	// 成功路径显式检查关闭错误；defer 也负责中途报错时的清理，Close 可重复调用。
	if err = world.Close(); err != nil {
		return err
	}
	if pack != nil {
		return pack.Close()
	}
	return nil
}
