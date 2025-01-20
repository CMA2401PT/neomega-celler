package layer_storage

// import (
// 	"io"
// 	"unsafe"
// )

// const (
// 	_HEAD_SIZE     = 6
// 	_LEVEL         = 0
// 	_BLKS_PER_WORD = 1
// 	_MASK          = 2
// 	_PALE_LAST_IDX = 3
// 	_LI            = 4
// 	_LV            = 5
// 	_PALE_START    = _HEAD_SIZE
// )

// type Storage32[B ~uint32] struct {
// 	data []uint32
// }

// func makeStorage32ByLevel[B ~uint32](level int, fill uint32) []uint32 {
// 	var wordByteSize = 4
// 	var wordBitSize = wordByteSize * 8
// 	var storageSize = 4096

// 	var paletteSize = 1 << level
// 	var blocksPerWord = 0
// 	var arraySize = 0
// 	var mask = ((1 << level) - 1)
// 	var totalSize = _HEAD_SIZE + paletteSize
// 	if level != 0 {
// 		blocksPerWord = wordBitSize / level
// 		arraySize = (storageSize + blocksPerWord - 1) / blocksPerWord
// 		totalSize = _HEAD_SIZE + paletteSize + arraySize
// 	}
// 	data := make([]uint32, totalSize)
// 	data[_LEVEL] = uint32(level)
// 	data[_BLKS_PER_WORD] = uint32(blocksPerWord)
// 	data[_MASK] = uint32(mask)
// 	data[_PALE_LAST_IDX] = 0
// 	data[_LI] = 0
// 	data[_LV] = uint32(fill)
// 	data[_PALE_START] = uint32(fill)
// 	return data
// }

// func getPaletteIndexByBlockIdx(data []uint32, posIdx int) int {
// 	level := data[_LEVEL]
// 	if level == 0 {
// 		return int(data[_LI])
// 	}
// 	blocksPerWord := data[_BLKS_PER_WORD]
// 	mask := data[_MASK]
// 	arryStart := _HEAD_SIZE + (1 << level)
// 	index := posIdx / int(blocksPerWord)
// 	offset := (posIdx % int(blocksPerWord)) * int(level)
// 	return int((data[index+int(arryStart)] >> offset) & uint32(mask))
// }

// func setPaletteIndexByBlockIdx(data []uint32, posIdx int, paletteIdx uint32) {
// 	level := data[_LEVEL]
// 	blocksPerWord := data[_BLKS_PER_WORD]
// 	mask := data[_MASK]
// 	arryStart := _HEAD_SIZE + (1 << level)
// 	index := posIdx / int(blocksPerWord)
// 	offset := (posIdx % int(blocksPerWord)) * int(level)
// 	data[index+arryStart] &= ^(mask << offset)
// 	data[index+arryStart] |= (paletteIdx & mask) << offset
// }

// func posToIdx(x, y, z int) int {
// 	x &= 0xf
// 	y &= 0xf
// 	z &= 0xf
// 	return (x << 8) | (z << 4) | y
// }

// func findBlockPaletteIdx(data []uint32, b uint32) (paletteIdx int, found bool) {
// 	if data[_LV] == uint32(b) {
// 		return int(data[_LI]), true
// 	}
// 	for paletteIdx = 0; paletteIdx <= int(data[_PALE_LAST_IDX]); paletteIdx++ {
// 		if data[_PALE_START+paletteIdx] == uint32(b) {
// 			data[_LI] = uint32(paletteIdx)
// 			data[_LV] = uint32(b)
// 			return paletteIdx, true
// 		}
// 	}
// 	return 0, false
// }

// func appendPalette(data []uint32, b uint32) {
// 	idx := data[_PALE_LAST_IDX] + 1
// 	data[_PALE_LAST_IDX] = idx
// 	data[_LI] = idx
// 	data[_LV] = uint32(b)
// 	data[_PALE_START+idx] = uint32(b)
// }

// func (s Storage32[B]) HasData() bool {
// 	if len(s.data) == 0 {
// 		return false
// 	}
// 	return true
// }

// func (s Storage32[B]) Get(x, y, z int) B {
// 	return B(s.data[_PALE_START+getPaletteIndexByBlockIdx(s.data, posToIdx(x, y, z))])
// }

// func (s *Storage32[B]) Fill(data B) {
// 	s.data = makeStorage32ByLevel[B](0, uint32(data))
// }

// func (s *Storage32[B]) Set(x, y, z int, b B) {
// 	if s.data[_LEVEL] == 0 && s.data[_PALE_START] == uint32(b) {
// 		return
// 	}
// 	idx, found := findBlockPaletteIdx(s.data, uint32(b))
// 	if !found {
// 		idx = int(s.data[_PALE_LAST_IDX]) + 1
// 		if idx == (1 << s.data[_LEVEL]) {
// 			newBS := makeStorage32ByLevel[B](int(s.data[_LEVEL]+1), 0)
// 			for i := _PALE_START - 3; i <= int(_PALE_START+newBS[_PALE_LAST_IDX]); i++ {
// 				newBS[i] = s.data[i]
// 			}
// 			for i := 0; i < 4096; i++ {
// 				setPaletteIndexByBlockIdx(newBS, i, uint32(getPaletteIndexByBlockIdx(s.data, i)))
// 			}
// 			s.data = newBS
// 		}
// 		appendPalette(s.data, uint32(b))
// 	}
// 	setPaletteIndexByBlockIdx(s.data, posToIdx(x, y, z), uint32(idx))
// }

// func (s Storage32[B]) Same() (b B, allSame bool) {
// 	if s.data == nil {
// 		return 0, false
// 	}
// 	level := s.data[_LEVEL]
// 	if level == 0 {
// 		return B(s.data[_LV]), true
// 	}
// 	firstPaletteIdx := getPaletteIndexByBlockIdx(s.data, 0)
// 	for i := 1; i < 4096; i++ {
// 		if firstPaletteIdx != getPaletteIndexByBlockIdx(s.data, i) {
// 			return 0, false
// 		}
// 	}
// 	return B(s.data[_PALE_START+firstPaletteIdx]), true
// }

// func (s Storage32[B]) Dump(w io.Writer) {
// 	if !s.HasData() {
// 		writeNumberSimple(w, uint32(0))
// 		return
// 	}
// 	writeNumberSimple(w, uint32(len(s.data)))
// 	p := (unsafe.Pointer(&s.data[0]))
// 	w.Write(unsafe.Slice((*byte)(p), len(s.data)*4))
// }

// func (s *Storage32[B]) Load(r io.Reader) {
// 	dataLen := readNumberSimple[uint32](r)
// 	if dataLen == 0 {
// 		s.data = nil
// 		return
// 	}
// 	data := make([]uint32, dataLen)
// 	reciver := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), dataLen*4)
// 	r.Read(reciver)
// 	s.data = data
// }
