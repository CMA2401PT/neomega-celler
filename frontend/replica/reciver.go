package replica

type reciver [4096][]uint16

func (rs *reciver) setvalue(x, y, z uint8, block uint16) {
	rx := x >> 4
	ry := y >> 4
	rz := z >> 4
	ri := (int(rx) << 8) | (int(rz) << 4) | int(ry)
	r := rs[ri]
	if r == nil {
		r = make([]uint16, 4096)
		rs[ri] = r
	}
	r[(int(x&0xf)<<8)|(int(z&0xf)<<4)|int(y&0xf)] = block
}

func (rs *reciver) Iter(cb func(P, func(func(P, uint16) bool)) bool) {
	for rx := uint8(0); rx < 16; rx++ {
		for rz := uint8(0); rz < 16; rz++ {
			for ry := uint8(0); ry < 16; ry++ {
				ri := (int(rx) << 8) | (int(rz) << 4) | int(ry)
				r := rs[ri]
				if r == nil {
					continue
				}
				iterFn := func(icb func(P, uint16) bool) {
					for x := uint8(0); x < 16; x++ {
						for z := uint8(0); z < 16; z++ {
							for y := uint8(0); y < 16; y++ {
								i := (int(x) << 8) | (int(z) << 4) | int(y)
								v := r[i]
								if v == 0 {
									continue
								}
								if !icb(P{X: x, Y: y, Z: z}, v) {
									return
								}
							}
						}
					}
				}
				if !cb(P{X: rx, Y: ry, Z: rz}, iterFn) {
					return
				}
			}
		}
	}
}
