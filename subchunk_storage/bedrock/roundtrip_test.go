package bedrock

import (
	"bytes"
	"celler"
	"celler/internal/nbtutil"
	"celler/kvdb"
	"celler/layer_storage"
	"fmt"
	"neomega-blocks/ir"
	blocks "neomega-blocks/presets/block-converts/v1"
	nbt "neomega-nbt/presets/v1"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBedrockStorageRoundTrip(t *testing.T) {
	air := ir.IRBlock("minecraft:air []").Hash()
	stone, ok := blocks.ConvertFromBedrock(ir.IRBlock("minecraft:stone []"))
	if !ok {
		t.Fatal("stone missing from preset")
	}
	path := filepath.Join(t.TempDir(), "db")
	db, err := kvdb.NewReadWriteLevelDB(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := NewReadWriteBedrockStorage(db, 0, air, DefaultBlockDecode, DefaultBlockEncode)
	pos := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: -2, Z: 3}, Y: -4}
	sc := writer.GetOrEmpty(pos)
	sc.CurrentVersion = 1
	blockPos := celler.ToSubChunkBlockPos(5, 6, 7)
	sc.BlockIDStorage.Set(blockPos, stone.Hash())
	entity := map[string]any{"id": "Chest", "Items": []any{}, "data": []int32{1, 2, 3}}
	entitySNBT, err := nbtutil.ToSNBT(entity)
	if err != nil {
		t.Fatal(err)
	}
	sc.BlockSNBTStorage.Set(blockPos, celler.SNBT(entitySNBT))
	writer.Set(pos, sc)
	// Two concatenated block-entity NBT roots in one chunk.
	second := celler.ToSubChunkBlockPos(1, 2, 3)
	sc.BlockSNBTStorage.Set(second, celler.SNBT(`{id:"Chest",Items:[]}`))
	writer.Set(pos, sc)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := kvdb.NewReadOnlyLevelDB(path)
	if err != nil {
		t.Fatal(err)
	}
	reader := NewReadOnlyBedrockStorage(ro, 0, air, DefaultBlockDecode)
	got := reader.Get(pos)
	if got == nil {
		t.Fatal("written subchunk missing")
	}
	if got.BlockIDStorage.Get(blockPos) != stone.Hash() || got.BlockIDStorage.Get(second) != air {
		t.Fatal("block data changed")
	}
	actual, err := nbtutil.FromSNBT(string(got.BlockSNBTStorage.Get(blockPos)))
	if err != nil || !reflect.DeepEqual(actual, entity) {
		t.Fatalf("entity changed: %v, %v", actual, err)
	}
	if got.BlockSNBTStorage.Get(second) == "" {
		t.Fatal("second entity lost")
	}
	if got.BlockSNBTStorage.Get(celler.ToSubChunkBlockPos(0, 0, 0)) != "" {
		t.Fatal("missing SNBT is not empty")
	}
	count := 0
	for p := range reader.IterAll {
		if p != pos {
			t.Fatalf("wrong position %v", p)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("enumerated %d chunks", count)
	}
	ro.Close()
	db, err = kvdb.NewReadWriteLevelDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	writer = NewReadWriteBedrockStorage(db, 0, air, DefaultBlockDecode, DefaultBlockEncode)
	writer.Drop(pos)
	if writer.Get(pos) != nil {
		t.Fatal("dropped subchunk still exists")
	}
}

func TestDefaultBlockDecodeCanonicalizesMissingProperties(t *testing.T) {
	raw, err := nbtutil.Encode(map[string]any{"name": "minecraft:oak_stairs", "states": map[string]any{}}, nbt.EncodingLittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := DefaultBlockDecode(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, ok := blocks.ConvertFromBedrock(ir.IRBlock("minecraft:oak_stairs []"))
	if !ok || hash != want.Hash() {
		t.Fatal("missing properties were not canonicalized")
	}
	encoded, err := DefaultBlockEncode(hash)
	if err != nil {
		t.Fatal(err)
	}
	roundtrip, err := DefaultBlockDecode(encoded)
	if err != nil || roundtrip != hash {
		t.Fatalf("roundtrip %d, %v", roundtrip, err)
	}
}

// A Reader may return fewer bytes than requested without reaching EOF.
type shortReader struct{ *bytes.Reader }

func (r shortReader) Read(p []byte) (int, error) {
	if len(p) > 3 {
		p = p[:3]
	}
	return r.Reader.Read(p)
}

func TestSubChunkShortReads(t *testing.T) {
	air := ir.IRBlock("minecraft:air []").Hash()
	db := kvdb.NewMockKVDB()
	storage := NewReadWriteBedrockStorage(db, 0, air, DefaultBlockDecode, DefaultBlockEncode)
	sc := storage.GetOrEmpty(celler.SubChunkPos{})
	var buf bytes.Buffer
	storage.EncodeSubChunkChunkBlocks(&buf, 0, sc.BlockIDStorage)
	dec := ChunkDecoder[uint32]{DefaultBlockID: air, BlockDecode: DefaultBlockDecode}
	got, err := dec.DecodeSubChunkBlocks(shortReader{bytes.NewReader(buf.Bytes())}, 0)
	if err != nil || got.Default() != air {
		t.Fatalf("short read: %v", err)
	}
}

func TestDiskPaletteWidths(t *testing.T) {
	encode := func(id uint32) ([]byte, error) {
		return nbtutil.Encode(map[string]any{"id": int32(id)}, nbt.EncodingLittleEndian)
	}
	decode := func(raw []byte) (uint32, error) {
		value, err := nbtutil.Decode(raw, nbt.EncodingLittleEndian)
		if err != nil {
			return 0, err
		}
		return uint32(value["id"].(int32)), nil
	}
	encoder := ChunkEncoder[uint32]{BlockEncode: encode}
	decoder := ChunkDecoder[uint32]{BlockDecode: decode}
	for _, count := range []int{1, 2, 65, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			palette := layer_storage.New(uint32(0)).Palette()
			for i := 1; i < count; i++ {
				palette.Set(celler.ToSubChunkBlockPos(i>>8, i&15, i>>4&15), uint32(i))
			}
			var buf bytes.Buffer
			encoder.EncodeSubChunkChunkBlocks(&buf, -4, palette)
			width := buf.Bytes()[3] >> 1
			if width == 7 || width == 9 {
				t.Fatalf("unsupported disk width %d", width)
			}
			got, err := decoder.DecodeSubChunkBlocks(bytes.NewReader(buf.Bytes()), -4)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 4096; i++ {
				pos := celler.ToSubChunkBlockPos(i>>8, i&15, i>>4&15)
				if got.Get(pos) != palette.Get(pos) {
					t.Fatalf("wrong value at %v", pos)
				}
			}
		})
	}
	// Zero-bit disk storage contains one NBT entry and no palette count.
	raw, err := encode(42)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decoder.DecodeSubChunkBlocks(bytes.NewReader(append([]byte{9, 1, 0, 0}, raw...)), 0)
	if err != nil || got.Default() != 42 {
		t.Fatalf("zero-bit palette: %v", err)
	}
}
