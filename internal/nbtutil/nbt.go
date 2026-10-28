// Package nbtutil shares the NBT options used by world storage adapters.
package nbtutil

import (
	"bytes"
	"io"
	nbt "neomega-nbt/presets/v1"
)

// World NBT stores numeric slices as arrays, not lists.
func Encode(value map[string]any, encoding nbt.Encoding) ([]byte, error) {
	opt := nbt.OptEncodeToNBT{Encoding: encoding}
	opt.SliceAsArray = true
	return nbt.EncodeToNBTBytes(value, opt)
}

func Decode(data []byte, encoding nbt.Encoding) (map[string]any, error) {
	return Read(bytes.NewReader(data), encoding)
}

// Read consumes exactly one root; Bedrock stores consecutive NBT roots.
func Read(reader io.Reader, encoding nbt.Encoding) (map[string]any, error) {
	var value map[string]any
	opt := nbt.OptDecodeFromNBT{Encoding: encoding}
	opt.ArrayAsSlice = true
	err := nbt.DecodeFromNBT(reader, &value, opt)
	return value, err
}

func ToSNBT(value map[string]any) (string, error) {
	return nbt.EncodeToSNBT(value, nbt.OptEncodeToSNBT{SliceAsArray: true})
}

func FromSNBT(data string) (map[string]any, error) {
	var value map[string]any
	err := nbt.DecodeFromSNBT(data, &value, nbt.OptDecodeFromSNBT{ArrayAsSlice: true})
	return value, err
}

// Coordinates in world NBT are TAG_Int values.
func Coordinate(value map[string]any, key string, fallback int) int {
	if v, ok := value[key].(int32); ok {
		return int(v)
	}
	return fallback
}

// Compounds accepts a typed compound list and the untyped empty-list marker.
func Compounds(value any) ([]map[string]any, bool) {
	if values, ok := value.([]map[string]any); ok {
		return values, true
	}
	if values, ok := value.([]any); ok && len(values) == 0 {
		return nil, true
	}
	return nil, false
}
