package java

import (
	"celler"
	"celler/internal/nbtutil"
	"celler/subchunk_storage/java/mca"
	"errors"
	"neomega-blocks/ir"
	blocks "neomega-blocks/presets/block-converts/v1"
	nbt "neomega-nbt/presets/v1"
	"reflect"
	"testing"
)

func encodeFixture(t *testing.T, value map[string]any) []byte {
	t.Helper()
	raw, err := nbtutil.Encode(value, nbt.EncodingBigEndian)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestJavaUniformSectionAndStorageRoundTrip(t *testing.T) {
	cp := celler.ChunkPos{X: -2, Z: 3}
	position := celler.SubChunkPos{ChunkPos: cp, Y: -4}
	// Java's single-entry palette has no packed data array.
	raw := encodeFixture(t, map[string]any{
		"sections": []any{map[string]any{
			"Y":            int8(-4),
			"block_states": map[string]any{"palette": []any{map[string]any{"Name": "minecraft:stone"}}},
		}},
		"block_entities": []any{map[string]any{"x": int32(-30), "y": int32(-61), "z": int32(52), "id": "minecraft:chest", "Items": []any{}}},
	})
	chunk, err := cd.DecodeChunkFromData(raw)
	if err != nil {
		t.Fatal(err)
	}
	sc := chunk.Subchunks[-4]
	stone, ok := blocks.ConvertFromJava(ir.IRBlock("minecraft:stone []"))
	if !ok {
		t.Fatal("stone missing from preset")
	}
	if sc == nil || sc.BlockIDStorage.Get(celler.ToSubChunkBlockPos(0, 0, 0)) != stone.Hash() {
		t.Fatal("uniform stone section lost")
	}
	gold, ok := blocks.ConvertFromJava(ir.IRBlock("minecraft:gold_block []"))
	if !ok {
		t.Fatal("gold missing from preset")
	}
	changedPos := celler.ToSubChunkBlockPos(5, 6, 7)
	sc.BlockIDStorage.Set(changedPos, gold.Hash())
	entityPos := celler.ToSubChunkBlockPos(2, 3, 4)
	expectedSNBT, err := nbtutil.FromSNBT(string(sc.BlockSNBTStorage.Get(entityPos)))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	db, err := mca.NewReadWriteMCAKvFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	writer := NewReadWriteJavaStorage(db)
	if writer.Get(position) != nil {
		t.Fatal("fixture should start empty")
	}
	writer.Set(position, sc) // Also exercises a cached missing chunk.
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := mca.NewReadOnlyMCAKvFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	reader := NewReadOnlyJavaStorage(ro)
	got := reader.Get(position)
	if got == nil {
		t.Fatal("written section missing")
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				pos := celler.ToSubChunkBlockPos(x, y, z)
				want := stone.Hash()
				if pos == changedPos {
					want = gold.Hash()
				}
				if got.BlockIDStorage.Get(pos) != want {
					t.Fatalf("wrong block at %v", pos)
				}
			}
		}
	}
	actualSNBT, err := nbtutil.FromSNBT(string(got.BlockSNBTStorage.Get(entityPos)))
	if err != nil || !reflect.DeepEqual(actualSNBT, expectedSNBT) {
		t.Fatalf("entity changed: %v, %v", actualSNBT, err)
	}
}

type failingMCA struct {
	mca.ReadWriteMCAChunkKV
	err error
}

func (db failingMCA) Put(celler.ChunkPos, []byte) error { return db.err }

func TestJavaCloseReportsWriteErrors(t *testing.T) {
	failure := errors.New("disk write failed")
	for _, count := range []int{1, 65} {
		writer := NewReadWriteJavaStorage(failingMCA{err: failure})
		for i := 0; i < count; i++ {
			writer.cache.Add(celler.ChunkPos{X: celler.ChunkUnit(i)}, &ChunkWithChangeMark{Changed: true})
		}
		if err := writer.Close(); !errors.Is(err, failure) {
			t.Fatalf("%d cached chunks: %v", count, err)
		}
	}
}
