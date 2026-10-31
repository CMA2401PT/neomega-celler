package layer_storage

import "testing"

func TestConvertPaletteOwnsIndexes(t *testing.T) {
	source := MakePaletteStorage(uint32(0))
	source.Set(At(1, 2, 3), 1)
	converted := ConvertPalette(source, func(value uint32) string {
		if value == 0 {
			return "air"
		}
		return "stone"
	})
	converted.Set(At(4, 5, 6), "stone")
	if source.Get(At(4, 5, 6)) != 0 {
		t.Fatal("converted layer mutated source indexes")
	}
	source.Set(At(1, 2, 3), 0)
	if converted.Get(At(1, 2, 3)) != "stone" {
		t.Fatal("source mutated converted indexes")
	}
}
