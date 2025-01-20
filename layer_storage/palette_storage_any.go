package layer_storage

import (
	"bytes"
	"celler"
	"celler/storage_io"
	"fmt"
	"io"
	"unsafe"
)

type Palette[T storage_io.Number | ~string] struct {
	Data                  []uint32
	Palette               []T
	Level                 uint32
	IdxsPerUint32         uint32
	Mask                  uint32
	LastPaletteIdx        uint32
	LastVisitPaletteIdx   uint32
	LastVisitPaletteValue T
}

func (Palette[T]) Make() *Palette[T] {
	return &Palette[T]{}
}

func (s *Palette[T]) getPaletteIndexByBlockIdx(posIdx int) int {
	if s.Level == 0 {
		return int(s.LastVisitPaletteIdx)
	}
	index := posIdx / int(s.IdxsPerUint32)
	offset := (posIdx % int(s.IdxsPerUint32)) * int(s.Level)
	return int((s.Data[index] >> offset) & uint32(s.Mask))
}

func (s *Palette[T]) setPaletteIndexByBlockIdx(posIdx int, paletteIdx uint32) {
	index := posIdx / int(s.IdxsPerUint32)
	offset := (posIdx % int(s.IdxsPerUint32)) * int(s.Level)
	s.Data[index] &= ^(s.Mask << offset)
	s.Data[index] |= (paletteIdx & s.Mask) << offset
}

func (s *Palette[T]) findBlockPaletteIdx(b T) (paletteIdx int, found bool) {
	if s.LastVisitPaletteValue == b {
		return int(s.LastVisitPaletteIdx), true
	}

	for paletteIdx = 0; paletteIdx <= int(s.LastPaletteIdx); paletteIdx++ {
		if s.Palette[paletteIdx] == b {
			s.LastVisitPaletteIdx = uint32(paletteIdx)
			s.LastVisitPaletteValue = b
			return paletteIdx, true
		}
	}
	return 0, false
}

func (s *Palette[T]) appendPalette(b T) {
	idx := s.LastPaletteIdx + 1
	s.LastPaletteIdx = idx
	s.LastVisitPaletteIdx = idx
	s.LastVisitPaletteValue = b
	s.Palette[idx] = b
}

func (s *Palette[T]) posToIdx(pos celler.SubChunkBlockPos) int {
	return (int(pos.X&0xf) << 8) | (int(pos.Z&0xf) << 4) | int(pos.Y&0xf)
}

func (s *Palette[T]) idxToPos(i int) celler.SubChunkBlockPos {
	return celler.SubChunkBlockPos{X: uint8(i >> 8 & 0xf), Y: uint8(i & 0xf), Z: uint8(i >> 4 & 0xf)}
}

func (s *Palette[T]) Get(pos celler.SubChunkBlockPos) T {
	return s.Palette[s.getPaletteIndexByBlockIdx(s.posToIdx(pos))]
}

func (s *Palette[T]) IterNonDefault(fn func(celler.SubChunkBlockPos, T) (next bool)) {
	if s.Level == 0 {
		return
	}
	for i := 0; i < 4096; i++ {
		pi := s.getPaletteIndexByBlockIdx(i)
		if pi == 0 {
			continue
		}
		if !fn(s.idxToPos(i), s.Palette[pi]) {
			return
		}
	}
}

func (s *Palette[T]) Default() T {
	return s.Palette[0]
}

func (s *Palette[T]) InitWithDefault(data T) {
	s.fill(0, data)
}

func (s *Palette[T]) fill(level int, fill T) {
	var wordByteSize = 4
	var wordBitSize = wordByteSize * 8
	var storageSize = 4096
	var paletteSize = 1 << level
	var idxsPerUint32 = 0
	var arraySize = 0
	var mask = ((1 << level) - 1)
	if level != 0 {
		idxsPerUint32 = wordBitSize / level
		arraySize = (storageSize + idxsPerUint32 - 1) / idxsPerUint32
	}
	palette := make([]T, paletteSize)
	palette[0] = fill
	s.Data = make([]uint32, arraySize)
	s.Palette = palette
	s.Level = uint32(level)
	s.IdxsPerUint32 = uint32(idxsPerUint32)
	s.Mask = uint32(mask)
	s.LastVisitPaletteIdx = 0
	s.LastVisitPaletteValue = fill
	s.LastPaletteIdx = 0
}

func (s *Palette[T]) Set(pos celler.SubChunkBlockPos, b T) {
	if s.Level == 0 && s.Palette[0] == b {
		return
	}
	idx, found := s.findBlockPaletteIdx(b)
	if !found {
		idx = int(s.LastPaletteIdx) + 1
		if idx == (1 << s.Level) {
			newBS := (&Palette[T]{})
			newBS.fill(int(s.Level)+1, s.Palette[0])
			copy(newBS.Palette, s.Palette)
			for i := 0; i < 4096; i++ {
				newBS.setPaletteIndexByBlockIdx(i, uint32(s.getPaletteIndexByBlockIdx(i)))
			}
			s.Data = newBS.Data
			s.Palette = newBS.Palette
			s.Level = newBS.Level
			s.IdxsPerUint32 = newBS.IdxsPerUint32
			s.Mask = newBS.Mask
		}
		s.appendPalette(b)
	}
	s.setPaletteIndexByBlockIdx(s.posToIdx(pos), uint32(idx))
}

func (s *Palette[T]) Same() (d T, allSame bool) {
	if s.Level == 0 {
		return s.LastVisitPaletteValue, true
	}
	firstPaletteIdx := s.getPaletteIndexByBlockIdx(0)
	for i := 1; i < 4096; i++ {
		if firstPaletteIdx != s.getPaletteIndexByBlockIdx(i) {
			return d, false
		}
	}
	return s.Palette[firstPaletteIdx], true
}

func (s *Palette[T]) CopyFrom(src celler.LayerStorageRead[T]) {
	if placeHolderSrc, ok := src.(*PlaceHolder[T]); ok {
		s.InitWithDefault(placeHolderSrc.Default())
		return
	}
	if paletteSrc, ok := src.(*Palette[T]); ok {
		s.Data = make([]uint32, len(paletteSrc.Data))
		copy(s.Data, paletteSrc.Data)
		s.Palette = make([]T, len(paletteSrc.Palette))
		copy(s.Palette, paletteSrc.Palette)
		s.Level = paletteSrc.Level
		s.IdxsPerUint32 = paletteSrc.IdxsPerUint32
		s.Mask = paletteSrc.Mask
		s.LastPaletteIdx = paletteSrc.LastPaletteIdx
		s.LastVisitPaletteIdx = paletteSrc.LastVisitPaletteIdx
		s.LastVisitPaletteValue = paletteSrc.LastVisitPaletteValue
		return
	}
	if mapSrc, ok := src.(*Map[T]); ok {
		defaultV := mapSrc.Default()
		s.InitWithDefault(defaultV)
		for p, v := range mapSrc.IterNonDefault {
			s.Set(p, v)
		}
		return
	}
	if plainSrc, ok := src.(*Plain[T]); ok {
		defaultV := src.Default()
		s.InitWithDefault(defaultV)
		for p, v := range plainSrc.IterNonDefault {
			s.Set(p, v)
		}
		return
	}
	{
		fmt.Printf("slow copy from: %T\n", src)
		defaultV := src.Default()
		s.InitWithDefault(defaultV)
		for p, v := range src.IterNonDefault {
			s.Set(p, v)
		}
	}
}

func MakePaletteStorage[T storage_io.Number | ~string](defaultV T) *Palette[T] {
	storage := &Palette[T]{}
	storage.InitWithDefault(defaultV)
	return storage
}

func (s *Palette[T]) Dump(w io.Writer) {
	head := []uint32{s.Level, s.IdxsPerUint32, s.Mask, s.LastPaletteIdx, s.LastVisitPaletteIdx}
	pb := bytes.NewBuffer(nil)
	storage_io.WriteArrayWithoutCount(pb, append([]T{s.LastVisitPaletteValue}, s.Palette...))
	pBytes := pb.Bytes()
	if len(pBytes)%4 != 0 {
		pBytes = append(pBytes, make([]byte, 4-len(pBytes)%4)...)
	}
	storage_io.WriteValue(w, uint32(5+len(pBytes)/4+len(s.Data)))
	w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&head[0])), 20))
	w.Write(pBytes)
	if len(s.Data) > 0 {
		p := (unsafe.Pointer(&s.Data[0]))
		w.Write(unsafe.Slice((*byte)(p), len(s.Data)*4))
	}
}

func (s *Palette[B]) Load(r io.Reader) {
	dataLen := storage_io.ReadValue[uint32](r)
	reciver := make([]byte, dataLen*4)
	r.Read(reciver)
	uint32Arr := unsafe.Slice((*uint32)(unsafe.Pointer(&reciver[0])), dataLen)
	s.Level = uint32Arr[0]
	s.IdxsPerUint32 = uint32Arr[1]
	s.Mask = uint32Arr[2]
	s.LastPaletteIdx = uint32Arr[3]
	s.LastVisitPaletteIdx = uint32Arr[4]
	pb := bytes.NewReader(reciver[20:])
	lastAndpalette := storage_io.ReadArrayWithoutCount[B](pb, int(1<<s.Level)+1)
	s.LastVisitPaletteValue = lastAndpalette[0]
	s.Palette = lastAndpalette[1:]
	var arraySize = 0
	if s.Level != 0 {
		arraySize = int(4096+s.IdxsPerUint32-1) / int(s.IdxsPerUint32)
		dataBytes := reciver[len(reciver)-4*arraySize:]
		s.Data = unsafe.Slice((*uint32)(unsafe.Pointer(&dataBytes[0])), arraySize)
	} else {
		s.Data = make([]uint32, 0)
	}

}
