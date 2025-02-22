package mca

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

type RwRegion struct {
	*RoRegion
	bw      Writer
	changed bool
}

func (rw *RwRegion) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.bw.Close()
}

var ErrTooLarge = errors.New("no data")

func (rw *RwRegion) findSpace(need uint32) (start uint32) {
	for i := int32(0); i < int32(need); i++ {
		if rw.isOccupy(start + uint32(i)) {
			start += uint32(i + 1)
			i = -1
		}
	}
	return
}

func (rw *RwRegion) setChunkData(sp InRegionPos, data []byte) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.changed = true

	cb := bytes.NewBuffer(nil)
	cb.WriteByte(2) //zlib compress
	zw := zlib.NewWriter(cb)
	zw.Write(data)
	zw.Close()
	data = cb.Bytes()

	requiredSectors := uint32((len(data) + 4 + 4096 - 1) / 4096)
	beginSector, currentSectors := rw.decodeSpace(rw.chunkDataLocation[sp.Cz][sp.Cx])

	// maximum chunk size is 1MB
	if requiredSectors >= 256 {
		return ErrTooLarge
	}

	if beginSector != 0 && currentSectors == requiredSectors {
		// we can simply overwrite the old sectors
	} else {
		// we need to allocate new sectors

		// mark the sectors previously used for this chunk as free
		for i := uint32(0); i < currentSectors; i++ {
			rw.markOccupy(beginSector+i, false)
		}

		// scan for a free space large enough to store this chunk
		beginSector = rw.findSpace(requiredSectors)

		// mark the sectors previously used for this chunk as used
		currentSectors = requiredSectors
		for i := uint32(0); i < requiredSectors; i++ {
			rw.markOccupy(beginSector+i, true)
		}

		// update file head
		rw.setHead(sp.Cx, sp.Cz, uint32((beginSector<<8)|(requiredSectors&0xFF)), uint32(time.Now().Unix()))
	}

	sectorsData := make([]byte, 4096*requiredSectors)
	binary.BigEndian.PutUint32(sectorsData, uint32(len(data)))
	copy(sectorsData[4:len(data)+4], data)
	_, err := rw.bw.WriteAt(sectorsData, 4096*int64(beginSector))
	return err
}

func (rw *RwRegion) setHead(x, z uint8, offset, timestamp uint32) {
	var buf [4]byte
	rw.chunkDataLocation[z][x] = offset
	binary.BigEndian.PutUint32(buf[:], offset)
	rw.bw.WriteAt(buf[:], int64(4*(int(z)*32+int(x))))
	rw.Timestamps[x][z] = timestamp
	binary.BigEndian.PutUint32(buf[:], timestamp)
	rw.bw.WriteAt(buf[:], (int64(4*(int(z)+int(x)*32) + 4096)))
}

func newRwRegion(rw RW) (*RwRegion, error) {
	rr := &RwRegion{
		RoRegion: &RoRegion{
			br: rw,
		},
		bw: rw,
	}
	err := rr.Init()
	if err == nil {
		return rr, nil
	}
	if err == io.EOF {
		rw.WriteAt(make([]byte, 4096*2), 0)
	}
	return rr, rr.Init()
}
