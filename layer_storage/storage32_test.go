package layer_storage

// import (
// 	"bytes"
// 	"testing"
// )

// func TestSubchunkData(t *testing.T) {
// 	var a0 Storage32[uint32]
// 	a0.Fill(0)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				b := a0.Get(x, y, z)
// 				if b != 0 {
// 					t.Errorf("%v %v %v: %v!= 0", x, y, z, b)
// 				}
// 			}
// 		}
// 	}
// 	allBlock, allSame := a0.Same()
// 	if !allSame || allBlock != 0 {
// 		t.Errorf("not all same")
// 	}

// 	a0.Fill(100)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				b := a0.Get(x, y, z)
// 				if b != 100 {
// 					t.Errorf("%v %v %v: %v!= 1", x, y, z, b)
// 				}
// 			}
// 		}
// 	}
// 	allBlock, allSame = a0.Same()
// 	if !allSame || allBlock != 100 {
// 		t.Errorf("not all same")
// 	}

// 	a0.Fill(0)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				a0.Set(x, y, z, 200)
// 			}
// 		}
// 	}
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				b := a0.Get(x, y, z)
// 				if b != 200 {
// 					t.Errorf("%v %v %v: %v!= 200", x, y, z, b)
// 				}
// 			}
// 		}
// 	}
// 	allBlock, allSame = a0.Same()
// 	if !allSame || allBlock != 200 {
// 		t.Errorf("not all same")
// 	}

// 	a0.Fill(100)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				a0.Set(x, y, z, uint32(x*100+y*10+z))
// 			}
// 		}
// 	}
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				if a0.Get(x, y, z) != uint32(x*100+y*10+z) {
// 					t.Errorf("expect %v @ %v %v %v, get %v", uint32(x*100+y*10*z), x, y, z, a0.Get(x, y, z))
// 				}
// 			}
// 		}
// 	}
// }

// func TestDumpAndLoad(t *testing.T) {
// 	var a0 Storage32[uint32]
// 	a0.Fill(0)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				a0.Set(x, y, z, uint32(x*100+y*10+z))
// 			}
// 		}
// 	}
// 	var a1 Storage32[uint32]
// 	a1.Fill(0)
// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				a1.Set(x, y, z, uint32(x*100+y*10+z))
// 			}
// 		}
// 	}

// 	outBuf := bytes.NewBuffer(nil)
// 	a0.Dump(outBuf)
// 	a1.Dump(outBuf)

// 	inBuf := bytes.NewReader(outBuf.Bytes())
// 	var a0back Storage32[uint32]
// 	a0back.Load(inBuf)
// 	var a1back Storage32[uint32]
// 	a1back.Load(inBuf)

// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				if a0back.Get(x, y, z) != uint32(x*100+y*10+z) {
// 					t.Errorf("expect %v @ %v %v %v, get %v", uint32(x*100+y*10*z), x, y, z, a0.Get(x, y, z))
// 				}
// 			}
// 		}
// 	}

// 	for x := 0; x < 16; x++ {
// 		for y := 0; y < 16; y++ {
// 			for z := 0; z < 16; z++ {
// 				if a1back.Get(x, y, z) != uint32(x*100+y*10+z) {
// 					t.Errorf("expect %v @ %v %v %v, get %v", uint32(x*100+y*10*z), x, y, z, a0.Get(x, y, z))
// 				}
// 			}
// 		}
// 	}
// }
