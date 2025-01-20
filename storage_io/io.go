package storage_io

import (
	"io"
	"reflect"
	"sort"
	"unsafe"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

func WriteValue[V Number | ~string](w io.Writer, v V) {
	valType := reflect.TypeOf(v)
	if valType.Kind() == reflect.String {
		str := *(*string)(unsafe.Pointer(&v))
		strLen := uint32(len(str))
		w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&strLen)), 4))
		d := unsafe.StringData(str)
		w.Write(unsafe.Slice(d, strLen))
	} else {
		byteSize := reflect.TypeOf(v).Size()
		w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&v)), byteSize))
	}
}

func ReadValue[V Number | ~string](r io.Reader) V {
	var _v V
	valType := reflect.TypeOf(_v)
	if valType.Kind() == reflect.String {
		strLen := uint32(0)
		r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&strLen)), 4))
		reciver := make([]byte, strLen)
		r.Read(reciver)
		return *(*V)(unsafe.Pointer(&reciver))
	} else {
		byteSize := valType.Size()
		holder := unsafe.Slice((*byte)(unsafe.Pointer(&_v)), byteSize)
		r.Read(holder)
		return _v
	}
}

// 内存中连续且不含有指针
func writeArrayContigiousWithoutCount[V any](w io.Writer, arr []V, elemSize int) {
	elementsCount := len(arr)
	if len(arr) > 0 {
		w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&arr[0])), uintptr(elemSize*elementsCount)))
	}
}
func WriteArrayContigiousWithoutCount[V any](w io.Writer, arr []V) {
	val := reflect.ValueOf(arr)
	elemType := val.Type().Elem()
	byteSize := elemType.Size()
	writeArrayContigiousWithoutCount(w, arr, int(byteSize))
}

func WriteArrayWithoutCount[V Number | ~string](w io.Writer, arr []V) {
	if len(arr) == 0 {
		return
	}
	val := reflect.ValueOf(arr)
	elemType := val.Type().Elem()
	elemKind := elemType.Kind()
	if elemKind == reflect.String {
		strArr := unsafe.Slice((*string)(unsafe.Pointer(&arr[0])), len(arr))
		for _, s := range strArr {
			WriteValue(w, s)
		}
	} else {
		byteSize := elemType.Size()
		writeArrayContigiousWithoutCount(w, arr, int(byteSize))
	}
}

func readArrayContigiousWithoutCount[V any](r io.Reader, elemCount int, elemSize int) []V {
	reciver := make([]V, elemCount)
	if elemCount == 0 {
		return reciver
	}
	r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&reciver[0])), uintptr(elemSize*elemCount)))
	return reciver
}
func ReadArrayContigiousWithoutCount[V any](r io.Reader, elemCount int) []V {
	var _v V
	elemType := reflect.TypeOf(_v)
	byteSize := elemType.Size()
	return readArrayContigiousWithoutCount[V](r, elemCount, int(byteSize))
}

func ReadArrayWithoutCount[V Number | ~string](r io.Reader, elemCount int) []V {
	if elemCount == 0 {
		return []V{}
	}
	var _v V
	elemType := reflect.TypeOf(_v)
	elemKind := elemType.Kind()
	if elemKind == reflect.String {
		outStr := make([]string, elemCount)
		for i := 0; i < int(elemCount); i++ {
			outStr[i] = ReadValue[string](r)
		}
		return *(*[]V)(unsafe.Pointer(&outStr))
	} else {
		byteSize := elemType.Size()
		return readArrayContigiousWithoutCount[V](r, elemCount, int(byteSize))
	}
}

func WriteArrayContigious[V any](w io.Writer, arr []V) {
	elemCount := len(arr)
	WriteValue(w, uint32(elemCount))
	WriteArrayContigiousWithoutCount(w, arr)
}

func ReadArrayContigious[V any](r io.Reader) []V {
	elemCount := ReadValue[uint32](r)
	return ReadArrayContigiousWithoutCount[V](r, int(elemCount))
}

func WriteArray[V Number | ~string](w io.Writer, arr []V) {
	elemCount := len(arr)
	WriteValue(w, uint32(elemCount))
	WriteArrayWithoutCount(w, arr)
}

func ReadArray[V Number | ~string](r io.Reader) []V {
	elemCount := ReadValue[uint32](r)
	return ReadArrayWithoutCount[V](r, int(elemCount))
}

func WriteMap[K, V Number | ~string, M ~map[K]V](w io.Writer, m M, order bool) {
	keyArr := make([]K, 0, len(m))
	valArr := make([]V, 0, len(m))
	if !order {
		for k, v := range m {
			keyArr = append(keyArr, k)
			valArr = append(valArr, v)
		}
	} else {
		for k := range m {
			keyArr = append(keyArr, k)

		}
		sort.Slice(keyArr, func(i, j int) bool {
			return keyArr[i] < keyArr[j]
		})
		for _, k := range keyArr {
			valArr = append(valArr, m[k])
		}
	}
	WriteArray(w, keyArr)
	WriteArrayWithoutCount(w, valArr)
}

func ReadMap[K, V Number | ~string, M ~map[K]V](r io.Reader) M {
	keyArr := ReadArray[K](r)
	valArr := ReadArrayWithoutCount[V](r, len(keyArr))
	m := make(M, len(keyArr))
	for i, k := range keyArr {
		m[k] = valArr[i]
	}
	return m
}

func WriteArrayWithFnWithoutCount[E any](w io.Writer, arr []E, writeFn func(io.Writer, E)) {
	for _, e := range arr {
		writeFn(w, e)
	}
}

func ReadArrayWithFnWithoutCount[E any](r io.Reader, readFn func(io.Reader) E, count int) []E {
	reciver := make([]E, count)
	for i := range reciver {
		reciver[i] = readFn(r)
	}
	return reciver
}

func WriteArrayWithFn[E any](w io.Writer, arr []E, writeFn func(io.Writer, E)) {
	elementsCount := len(arr)
	WriteValue(w, uint32(elementsCount))
	WriteArrayWithFnWithoutCount(w, arr, writeFn)
}

func ReadArrayWithFn[E any](r io.Reader, readFn func(io.Reader) E) []E {
	lenData := ReadValue[uint32](r)
	return ReadArrayWithFnWithoutCount(r, readFn, int(lenData))
}
