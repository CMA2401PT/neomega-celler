// inspect-java reads a native Java world directory or a PackFS container.
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
func run() error {
	directory := flag.String("world", "", "Java world directory")
	container := flag.String("pack", "", "PackFS container whose root is the Java world")
	dimension := flag.String("dimension", ".", "relative dimension directory")
	x := flag.Int("x", 0, "chunk X")
	z := flag.Int("z", 0, "chunk Z")
	flag.Parse()
	if (*directory == "") == (*container == "") {
		return fmt.Errorf("supply exactly one of -world or -pack")
	}
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
	world, err := java.Open(files, java.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer world.Close()
	chunk, err := world.ReadChunk(java.Dimension(*dimension), celler.ChunkPos{X: int32(*x), Z: int32(*z)})
	if err != nil {
		return err
	}
	ys, err := chunk.SectionYs()
	if err != nil {
		return err
	}
	for _, y := range ys {
		section, err := chunk.Section(y)
		if err != nil {
			return err
		}
		count := 0
		if section.Blocks != nil {
			count = len(section.Blocks.GetUsedPalette())
		}
		fmt.Printf("Y=%d palette=%d biomes=%t\n", y, count, section.Biomes != nil)
	}
	if err = world.Close(); err != nil {
		return err
	}
	if pack != nil {
		return pack.Close()
	}
	return nil
}
