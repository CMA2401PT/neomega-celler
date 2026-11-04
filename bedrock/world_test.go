package bedrock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"celler"
	"celler/layer_storage"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"neomega-blocks/ir"
	nbt "neomega-nbt/presets/v1"
)

func TestNativeLayersAndPaletteVersions(t *testing.T) {
	// An independently assembled three-layer version-9 record. Uniform palettes
	// use width zero, so no index words or palette count are present on disk.
	raw := []byte{9, 3, 252}
	documents := []nbt.Document{}
	for i := range 3 {
		doc := nbt.Document{Name: "palette", Value: nbt.Compound{
			"name": fmt.Sprintf("example:unknown_%d", i), "states": nbt.Compound{"direction": int32(i)},
			"version": int32(18000000 + i), "extra": []int32{1, 2},
		}}
		documents = append(documents, doc)
		encoded, err := nbt.EncodeDocument(doc, nbt.EncodingLittleEndian)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, 0)
		raw = append(raw, encoded...)
	}
	s, err := DecodeSection(raw, -4)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Layers) != 3 {
		t.Fatal("lost layers")
	}
	for i, l := range s.Layers {
		if l.Blocks.Default().Version != int32(18000000+i) {
			t.Fatal("lost palette version")
		}
	}
	changed := Block{State: ir.IRBlock(`example:new ["powered"=true]`), Version: 19000000}
	s.Layers[1].Blocks.Set(layer_storage.At(2, 5, 7), changed)
	encoded, err := s.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSection(encoded, -4)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Layers[1].Blocks.Get(layer_storage.At(2, 5, 7)) != changed {
		t.Fatal("changed block lost")
	}
	for i, l := range decoded.Layers {
		if !reflect.DeepEqual(l.entries[0], documents[i]) {
			t.Fatal("palette NBT lost")
		}
	}
	for _, bad := range [][]byte{{9, 1, 252}, {9, 1, 252, 3}, {9, 0, 252, 99}, {99}} {
		if _, err := DecodeSection(bad, -4); err == nil {
			t.Fatalf("accepted invalid record %x", bad)
		}
	}
	for _, version := range []byte{1, 8, 9} {
		s := &Section{Y: 3, Version: version, Layers: []*Layer{NewLayer(Block{State: "minecraft:air []", Version: 1})}}
		for i := 0; i < 33; i++ {
			s.Layers[0].Blocks.Set(layer_storage.At(i&15, 3, i>>4), Block{State: ir.IRBlock(fmt.Sprintf("example:block_%d []", i)), Version: int32(i)})
		}
		raw, err := s.Encode()
		if err != nil {
			t.Fatal(err)
		}
		back, err := DecodeSection(raw, 3)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 4096; i++ {
			p := layer_storage.At(i>>8, i&15, i>>4&15)
			if back.Layers[0].Blocks.Get(p) != s.Layers[0].Blocks.Get(p) {
				t.Fatalf("version %d, position %v", version, p)
			}
		}
	}
}

func TestWorldDirectoryRecordsAndSnapshots(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "world")
	w, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	pos := celler.ChunkPos{X: -35, Z: 33}
	section := &Section{Y: -4, Version: 9, Layers: []*Layer{
		NewLayer(Block{State: "example:base []", Version: 10}),
		NewLayer(Block{State: "example:fluid []", Version: 11}),
	}}
	for _, dim := range []int32{0, 1, 2, 257} {
		c := NewChunk(pos, dim)
		if err = c.SetSection(section); err != nil {
			t.Fatal(err)
		}
		c.Records[string([]byte{0x40})] = []byte{5, 6, 7}
		if err = c.SetBlockEntities([]nbt.Document{{Value: nbt.Compound{"id": "Chest", "x": int32(-559), "Items": []nbt.Compound{}, "list": []int32{1, 2}}}}); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteChunk(c, &opt.WriteOptions{Sync: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.DB.Put([]byte("unrelated"), []byte("preserve"), nil); err != nil {
		t.Fatal(err)
	}
	level := Level{Version: 10, NBT: nbt.Document{Value: nbt.Compound{"LevelName": "test", "ints": []int32{1, 2}}}}
	if err = w.WriteLevel(level); err != nil {
		t.Fatal(err)
	}
	count := 0
	if err = w.ScanChunks(func(c *Chunk) error {
		count++
		if c.Pos != pos {
			return fmt.Errorf("wrong coordinates")
		}
		if err := w.DB.Delete(append(ChunkPrefix(pos, c.Dimension), 0x40), nil); err != nil {
			return err
		}
		if !bytes.Equal(c.Records[string([]byte{0x40})], []byte{5, 6, 7}) {
			return fmt.Errorf("snapshot bytes changed")
		}
		// Write a chunk from the scan: the read snapshot remains stable.
		return w.WriteChunk(c, nil)
	}); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatal(count)
	}
	c, err := w.ReadChunk(0, pos)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.DeleteSection(-4); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteChunk(c, nil); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := Open(dir, Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	c, err = ro.ReadChunk(0, pos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Section(-4); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	c, err = ro.ReadChunk(1, pos)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Section(-4)
	if err != nil || len(s.Layers) != 2 {
		t.Fatal(s, err)
	}
	docs, err := c.BlockEntities()
	if err != nil || docs[0].Value["id"] != "Chest" {
		t.Fatal(docs, err)
	}
	back, err := ro.ReadLevel()
	if err != nil || !reflect.DeepEqual(back, level) {
		t.Fatal(back, err)
	}
	if err = ro.WriteChunk(c, nil); !errors.Is(err, fs.ErrPermission) {
		t.Fatal(err)
	}
	if _, err = ro.ReadChunk(0, celler.ChunkPos{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	data, err := ro.DB.Get([]byte("unrelated"), nil)
	if err != nil || string(data) != "preserve" {
		t.Fatal(string(data), err)
	}
}

func TestBiomeRecordRoundTrip(t *testing.T) {
	c := NewChunk(celler.ChunkPos{}, 0)
	b := &Biomes3D{Sections: []BiomeSection{{Values: layer_storage.MakePaletteStorage(int32(7)), Runtime: true}, {}}}
	b.Heights[17] = -20
	b.Sections[0].Values.Set(layer_storage.At(3, 6, 9), 33)
	if err := c.SetBiomes3D(b); err != nil {
		t.Fatal(err)
	}
	back, err := c.Biomes3D()
	if err != nil {
		t.Fatal(err)
	}
	if back.Heights != b.Heights || len(back.Sections) != 2 || back.Sections[1].Values != nil || !back.Sections[0].Runtime {
		t.Fatal(back)
	}
	if back.Sections[0].Values.Get(layer_storage.At(3, 6, 9)) != 33 {
		t.Fatal("lost biome")
	}
	var b2 Biomes2D
	b2.Heights[0] = 2
	b2.Values[255] = 19
	c.SetBiomes2D(b2)
	back2, err := c.Biomes2D()
	if err != nil || back2 != b2 {
		t.Fatal(err)
	}
	// A 3-bit palette whose index is out of bounds must be an error.
	data := []byte{9, 1, 0, 6}
	words := make([]byte, ((4096+9)/10)*4)
	binary.LittleEndian.PutUint32(words, 7)
	data = append(data, words...)
	data = binary.LittleEndian.AppendUint32(data, 0)
	if _, err := DecodeSection(data, 0); err == nil {
		t.Fatal("accepted invalid palette")
	}
}

func TestReadChunkActors(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "world")
	w, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	pos := celler.ChunkPos{X: 2, Z: -3}
	actorID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	value, err := nbt.EncodeDocument(nbt.Document{Value: nbt.Compound{
		"identifier": "minecraft:zombie",
		"Pos":        []float32{0, 64, 0},
	}}, nbt.EncodingLittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.DB.Put(append([]byte("actorprefix"), actorID...), value, nil); err != nil {
		t.Fatal(err)
	}
	// digp 的值是每 8 字节一个实体 id；第二条记录不存在，应被跳过而不是报错。
	ids := append([]byte{}, actorID...)
	ids = append(ids, []byte{9, 9, 9, 9, 9, 9, 9, 9}...)
	digp := append([]byte("digp"), ChunkPrefix(pos, 1)...)
	if err = w.DB.Put(digp, ids, nil); err != nil {
		t.Fatal(err)
	}

	documents, err := w.ReadChunkActors(1, pos)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].Value["identifier"] != "minecraft:zombie" {
		t.Fatalf("actors: %#v", documents)
	}

	documents, err = w.ReadChunkActors(0, pos)
	if err != nil || len(documents) != 0 {
		t.Fatalf("missing digp: %#v, %v", documents, err)
	}

	if err = w.DB.Put(append([]byte("digp"), ChunkPrefix(celler.ChunkPos{}, 0)...), []byte{1}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = w.ReadChunkActors(0, celler.ChunkPos{}); err == nil {
		t.Fatal("accepted malformed digp")
	}
}

func TestWriteChunkActors(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "world")
	w, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	pos := celler.ChunkPos{X: -1, Z: 4}
	actors := []Actor{
		{ID: [8]byte{1, 2, 3, 4, 5, 6, 7, 8}, NBT: nbt.Document{Value: nbt.Compound{"identifier": "minecraft:zombie"}}},
		{ID: [8]byte{9, 10, 11, 12, 13, 14, 15, 16}, NBT: nbt.Document{Value: nbt.Compound{"identifier": "minecraft:pig"}}},
	}
	if err := w.WriteChunkActors(1, pos, actors); err != nil {
		t.Fatal(err)
	}
	documents, err := w.ReadChunkActors(1, pos)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 2 || documents[0].Value["identifier"] != "minecraft:zombie" || documents[1].Value["identifier"] != "minecraft:pig" {
		t.Fatalf("actors: %#v", documents)
	}
	if err := w.WriteChunkActors(1, pos, nil); err != nil {
		t.Fatal(err)
	}
	documents, err = w.ReadChunkActors(1, pos)
	if err != nil || len(documents) != 0 {
		t.Fatalf("cleared actors: %#v, %v", documents, err)
	}
}
