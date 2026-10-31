package java

import (
	"fmt"
	"reflect"
	"testing"

	"celler"
	"celler/layer_storage"
	"neomega-blocks/ir"
	nbt "neomega-nbt/presets/v1"
)

func TestSectionEditPreservesNativeNBT(t *testing.T) {
	entries := make([]nbt.Compound, 17)
	for i := range entries {
		entries[i] = nbt.Compound{"Name": fmt.Sprintf("example:block_%d", i), "Properties": nbt.Compound{"waterlogged": "true"}, "custom": int16(i)}
	}
	// Independently pack Java Y-Z-X indexes into padded 64-bit words, 12 per word.
	words := make([]int64, (4096+11)/12)
	for i := 0; i < 4096; i++ {
		words[i/12] |= int64(i%17) << ((i % 12) * 5)
	}
	unknown := nbt.Compound{"intList": []int32{1, 2}, "intArray": nbt.Array([]int32{1, 2}), "emptyLongList": []int64{}, "emptyCompoundList": []nbt.Compound{}}
	original := nbt.Document{Name: "root", Value: nbt.Compound{
		"xPos": int32(-33), "zPos": int32(65), "DataVersion": int32(4189), "unknown": unknown,
		"block_entities": []nbt.Compound{{"id": "minecraft:chest", "x": int32(-528), "Items": []nbt.Compound{}}},
		"sections": []nbt.Compound{
			{"Y": int8(-4), "SkyLight": nbt.Array([]byte{1, 2, 3}), "block_states": nbt.Compound{"palette": entries, "data": nbt.Array(words), "extra": "keep"}, "biomes": nbt.Compound{"palette": []string{"example:biome"}}},
			{"Y": int8(24), "BlockLight": nbt.Array([]byte{4, 5})},
		},
	}}
	data, err := nbt.EncodeDocument(original, nbt.EncodingBigEndian)
	if err != nil {
		t.Fatal(err)
	}
	c, err := DecodeChunk(data)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Section(-4)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4096; i++ {
		want := ir.IRBlock(fmt.Sprintf("example:block_%d [\"waterlogged\"=\"true\"]", i%17))
		p := layer_storage.At(i&15, i>>8, i>>4&15)
		if got := s.Blocks.Get(p); got != want {
			t.Fatalf("%v: %s != %s", p, got, want)
		}
	}
	s.Blocks.Set(layer_storage.At(3, 7, 9), `example:edited ["facing"="north"]`)
	s.Biomes[19] = "example:other_biome"
	if err = c.SetSection(s); err != nil {
		t.Fatal(err)
	}
	encoded, err := c.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeChunk(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if back.NBT.Name != "root" || !reflect.DeepEqual(back.NBT.Value["unknown"], unknown) || !reflect.DeepEqual(back.NBT.Value["block_entities"], original.Value["block_entities"]) {
		t.Fatal("root metadata or entity changed")
	}
	after, err := back.Section(-4)
	if err != nil {
		t.Fatal(err)
	}
	if after.Blocks.Get(layer_storage.At(3, 7, 9)) != `example:edited ["facing"="north"]` || after.Biomes[19] != "example:other_biome" {
		t.Fatal("edit lost")
	}
	if !reflect.DeepEqual(after.entries[0], entries[0]) || after.NBT["block_states"].(nbt.Compound)["extra"] != "keep" {
		t.Fatal("palette metadata lost")
	}
	light, err := back.Section(24)
	if err != nil || light.Blocks != nil || !reflect.DeepEqual(light.NBT["BlockLight"], nbt.Array([]byte{4, 5})) {
		t.Fatal("light-only section lost", err)
	}
	ys, err := back.SectionYs()
	if err != nil || !reflect.DeepEqual(ys, []int32{-4, 24}) {
		t.Fatal(ys, err)
	}
}

func TestSinglePaletteAndMalformedData(t *testing.T) {
	c := NewChunk(celler.ChunkPos{}, 4189)
	s := &Section{Y: 0, Blocks: layer_storage.MakePaletteStorage(ir.IRBlock("example:not_air []"))}
	if err := c.SetSection(s); err != nil {
		t.Fatal(err)
	}
	back, err := c.Section(0)
	if err != nil || back.Blocks.Get(layer_storage.At(15, 15, 15)) != "example:not_air []" {
		t.Fatal(back, err)
	}
	raw := c.NBT.Value["sections"].([]nbt.Compound)[0]["block_states"].(nbt.Compound)
	raw["palette"] = []nbt.Compound{{"Name": "minecraft:air"}, {"Name": "minecraft:stone"}}
	raw["data"] = nbt.Array([]int64{0})
	if _, err = c.Section(0); err == nil {
		t.Fatal("accepted truncated data")
	}
	words := make([]int64, 256)
	words[0] = 3
	raw["data"] = nbt.Array(words)
	if _, err = c.Section(0); err == nil {
		t.Fatal("accepted out-of-range index")
	}
}
