// inspect-bedrock prints native section layer counts from an existing world.
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
func run() error {
	worldPath := flag.String("world", "", "Bedrock world directory containing db")
	dimension := flag.Int("dimension", 0, "native dimension ID")
	x := flag.Int("x", 0, "chunk X")
	z := flag.Int("z", 0, "chunk Z")
	flag.Parse()
	if *worldPath == "" {
		return fmt.Errorf("-world is required")
	}
	world, err := bedrock.Open(*worldPath, bedrock.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer world.Close()
	chunk, err := world.ReadChunk(int32(*dimension), celler.ChunkPos{X: int32(*x), Z: int32(*z)})
	if err != nil {
		return err
	}
	for _, y := range chunk.SectionYs() {
		section, err := chunk.Section(y)
		if err != nil {
			return err
		}
		fmt.Printf("Y=%d version=%d layers=%d\n", y, section.Version, len(section.Layers))
		for i, layer := range section.Layers {
			fmt.Printf("  layer=%d palette=%d\n", i, len(layer.Blocks.GetUsedPalette()))
		}
	}
	return world.Close()
}
