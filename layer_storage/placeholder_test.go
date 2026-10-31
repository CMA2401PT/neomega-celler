package layer_storage

import (
	"testing"
)

func TestPlaceHolderStorageInterface(t *testing.T) {
	a0 := &PlaceHolder[uint32]{}
	func(Storage[uint32]) {}(a0)
}
