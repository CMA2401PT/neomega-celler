package layer_storage

import (
	"fmt"
	"testing"
)

func TestPaletteStorageAnyInterface(t *testing.T) {
	a0 := &Palette[uint32]{}
	func(Storage[uint32]) {}(a0)
}

func TestPaletteStorageAnyData(t *testing.T) {
	var a0 Palette[uint32]
	a0.InitWithDefault(0)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				b := a0.Get(At(x, y, z))
				if b != 0 {
					t.Errorf("%v: %v!= 0", At(x, y, z), b)
				}
			}
		}
	}
	allBlock, allSame := a0.Same()
	if !allSame || allBlock != 0 {
		t.Errorf("not all same")
	}

	a0.InitWithDefault(100)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				b := a0.Get(At(x, y, z))
				if b != 100 {
					t.Errorf("%v: %v!= 1", At(x, y, z), b)
				}
			}
		}
	}
	allBlock, allSame = a0.Same()
	if !allSame || allBlock != 100 {
		t.Errorf("not all same")
	}

	a0.InitWithDefault(0)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				a0.Set(At(x, y, z), 200)
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				b := a0.Get(At(x, y, z))
				if b != 200 {
					t.Errorf("%v: %v!= 200", At(x, y, z), b)
				}
			}
		}
	}
	allBlock, allSame = a0.Same()
	if !allSame || allBlock != 200 {
		t.Errorf("not all same")
	}

	a0.InitWithDefault(100)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				a0.Set(At(x, y, z), uint32(x*100+y*10+z))
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if a0.Get(At(x, y, z)) != uint32(x*100+y*10+z) {
					t.Errorf("expect %v @ %v, get %v", uint32(x*100+y*10*z), At(x, y, z), a0.Get(At(x, y, z)))
				}
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					a0.Set(At(x, y, z), uint32(x*100+y*10+z))
				} else if z%3 == 1 {
					a0.Set(At(x, y, z), uint32(x*100+y*10-z))
				} else {
					a0.Set(At(x, y, z), uint32(x*100+y*10+2*z))
				}
			}
		}
	}
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
				if a0.Get(At(x, y, z)) != v {
					t.Errorf("expect %v @ %v, get %v", v, At(x, y, z), a0.Get(At(x, y, z)))
				}
			}
		}
	}

	var aS Palette[string]
	aS.InitWithDefault("?")
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				aS.Set(At(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if aS.Get(At(x, y, z)) != fmt.Sprintf("%v", uint32(x*100+y*10+z)) {
					t.Errorf("expect %v @ %v, get %v", uint32(x*100+y*10*z), At(x, y, z), a0.Get(At(x, y, z)))
				}
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					aS.Set(At(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
				} else if z%3 == 1 {
					aS.Set(At(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10-z)))
				} else {
					aS.Set(At(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+2*z)))
				}
			}
		}
	}
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
				if aS.Get(At(x, y, z)) != fmt.Sprintf("%v", v) {
					t.Errorf("expect %v @ %v, get %v", v, At(x, y, z), a0.Get(At(x, y, z)))
				}
			}
		}
	}

}
