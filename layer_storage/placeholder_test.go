package layer_storage

import (
	"celler"
	"testing"
)

func TestPlaceHolderStorageInterface(t *testing.T) {
	a0 := &PlaceHolder[uint32]{}
	func(celler.In) {}(a0)
	func(celler.Out) {}(a0)
	func(celler.Storage[uint32]) {}(a0)
}
