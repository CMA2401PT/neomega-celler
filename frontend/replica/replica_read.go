package replica

import (
	"bytes"
	"unsafe"

	"github.com/pierrec/lz4/v4"
)

type ReplicaReader struct {
	Cx     uint8
	Cy     uint8
	Cz     uint8
	cacheP uint32
	cache  []uint32
	*lz4.Reader
}

func (r *ReplicaReader) feed() {
	c, _ := r.Reader.Read(unsafe.Slice((*byte)(unsafe.Pointer(&r.cache[0])), CacheSize*4))
	r.cacheP = 0
	r.cache = r.cache[:uint32(c)/4]
}

func (r *ReplicaReader) fetchCmd() (cmd uint32) {
	if r.cacheP == uint32(len(r.cache)) {
		// feed more
		r.feed()
		if len(r.cache) == 0 {
			return 0
		}
	}
	cmd = r.cache[r.cacheP]
	r.cacheP++
	return cmd
}

type P struct {
	X, Y, Z uint8
}

func (r *ReplicaReader) IterCmds(cb func(P, uint16) (next bool)) {
	cmd := uint32(0)
	for {
		cmd = r.fetchCmd()
		if cmd == 0 {
			return
		}
		if cmd&uint32(0x80000000) == 0 {
			// move cmd
			r.Cx = uint8(cmd >> 16)
			r.Cy = uint8(cmd >> 8)
			r.Cz = uint8(cmd)
			continue
		}
		// short cmd
		repeat := uint8(cmd)
		blk := uint16(cmd >> 8)
		for repeat != 0 {
			if cmd&uint32(0x40000000) != 0 {
				r.Cx++
			} else if cmd&uint32(0x20000000) != 0 {
				r.Cx--
			}
			if cmd&uint32(0x10000000) != 0 {
				r.Cy++
			} else if cmd&uint32(0x08000000) != 0 {
				r.Cy--
			}
			if cmd&uint32(0x04000000) != 0 {
				r.Cz++
			} else if cmd&uint32(0x02000000) != 0 {
				r.Cz--
			}
			cb(P{X: r.Cx, Y: r.Cy, Z: r.Cz}, uint16(blk))
			repeat--
		}
	}
}

func (r *ReplicaReader) Dump() *reciver {
	reciver := &reciver{}
	for p, blk := range r.IterCmds {
		reciver.setvalue(p.X, p.Y, p.Z, blk)
	}
	return reciver
}

func NewReplicaReader(data []byte) *ReplicaReader {
	br := bytes.NewReader(data)
	cr := lz4.NewReader(br)
	r := &ReplicaReader{
		cache:  make([]uint32, CacheSize),
		cacheP: 0,
		Reader: cr,
	}
	r.feed()
	return r
}
