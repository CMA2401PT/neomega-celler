package mcworld

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"celler"
	"celler/bedrock"
	"celler/layer_storage"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

func TestReadWithoutExtracting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "world")
	world, err := bedrock.Open(dir, bedrock.Options{})
	if err != nil {
		t.Fatal(err)
	}
	pos := celler.ChunkPos{X: -2, Z: 3}
	section := &bedrock.Section{Y: -4, Version: 9, Layers: []*bedrock.Layer{
		bedrock.NewLayer(bedrock.Block{State: "minecraft:stone []", Version: 7}),
	}}
	chunk := bedrock.NewChunk(pos, 0)
	if err = chunk.SetSection(section); err != nil {
		t.Fatal(err)
	}
	if err = world.WriteChunk(chunk, &opt.WriteOptions{Sync: true}); err != nil {
		t.Fatal(err)
	}
	if err = world.Close(); err != nil {
		t.Fatal(err)
	}
	for _, method := range []uint16{zip.Store, zip.Deflate} {
		for _, prefix := range []string{"", "nested/"} {
			dest := filepath.Join(t.TempDir(), "world.mcworld")
			if err = zipDirectory(dir, dest, method, prefix); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(dest)
			if err != nil {
				t.Fatalf("method %d prefix %q: %v", method, prefix, err)
			}
			got, err := opened.Bedrock().ReadChunk(0, pos)
			if err != nil {
				opened.Close()
				t.Fatal(err)
			}
			decoded, err := got.Section(-4)
			if err != nil {
				opened.Close()
				t.Fatal(err)
			}
			block := decoded.Layers[0].Blocks.Get(layer_storage.At(0, 0, 0))
			if block.State != "minecraft:stone []" || block.Version != 7 {
				opened.Close()
				t.Fatal(block)
			}
			if err = opened.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSampleWorld(t *testing.T) {
	path := filepath.Join("..", "..", "..", "resource", "全方块和物品 - 已导入-20261030-192755.mcworld")
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	opened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	name, err := fs.ReadFile(opened.Bedrock().Files, "levelname.txt")
	if err != nil || string(name) != "全方块和物品" {
		t.Fatalf("%q %v", name, err)
	}
	if _, err = opened.Bedrock().ReadLevel(); err != nil {
		t.Fatal(err)
	}
	found := errors.New("found")
	err = opened.Bedrock().ScanChunks(func(chunk *bedrock.Chunk) error {
		if len(chunk.SectionYs()) == 0 {
			return nil
		}
		if _, err := chunk.Section(chunk.SectionYs()[0]); err != nil {
			return err
		}
		return found
	})
	if !errors.Is(err, found) {
		t.Fatal(err)
	}
}

func zipDirectory(dir, dest string, method uint16, prefix string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	defer writer.Close()
	return filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(mustInfo(name))
		if err != nil {
			return err
		}
		header.Name = prefix + filepath.ToSlash(rel)
		header.Method = method
		target, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(target, file)
		return err
	})
}

func mustInfo(name string) fs.FileInfo {
	info, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	return info
}
