package mca

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"unsafe"
)

type RoRegion struct {
	br                Reader
	occupyTable       []uint8
	chunkDataLocation [32][32]uint32 // 32*32*4 = 4096 (Data Sector 1)  lower 8bit(1 byte) (0~255 data sectors) upper 24bit(sector begin index)
	Timestamps        [32][32]uint32 // 32*32*4 = 4096 (Data Sector 1)
	mu                sync.RWMutex
}

func (rr *RoRegion) markOccupy(sector uint32, occupy bool) {
	p := sector / 8
	off := sector % 8
	if int(p) >= len(rr.occupyTable) {
		rr.occupyTable = append(rr.occupyTable, make([]uint8, int(p)-len(rr.occupyTable)+1)...)
	}
	v := rr.occupyTable[p]
	if occupy {
		v |= (1 << off)
	} else {
		v &= (^(1 << off))
	}
	rr.occupyTable[p] = v
}

func (rr *RoRegion) isOccupy(sector uint32) bool {
	p := sector / 8
	off := sector % 8
	if int(p) >= len(rr.occupyTable) {
		return false
	}
	return (rr.occupyTable[p] & (1 << off)) != 0
}

func (rr *RoRegion) Init() error {
	rr.mu = sync.RWMutex{}
	tmp := make([]uint8, 4096)
	_, err := rr.br.ReadAt(tmp, 0)
	if err != nil {
		return err
	}

	err = binary.Read(bytes.NewBuffer(tmp), binary.BigEndian, &rr.chunkDataLocation)
	if err != nil {
		return err
	}
	_, err = rr.br.ReadAt(tmp, 4096)
	if err != nil {
		return err
	}
	// read the timestamps
	err = binary.Read(bytes.NewBuffer(tmp), binary.BigEndian, &rr.Timestamps)
	if err != nil {
		return err
	}
	rr.markOccupy(0, true)
	rr.markOccupy(1, true)
	for _, chunks := range rr.chunkDataLocation {
		for _, chunk := range chunks {
			if startSector, occupiedSector := rr.decodeSpace(chunk); startSector != 0 {
				for i := uint32(0); i < occupiedSector; i++ {
					rr.markOccupy(uint32(startSector)+i, true)
				}
			}
		}
	}
	return nil
}

func (rr *RoRegion) decodeSpace(space uint32) (sec, num uint32) {
	return (space >> 8) & 0xFFFFFF, space & 0xFF
}

func (rr *RoRegion) Close() error {
	if rr == nil {
		return nil
	}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.br.Close()
}

var ErrNoData = errors.New("no data")

func (rr *RoRegion) getChunkData(sp InRegionPos) ([]byte, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	sec, _ := rr.decodeSpace(rr.chunkDataLocation[sp.Cz][sp.Cx])
	if sec == 0 {
		return nil, ErrNoData
	}

	begin := 4096 * int64(sec)
	var length uint32
	reciver := unsafe.Slice((*byte)(unsafe.Pointer(&length)), 4)
	_, err := rr.br.ReadAt(reciver, begin)
	if err != nil {
		return nil, err
	}
	length = binary.BigEndian.Uint32(reciver)
	data := make([]byte, length)
	_, err = rr.br.ReadAt(data, begin+4)
	if err != nil {
		return nil, err
	}
	ct := data[0]
	if ct == 3 {
		return data[1:], err
	}
	var cr io.ReadCloser
	switch ct {
	default:
		err = errors.New("unknown compression")
		return nil, err
	case 1:
		cr, err = gzip.NewReader(bytes.NewBuffer(data[1:]))
	case 2:
		cr, err = zlib.NewReader(bytes.NewBuffer(data[1:]))
	}
	if err != nil {
		return nil, err
	}
	defer cr.Close()
	data, err = io.ReadAll(cr)
	return data, err
}

func newRoRegion(br Reader) (*RoRegion, error) {
	rr := &RoRegion{
		br: br,
	}
	return rr, rr.Init()
}
