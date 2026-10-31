package java

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"celler"
	"celler/java/region"
	"celler/layer_storage"
	"celler/storage"
	"neomega-blocks/ir"
	nbt "neomega-nbt/presets/v1"
	packfs "neomega-packfs"
)

func exerciseWorld(t *testing.T, files storage.Writer) {
	t.Helper()
	w, err := Open(files, Options{MaxOpenRegions: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, p := range []celler.ChunkPos{{X: -33, Z: -1}, {X: 33, Z: 65}, {X: 0, Z: 0}} {
		c := NewChunk(p, 4189)
		c.NBT.Value["Status"] = "minecraft:full"
		if err = c.SetSection(&Section{Y: -4, Blocks: layer_storage.MakePaletteStorage(ir.IRBlock("example:stored []"))}); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteChunk(Overworld, c); err != nil {
			t.Fatal(err)
		}
	}
	record := region.Record{Compression: region.Uncompressed, Data: []byte("raw POI bytes"), Timestamp: 77}
	if err = w.WriteRaw(Nether, POI, celler.ChunkPos{X: -1, Z: 32}, record); err != nil {
		t.Fatal(err)
	}
	level := nbt.Document{Name: "level", Value: nbt.Compound{"Data": nbt.Compound{"LevelName": "native world", "intList": []int32{1, 2}}}}
	if err = w.WriteLevel(level); err != nil {
		t.Fatal(err)
	}
	count := 0
	if err = w.Scan(Overworld, Terrain, func(p celler.ChunkPos) error {
		c, err := w.ReadChunk(Overworld, p)
		if err != nil {
			return err
		}
		s, err := c.Section(-4)
		if err != nil {
			return err
		}
		if s.Blocks.Default() != "example:stored []" {
			return fmt.Errorf("wrong block")
		}
		count++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatal(count)
	}
	if err = w.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := Open(files, Options{ReadOnly: true, MaxOpenRegions: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	actual, err := ro.ReadLevel()
	if err != nil || !reflect.DeepEqual(actual, level) {
		t.Fatal(actual, err)
	}
	raw, err := ro.ReadRaw(Nether, POI, celler.ChunkPos{X: -1, Z: 32})
	if err != nil || !reflect.DeepEqual(raw, record) {
		t.Fatal(raw, err)
	}
	if err = ro.WriteRaw(Overworld, Terrain, celler.ChunkPos{}, record); !errors.Is(err, fs.ErrPermission) {
		t.Fatal(err)
	}
	if _, err = ro.ReadChunk(Overworld, celler.ChunkPos{X: 999, Z: 999}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestWorldBackendsAndSnapshotExport(t *testing.T) {
	dir, err := storage.OpenDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exerciseWorld(t, dir)
	containerPath := filepath.Join(t.TempDir(), "world.pack")
	p, err := packfs.Open(containerPath)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	exerciseWorld(t, storage.Pack{PackFS: p})
	if err = p.Snapshot("before"); err != nil {
		t.Fatal(err)
	}
	w, err := Open(storage.Pack{PackFS: p}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Delete(Overworld, Terrain, celler.ChunkPos{}); err != nil {
		t.Fatal(err)
	}
	if err = w.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	view, err := p.SnapshotFS("before")
	if err != nil {
		t.Fatal(err)
	}
	snap, err := Open(view, Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = snap.ReadChunk(Overworld, celler.ChunkPos{}); err != nil {
		t.Fatal("snapshot lost chunk", err)
	}
	if err = snap.Close(); err != nil {
		t.Fatal(err)
	}
	export := t.TempDir()
	if err = fs.WalkDir(view, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		local := filepath.Join(export, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(local, 0755)
		}
		data, err := fs.ReadFile(view, name)
		if err != nil {
			return err
		}
		return os.WriteFile(local, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	if err = view.Close(); err != nil {
		t.Fatal(err)
	}
	native, err := OpenDirectory(export, Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = native.ReadChunk(Overworld, celler.ChunkPos{}); err != nil {
		t.Fatal("exported directory failed", err)
	}
	if err = native.Close(); err != nil {
		t.Fatal(err)
	}
	if err = p.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := packfs.Open(containerPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	current, err := Open(storage.Pack{PackFS: reopened}, Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	if _, err = current.ReadChunk(Overworld, celler.ChunkPos{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("deletion not persisted", err)
	}
}

func TestRegionExternalAndRepeatedEdits(t *testing.T) {
	for _, backend := range []string{"directory", "packfs"} {
		t.Run(backend, func(t *testing.T) {
			var files storage.Writer
			if backend == "directory" {
				var err error
				files, err = storage.OpenDirectory(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
			} else {
				p, err := packfs.Open(filepath.Join(t.TempDir(), "large.pack"))
				if err != nil {
					t.Fatal(err)
				}
				defer p.Close()
				files = storage.Pack{PackFS: p}
			}
			w, err := Open(files, Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			pos := celler.ChunkPos{X: -1, Z: -33}
			large := bytes.Repeat([]byte{0x5a}, 1100000)
			for _, data := range [][]byte{large, []byte("small"), large, []byte("last")} {
				record := region.Record{Compression: region.Uncompressed, Data: data, Timestamp: 123}
				if err = w.WriteRaw(End, Entities, pos, record); err != nil {
					t.Fatal(err)
				}
				got, err := w.ReadRaw(End, Entities, pos)
				if err != nil || !bytes.Equal(got.Data, data) {
					t.Fatal("read before flush", err)
				}
				if err = w.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = fs.Stat(files, "DIM1/entities/c.-1.-33.mcc"); !errors.Is(err, fs.ErrNotExist) {
				t.Fatal("obsolete external file remains", err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			ro, err := Open(files, Options{ReadOnly: true})
			if err != nil {
				t.Fatal(err)
			}
			defer ro.Close()
			got, err := ro.ReadRaw(End, Entities, pos)
			if err != nil || string(got.Data) != "last" || got.Timestamp != 123 {
				t.Fatal(got, err)
			}
		})
	}
}
