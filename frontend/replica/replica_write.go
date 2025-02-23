package replica

import (
	"bytes"
	"unsafe"

	"github.com/pierrec/lz4/v4"
)

type ReplicaWriter struct {
	Cx     uint8
	Cy     uint8
	Cz     uint8
	Cmd    uint32
	cacheP uint32
	cache  []uint32
	*bytes.Buffer
	*lz4.Writer
}

// command: 4 byte
// set block: at current pos
// 10000000 [2byte block id] [1byte repaeat time]
// set block with 1 move
// 1[10 x+, 01 x-][10 y+, 01 y-][10 z+, 01 z-][reserve][2byte block id] [1byte repaeat time]
// move
// 00000000 [1byte x][1byte y][1byte z]

func (w *ReplicaWriter) SetCmd(x, y, z uint8, blkId uint16) {
	defer func() {
		w.Cx = x
		w.Cy = y
		w.Cz = z
	}()
	cmd := w.tryShortSetBlock(x, y, z, blkId)
	if cmd != 0 { //short cmd success
		if cmd&0xFFFFFF00 == w.Cmd&0xFFFFFF00 { //fast repeat
			repeat := uint8(w.Cmd)
			repeat += 1
			w.Cmd = (w.Cmd & 0xFFFFFF00) | uint32(repeat)
			if repeat == 255 { //max repeat
				w.pushCmd()
			}
			return
		} else {
			w.pushCmd()
			w.Cmd = cmd | 1
			return
		}
	}
	// short fail, encode to move + move-set
	w.pushCmd()
	w.Cmd = uint32(x)<<16 | uint32(y)<<8 | uint32(z)

	w.pushCmd()
	w.Cmd = uint32(0x80000001) | uint32(blkId)<<8
}

func (w *ReplicaWriter) Close() []byte {
	w.pushCmd()
	w.Writer.Write(unsafe.Slice((*byte)(unsafe.Pointer(&w.cache[0])), w.cacheP*4))
	w.Writer.Close()
	return w.Buffer.Bytes()
}

const CacheSize = 128

func (w *ReplicaWriter) pushCmd() {
	if w.Cmd == 0 {
		return
	}
	w.cache[w.cacheP] = w.Cmd
	w.Cmd = 0
	w.cacheP += 1
	if w.cacheP == CacheSize {
		w.Writer.Write(unsafe.Slice((*byte)(unsafe.Pointer(&w.cache[0])), CacheSize*4))
		w.cacheP = 0
	}
}

func (w *ReplicaWriter) tryShortSetBlock(x, y, z uint8, blkId uint16) uint32 {
	cmd := uint32(0x80000000)
	if x == w.Cx {
	} else if x == w.Cx+1 {
		cmd |= uint32(0x40000000)
	} else if x == w.Cx-1 {
		cmd |= uint32(0x20000000)
	} else {
		return 0
	}
	if y == w.Cy {
	} else if y == w.Cy+1 {
		cmd |= uint32(0x10000000)
	} else if y == w.Cy-1 {
		cmd |= uint32(0x08000000)
	} else {
		return 0
	}
	if z == w.Cz {
	} else if z == w.Cz+1 {
		cmd |= uint32(0x04000000)
	} else if z == w.Cz-1 {
		cmd |= uint32(0x02000000)
	} else {
		return 0
	}
	w.Cx = x
	w.Cy = y
	w.Cz = z
	cmd |= uint32(blkId) << 8
	return cmd
}

func NewReplicaWriter() *ReplicaWriter {
	bw := bytes.NewBuffer(nil)
	cw := lz4.NewWriter(bw)
	return &ReplicaWriter{
		Buffer: bw,
		Writer: cw,
		cache:  make([]uint32, CacheSize),
		cacheP: 0,
	}
}
