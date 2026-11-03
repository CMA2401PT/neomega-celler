package java

import (
	"reflect"
	"testing"

	"celler"
	"celler/layer_storage"
	"neomega-blocks/ir"
	nbt "neomega-nbt/presets/v1"
)

func TestModernPaletteReadAndWrite(t *testing.T) {
	// 从真实 NBT 编解码边界验证紧凑列表、混合列表和现代字段，并保留未修改条目。
	for _, test := range []struct {
		name    string
		palette any
		want    []ir.IRBlock
	}{
		{
			name:    "compact",
			palette: []string{"minecraft:grass_block", "minecraft:air"},
			want:    []ir.IRBlock{`minecraft:grass_block ["snowy"="false"]`, "minecraft:air []"},
		},
		{
			name: "mixed",
			palette: []nbt.Compound{
				{"": "minecraft:grass_block"},
				{"id": "minecraft:grass_block", "properties": nbt.Compound{"snowy": "true"}, "custom": int16(7)},
			},
			want: []ir.IRBlock{`minecraft:grass_block ["snowy"="false"]`, `minecraft:grass_block ["snowy"="true"]`},
		},
		{
			name: "modern_compounds",
			palette: []nbt.Compound{
				{"id": "minecraft:grass_block"},
				{"id": "example:unknown", "properties": nbt.Compound{"custom": "yes"}},
			},
			want: []ir.IRBlock{`minecraft:grass_block ["snowy"="false"]`, `example:unknown ["custom"="yes"]`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// 两个相邻方块分别引用调色板前两项，其余位置使用首项。
			words := make([]int64, 256)
			words[0] = 1 << 4
			original := nbt.Document{Value: nbt.Compound{
				"xPos": int32(0), "zPos": int32(0), "DataVersion": int32(5023),
				"sections": []nbt.Compound{{
					"Y":            int8(-4),
					"block_states": nbt.Compound{"palette": test.palette, "data": nbt.Array(words)},
				}},
			}}
			data, err := nbt.EncodeDocument(original, nbt.EncodingBigEndian)
			if err != nil {
				t.Fatal(err)
			}
			chunk, err := DecodeChunk(data)
			if err != nil {
				t.Fatal(err)
			}
			section, err := chunk.Section(-4)
			if err != nil {
				t.Fatal(err)
			}
			for x, want := range test.want {
				if got := section.Blocks.Get(layer_storage.At(x, 0, 0)); got != want {
					t.Fatalf("x=%d: got %s, want %s", x, got, want)
				}
			}
			if err := chunk.SetSection(section); err != nil {
				t.Fatal(err)
			}
			raw := chunk.NBT.Value["sections"].([]nbt.Compound)[0]["block_states"].(nbt.Compound)
			if !reflect.DeepEqual(raw["palette"], test.palette) {
				t.Fatalf("unmodified palette changed: %#v", raw["palette"])
			}

			section.Blocks.Set(layer_storage.At(2, 0, 0), `minecraft:oak_log ["axis"="x"]`)
			if err := chunk.SetSection(section); err != nil {
				t.Fatal(err)
			}
			encoded, err := chunk.Encode()
			if err != nil {
				t.Fatal(err)
			}
			back, err := DecodeChunk(encoded)
			if err != nil {
				t.Fatal(err)
			}
			after, err := back.Section(-4)
			if err != nil {
				t.Fatal(err)
			}
			if got := after.Blocks.Get(layer_storage.At(2, 0, 0)); got != `minecraft:oak_log ["axis"="x"]` {
				t.Fatalf("edited state lost: %s", got)
			}
			for x, want := range test.want {
				if got := after.Blocks.Get(layer_storage.At(x, 0, 0)); got != want {
					t.Fatalf("unchanged x=%d: got %s, want %s", x, got, want)
				}
			}
			raw = back.NBT.Value["sections"].([]nbt.Compound)[0]["block_states"].(nbt.Compound)
			entries, ok := nbt.Compounds(raw["palette"])
			if !ok {
				t.Fatalf("mixed palette must use compound list: %T", raw["palette"])
			}
			wantEntry := nbt.Compound{"id": "minecraft:oak_log", "properties": nbt.Compound{"axis": "x"}}
			if !reflect.DeepEqual(entries[2], wantEntry) {
				t.Fatalf("edited entry uses wrong format: %#v", entries[2])
			}
			if test.name == "mixed" && entries[1]["custom"] != int16(7) {
				t.Fatal("unchanged modern entry metadata lost")
			}
		})
	}
}

func TestModernPaletteRejectsMalformedEntries(t *testing.T) {
	// 错误的 NBT 类型应在读取边界报错，不能被默认属性掩盖。
	for _, palette := range []any{
		[]string{},
		[]nbt.Compound{{"id": int32(1)}},
		[]nbt.Compound{{"": int32(1)}},
		[]nbt.Compound{{"id": "minecraft:grass_block", "properties": "snowy=false"}},
		[]nbt.Compound{{"id": "minecraft:grass_block", "properties": nbt.Compound{"snowy": int8(0)}}},
	} {
		chunk := NewChunk(celler.ChunkPos{}, 5023)
		chunk.NBT.Value["sections"] = []nbt.Compound{{
			"Y": int8(0), "block_states": nbt.Compound{"palette": palette},
		}}
		if _, err := chunk.Section(0); err == nil {
			t.Fatalf("accepted invalid palette: %#v", palette)
		}
	}
}
