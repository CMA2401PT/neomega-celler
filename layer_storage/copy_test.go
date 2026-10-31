package layer_storage

import (
	"testing"
)

func CopyStorage[D comparable, OutStorageT Storage[D], OutStorageM StorageMaker[D, OutStorageT]](src RandomRead[D]) OutStorageT {
	var maker OutStorageM
	out := maker.Make()
	out.CopyFrom(src)
	return out
}

func TestCopyFrom(t *testing.T) {
	pa := &Palette[uint32]{}
	pa.InitWithDefault(100)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					pa.Set(At(x, y, z), uint32(x*100+y*10+z))
				} else if z%3 == 1 {
					pa.Set(At(x, y, z), uint32(x*100+y*10-z))
				} else {
					pa.Set(At(x, y, z), uint32(x*100+y*10+2*z))
				}
			}
		}
	}

	check := func(final RandomRead[uint32]) {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				for z := 0; z < 16; z++ {
					v := uint32(0)
					if z%3 == 0 {
						v = uint32(x*100 + y*10 + z)
					} else if z%3 == 1 {
						v = uint32(x*100 + y*10 - z)
					} else {
						v = uint32(x*100 + y*10 + 2*z)
					}
					if final.Get(At(x, y, z)) != v {
						t.Errorf("expect %v @ %v, get %v", v, At(x, y, z), final.Get(At(x, y, z)))
					}
				}
			}
		}
	}

	papa := CopyStorage[uint32, *Palette[uint32], Palette[uint32]](pa)
	mappa := CopyStorage[uint32, *Map[uint32], Map[uint32]](papa)
	plainmap := CopyStorage[uint32, *Plain[uint32], Plain[uint32]](mappa)
	mapplain := CopyStorage[uint32, *Map[uint32], Map[uint32]](plainmap)
	mapmap := CopyStorage[uint32, *Map[uint32], Map[uint32]](mapplain)
	pamap := CopyStorage[uint32, *Palette[uint32], Palette[uint32]](mapmap)
	plainpa := CopyStorage[uint32, *Plain[uint32], Plain[uint32]](pamap)
	plainpalin := CopyStorage[uint32, *Plain[uint32], Plain[uint32]](plainpa)
	paplain := CopyStorage[uint32, *Palette[uint32], Palette[uint32]](plainpalin)
	check(paplain)
}

func TestCopyFromPlaceHolder(t *testing.T) {
	pa := &Palette[uint32]{}
	pa.InitWithDefault(100)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					pa.Set(At(x, y, z), uint32(x*100+y*10+z))
				} else if z%3 == 1 {
					pa.Set(At(x, y, z), uint32(x*100+y*10-z))
				} else {
					pa.Set(At(x, y, z), uint32(x*100+y*10+2*z))
				}
			}
		}
	}
	h := &PlaceHolder[uint32]{}
	h.CopyFrom(pa)
	pa2 := &Palette[uint32]{}
	pa2.CopyFrom(h)
	h2 := &PlaceHolder[uint32]{}
	h2.CopyFrom(pa2)
	if h2.Default() != 100 {
		t.Errorf("expect %v", h2.Default())
	}
}
