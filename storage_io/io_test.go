package storage_io

import (
	"bytes"
	"testing"
)

func inAndOut[T Number](data T, t *testing.T) {
	outBuf := bytes.NewBuffer(nil)
	WriteValue(outBuf, data)
	inBuf := bytes.NewReader(outBuf.Bytes())
	back := ReadValue[T](inBuf)
	if back != data {
		t.Errorf("%v != %v", back, data)
	}
}

func inAndOutArr[V Number | ~string, T ~[]V](data T, t *testing.T) {
	outBuf := bytes.NewBuffer(nil)
	WriteArray(outBuf, data)
	inBuf := bytes.NewReader(outBuf.Bytes())
	back := ReadArray[V](inBuf)
	for i, in := range data {
		if in != back[i] {
			t.Errorf("%v != %v @ %v", in, back[i], i)
		}
	}
}

func inAndOutArrAny[V comparable](data []V, t *testing.T) {
	outBuf := bytes.NewBuffer(nil)
	WriteArrayContigious(outBuf, data)
	inBuf := bytes.NewReader(outBuf.Bytes())
	back := ReadArrayContigious[V](inBuf)
	for i, in := range data {
		if in != back[i] {
			t.Errorf("%v != %v @ %v", in, back[i], i)
		}
	}
}

func inAndOutString[T ~string](data T, t *testing.T) {
	outBuf := bytes.NewBuffer(nil)
	WriteValue(outBuf, data)
	inBuf := bytes.NewReader(outBuf.Bytes())
	back := ReadValue[string](inBuf)
	if back != string(data) {
		t.Errorf("%v != %v", back, data)
	}
}

type SubChunkBlockPos struct {
	X, Y, Z uint8 // 以 SubChunk 为基准的偏移
}

type SubChunkBlockDeltaNoSNBT struct {
	// 考虑到一个方块具有多个分离的属性，可能一次变更一个或者多个，使用 Mark 来标记其变更
	ChangeType byte
	// 变更的方块相对于子区块位置
	Pos     SubChunkBlockPos
	BlockID uint32
	Attach  uint16
}

func TestIO(t *testing.T) {
	inAndOut(uint32(123), t)
	inAndOut(uint8(123), t)
	inAndOut(int16(123), t)

	inAndOutString("abc", t)

	inAndOutArr([]uint32{1, 2, 3, 4}, t)
	inAndOutArr([]uint8{1, 2, 3}, t)
	inAndOutArr([]string{"abc", "2abc", "3abc", "4abc5"}, t)
	inAndOutArrAny([]SubChunkBlockDeltaNoSNBT{
		SubChunkBlockDeltaNoSNBT{
			2, SubChunkBlockPos{3, 4, 5}, 6, 7,
		},
		SubChunkBlockDeltaNoSNBT{
			8, SubChunkBlockPos{9, 10, 11}, 12, 13,
		},
	}, t)
}
