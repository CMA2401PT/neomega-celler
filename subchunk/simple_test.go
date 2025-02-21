package subchunk

import (
	"bytes"
	"celler"
	"celler/layer_storage"
	"testing"
)

func TestSimpleSubChunk(t *testing.T) {
	o := MakeVersionedSubChunk(
		1, nil,
		layer_storage.MakePaletteStorage(uint32(100)),
		layer_storage.MakePaletteStorage(uint32(0)),
		layer_storage.MakeMapStorage(celler.SNBT("")),
	)

	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				o.ChangeBlock(celler.BlockChange[uint32, uint32]{}.
					SetPos(celler.ToSubChunkBlockPos(x, y, z)).
					SetID(200),
				)
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
				o.ChangeBlock(celler.BlockChange[uint32, uint32]{}.
					SetPos(celler.ToSubChunkBlockPos(x, y, z)).
					SetID(v),
				)
			}
		}
	}

	check := func(a0 *layer_storage.Palette[uint32]) {
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
	}

	c := MakeVersionedSubChunk(
		0, nil,
		layer_storage.MakePaletteStorage(uint32(100)),
		layer_storage.MakePaletteStorage(uint32(0)),
		layer_storage.MakeMapStorage(celler.SNBT("")),
	)

	buf := bytes.NewBuffer(nil)
	o.Dump(buf)
	c.Load(bytes.NewReader(buf.Bytes()))

	if c.Version() != 8193 {
		t.Errorf("expect 8193, get%v", c.Version())
	}

	check(c.BlockIDStorage)

	dst := MakeVersionedSubChunk(
		1, nil,
		layer_storage.MakePaletteStorage(uint32(100)),
		layer_storage.MakePaletteStorage(uint32(0)),
		layer_storage.MakeMapStorage(celler.SNBT("")),
	)

	changes := c.QueryChange(dst.Version())
	dst.ApplyChange(changes)

	check(dst.BlockIDStorage)

	changes = c.QueryChange(dst.Version())
	dst.ApplyChange(changes)

	check(dst.BlockIDStorage)

	startCount := 10
	endCount := 100
	count := 0
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				v := uint32(0)
				if z%3 == 0 {
					v = uint32(x*200 + y*20 + z*2)
				} else if z%3 == 1 {
					v = uint32(x*200 + y*20 - z*2)
				} else {
					v = uint32(x*200 + y*20 + 4*z)
				}
				if count > startCount && count < endCount {
					c.ChangeBlock(celler.BlockChange[uint32, uint32]{}.
						SetPos(celler.ToSubChunkBlockPos(x, y, z)).
						SetID(v),
					)
				}
				count += 1
			}
		}
	}

	changes = c.QueryChange(dst.Version())
	dst.ApplyChange(changes)

	count = 0
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				v := uint32(0)
				if z%3 == 0 {
					v = uint32(x*200 + y*20 + z*2)
				} else if z%3 == 1 {
					v = uint32(x*200 + y*20 - z*2)
				} else {
					v = uint32(x*200 + y*20 + 4*z)
				}
				if count > startCount && count < endCount {
					if dst.BlockIDStorage.Get(celler.ToSubChunkBlockPos(x, y, z)) != v {
						t.Errorf("expect %v @ %v, get %v", v, celler.ToSubChunkBlockPos(x, y, z), dst.BlockIDStorage.Get(celler.ToSubChunkBlockPos(x, y, z)))
					}
				}
				count += 1
			}
		}
	}
}
