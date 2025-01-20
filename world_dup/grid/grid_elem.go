package grid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

var (
	ErrNoSector             = errors.New("sector does not exist")
	ErrNoData               = errors.New("data is missing")
	ErrSectorNegativeLength = errors.New("declared length of data is negative")
	ErrTooLarge             = errors.New("data too large")
)

// Region contain 32*32 chunks in one .mca file
// Not MT-Safe!
type Region struct {
	data              []byte
	chunkDataLocation [32][32]int32 // 32*32*4 = 4096 (Data Sector 1)  lower 8bit(1 byte) (0~255 data sectors) upper 24bit(sector begin index)
	Timestamps        [32][32]int32 // 32*32*4 = 4096 (Data Sector 1)

	// DataSectorUsed4096 record if a sector is in used.
	// contrary to mojang's, because false is the default value in Go.
	DataSectorUsed4096 map[int32]bool
	mu                 sync.RWMutex
	Changed            bool
}

// whichChunkInsideRegion calculate chunk's coordinates relative to region
// 计算chunk在region中的相对坐标。即，除以32并取余。
func whichChunkInsideRegion(chunkX, chunkZ int) (int, int) {
	// c & (32-1)
	// is equal to:
	// (c %= 32) > 0 ? c : -c; //C language
	return chunkX & 31, chunkZ & 31
}

// chunkInWhichRegion calculate the region's coordinates where the chunk in
// 计算chunk在哪一个region中
func chunkInWhichRegion(chunkX, chunkZ int) (int, int) {
	return chunkX >> 5, chunkZ >> 5
}

// LoadRegion works like Open but read from an io.ReadWriteSeeker.
func LoadRegion(currentData []byte) (r *Region, err error) {
	r = &Region{
		data:               currentData,
		DataSectorUsed4096: make(map[int32]bool),
		mu:                 sync.RWMutex{},
	}

	// read the offsets
	err = binary.Read(bytes.NewBuffer(currentData), binary.BigEndian, &r.chunkDataLocation)
	if err != nil {
		return nil, err
	}
	r.DataSectorUsed4096[0] = true

	// read the timestamps
	err = binary.Read(bytes.NewBuffer(currentData[4096:]), binary.BigEndian, &r.Timestamps)
	if err != nil {
		return nil, err
	}
	r.DataSectorUsed4096[1] = true

	// generate sectorFree table
	for _, chunks := range r.chunkDataLocation {
		for _, chunk := range chunks {
			if startSector, occupiedSector := sectorStartAndNumOccupied(chunk); startSector != 0 {
				for i := int32(0); i < occupiedSector; i++ {
					r.DataSectorUsed4096[startSector+i] = true
				}
			}
		}
	}

	return r, nil
}

// // Create open .mca file with os.O_CREATE|os. O_EXCL, and init the region
// func Create(name string) (*Region, error) {
// 	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0o666)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return CreateWriter(f)
// }

// CreateWriter create Region by an io.ReadWriteSeeker
func NewRegion() (r *Region, err error) {
	r = &Region{}
	r.mu = sync.RWMutex{}
	r.DataSectorUsed4096 = make(map[int32]bool)
	buf := bytes.NewBuffer([]byte{})

	// write the offsets
	err = binary.Write(buf, binary.BigEndian, &r.chunkDataLocation)
	if err != nil {
		return nil, err
	}
	r.DataSectorUsed4096[0] = true

	// write the timestamps
	err = binary.Write(buf, binary.BigEndian, &r.Timestamps)
	if err != nil {
		return nil, err
	}
	r.DataSectorUsed4096[1] = true
	r.data = buf.Bytes()
	r.Changed = true
	return r, nil
}

func sectorStartAndNumOccupied(offset int32) (sec, num int32) {
	return (offset >> 8) & 0xFFFFFF, offset & 0xFF
}

// ReadSector find and read the Chunk data from region
func (r *Region) ReadSector(x, z int) (data []byte, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sec, num := sectorStartAndNumOccupied(r.chunkDataLocation[z][x])
	if sec == 0 {
		return nil, ErrNoSector
	}
	if len(r.data) < 4096*int(sec+num-1) {
		return nil, ErrNoData
	}
	reader := bytes.NewReader(r.data[4096*int(sec):])

	var length int32
	err = binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if length == 0 {
		return nil, ErrNoData
	}
	if length < 0 {
		return nil, ErrSectorNegativeLength
	}
	if length > 4096*num {
		return nil, ErrTooLarge
	}
	data = make([]byte, length)
	_, err = io.ReadFull(reader, data)

	return
}

// WriteSector write Chunk data into region file
func (r *Region) WriteSector(x, z int, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Changed = true
	requiredSectors := int32((len(data) + 4 + 4096 - 1) / 4096)
	beginSector, currentSectors := sectorStartAndNumOccupied(r.chunkDataLocation[z][x])

	// maximum chunk size is 1MB
	if requiredSectors >= 256 {
		return ErrTooLarge
	}

	if beginSector != 0 && currentSectors == requiredSectors {
		// we can simply overwrite the old sectors
	} else {
		// we need to allocate new sectors

		// mark the sectors previously used for this chunk as free
		for i := int32(0); i < currentSectors; i++ {
			r.DataSectorUsed4096[beginSector+i] = false
		}

		// scan for a free space large enough to store this chunk
		beginSector = r.findSpace(requiredSectors)

		// mark the sectors previously used for this chunk as used
		currentSectors = requiredSectors
		for i := int32(0); i < requiredSectors; i++ {
			r.DataSectorUsed4096[beginSector+i] = true
		}

		r.chunkDataLocation[z][x] = (beginSector << 8) | (requiredSectors & 0xFF)

		// update file head
		r.setHead(x, z, uint32(r.chunkDataLocation[z][x]), uint32(time.Now().Unix()))
		r.Timestamps[x][z] = int32(time.Now().Unix())
	}

	sectorsData := make([]byte, 4096*requiredSectors)
	binary.BigEndian.PutUint32(sectorsData, uint32(len(data)))
	copy(sectorsData[4:len(data)-4], data)
	r.writeOrAppend(sectorsData, 4096*int(beginSector))
	return nil
}

// ExistSector return if a sector is existed
func (r *Region) ExistSector(x, z int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.chunkDataLocation[z][x] != 0
}

func (r *Region) findSpace(need int32) (n int32) {
	for i := int32(0); i < need; i++ {
		if r.DataSectorUsed4096[n+i] {
			n += i + 1
			i = -1
		}
	}
	return
}

func (r *Region) setHead(x, z int, offset, timestamp uint32) {
	var buf [4]byte

	binary.BigEndian.PutUint32(buf[:], offset)
	r.writeOrAppend(buf[:], 4*(int(z)*32+int(x)))
	binary.BigEndian.PutUint32(buf[:], timestamp)
	r.writeOrAppend(buf[:], 4096+4*((z)*32+(x)))
}

func (r *Region) writeOrAppend(data []byte, offset int) {
	if offset+len(data) > len(r.data) {
		r.data = append(r.data, make([]byte, offset+len(data)-len(r.data))...)
	}
	copy(r.data[offset:offset+len(data)], data)
}
