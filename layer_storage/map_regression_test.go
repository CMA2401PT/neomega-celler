package layer_storage

import (
	"testing"
)

func TestMapDefaultAndUniformity(t *testing.T) {
	s := MakeMapStorage[uint32](100)
	p := At(2, 3, 4)
	if got := s.Get(p); got != 100 {
		t.Fatalf("missing entry = %d, want 100", got)
	}
	s.Set(p, 7)
	if _, same := s.Same(); same {
		t.Fatal("partial non-default map reported uniform")
	}
	s.Set(p, 100)
	if got, same := s.Same(); !same || got != 100 {
		t.Fatalf("reset: (%d, %v)", got, same)
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				s.Set(At(x, y, z), 7)
			}
		}
	}
	if got, same := s.Same(); !same || got != 7 {
		t.Fatalf("filled: (%d, %v)", got, same)
	}
	s.Set(p, 100)
	var loaded Map[uint32]
	loaded.CopyFrom(s)
	if loaded.Get(p) != 100 {
		t.Fatal("default lost after copy")
	}
	if _, same := loaded.Same(); same {
		t.Fatal("one default position lost after copy")
	}
}
