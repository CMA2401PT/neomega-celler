package layer_storage

import (
	"bytes"
	"celler"
	"fmt"
	"testing"
)

func TestPlainStorageAnyInterface(t *testing.T) {
	a0 := &Plain[uint32]{}
	func(celler.In) {}(a0)
	func(celler.Out) {}(a0)
	func(celler.Storage[uint32]) {}(a0)
}

func TestPlainStorageAnyData(t *testing.T) {
	var a0 Plain[uint32]
	a0.InitWithDefault(0)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				b := a0.Get(celler.ToSubChunkBlockPos(x, y, z))
				if b != 0 {
					t.Errorf("%v: %v!= 0", celler.ToSubChunkBlockPos(x, y, z), b)
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
				b := a0.Get(celler.ToSubChunkBlockPos(x, y, z))
				if b != 100 {
					t.Errorf("%v: %v!= 1", celler.ToSubChunkBlockPos(x, y, z), b)
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
				a0.Set(celler.ToSubChunkBlockPos(x, y, z), 200)
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				b := a0.Get(celler.ToSubChunkBlockPos(x, y, z))
				if b != 200 {
					t.Errorf("%v: %v!= 200", celler.ToSubChunkBlockPos(x, y, z), b)
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
				a0.Set(celler.ToSubChunkBlockPos(x, y, z), uint32(x*100+y*10+z))
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if a0.Get(celler.ToSubChunkBlockPos(x, y, z)) != uint32(x*100+y*10+z) {
					t.Errorf("expect %v @ %v, get %v", uint32(x*100+y*10*z), celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
				}
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					a0.Set(celler.ToSubChunkBlockPos(x, y, z), uint32(x*100+y*10+z))
				} else if z%3 == 1 {
					a0.Set(celler.ToSubChunkBlockPos(x, y, z), uint32(x*100+y*10-z))
				} else {
					a0.Set(celler.ToSubChunkBlockPos(x, y, z), uint32(x*100+y*10+2*z))
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
				if a0.Get(celler.ToSubChunkBlockPos(x, y, z)) != v {
					t.Errorf("expect %v @ %v, get %v", v, celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
				}
			}
		}
	}

	var aS Plain[string]
	aS.InitWithDefault("?")
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				aS.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if aS.Get(celler.ToSubChunkBlockPos(x, y, z)) != fmt.Sprintf("%v", uint32(x*100+y*10+z)) {
					t.Errorf("expect %v @ %v, get %v", uint32(x*100+y*10*z), celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
				}
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					aS.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
				} else if z%3 == 1 {
					aS.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10-z)))
				} else {
					aS.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+2*z)))
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
				if aS.Get(celler.ToSubChunkBlockPos(x, y, z)) != fmt.Sprintf("%v", v) {
					t.Errorf("expect %v @ %v, get %v", v, celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
				}
			}
		}
	}

}

func TestPlainStorageAnyDumpAndLoad(t *testing.T) {
	var aT Plain[uint32]
	aT.InitWithDefault(100)
	outBuf := bytes.NewBuffer(nil)
	aT.Dump(outBuf)
	testoutBytes := outBuf.Bytes()
	if len(testoutBytes) != 4096*4+4 {
		t.Errorf("expect len=4096*4+4, get %v", len(testoutBytes))
	}

	var a0 Plain[uint32]
	a0.InitWithDefault(0)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				a0.Set(celler.ToSubChunkBlockPos(x, y, z), uint32(x*100+y*10+z))
			}
		}
	}
	var a1 Plain[string]
	a1.InitWithDefault("?")
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				a1.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
			}
		}
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if z%3 == 0 {
					a1.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+z)))
				} else if z%3 == 1 {
					a1.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10-z)))
				} else {
					a1.Set(celler.ToSubChunkBlockPos(x, y, z), fmt.Sprintf("%v", uint32(x*100+y*10+2*z)))
				}
			}
		}
	}

	outBuf = bytes.NewBuffer(nil)
	a0.Dump(outBuf)
	a1.Dump(outBuf)

	inBuf := bytes.NewReader(outBuf.Bytes())
	var a0back Plain[uint32]
	a0back.Load(inBuf)
	var a1back Plain[string]
	a1back.Load(inBuf)

	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				if a0back.Get(celler.ToSubChunkBlockPos(x, y, z)) != uint32(x*100+y*10+z) {
					t.Errorf("expect %v @ %v, get %v", uint32(x*100+y*10*z), celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
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
				if a1back.Get(celler.ToSubChunkBlockPos(x, y, z)) != fmt.Sprintf("%v", v) {
					t.Errorf("expect %v @ %v, get %v", v, celler.ToSubChunkBlockPos(x, y, z), a0.Get(celler.ToSubChunkBlockPos(x, y, z)))
				}
			}
		}
	}
}
