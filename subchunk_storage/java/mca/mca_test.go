package mca

import (
	"bytes"
	"celler"
	"testing"
)

func TestMCA(t *testing.T) {
	dir := t.TempDir()
	values := map[celler.ChunkPos][]byte{
		{X: 1, Z: 3}:   bytes.Repeat([]byte("first"), 1700),
		{X: -2, Z: 35}: []byte("negative coordinate"),
	}
	db, err := NewReadWriteMCAKvFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for pos, value := range values {
		if err := db.Put(pos, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := NewReadOnlyMCAKvFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	seen := map[celler.ChunkPos]bool{}
	for pos := range ro.IterAll {
		seen[pos] = true
		value, err := ro.Get(pos)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value, values[pos]) {
			t.Fatalf("wrong data/coordinates at %v", pos)
		}
	}
	if len(seen) != len(values) {
		t.Fatalf("enumerated %d chunks, want %d", len(seen), len(values))
	}
	for pos := range values {
		if !seen[pos] {
			t.Fatalf("missing chunk %v", pos)
		}
	}
}

func TestMCAReopenEvictedRegion(t *testing.T) {
	dir := t.TempDir()
	db, err := NewReadWriteMCAKvFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// More than 16 regions exceeds the region cache capacity.
	for i := 0; i < 18; i++ {
		pos := celler.ChunkPos{X: celler.ChunkUnit(i * 32), Z: 3}
		if err := db.Put(pos, []byte{byte(i)}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 18; i++ {
		pos := celler.ChunkPos{X: celler.ChunkUnit(i * 32), Z: 3}
		value, err := db.Get(pos)
		if err != nil || !bytes.Equal(value, []byte{byte(i)}) {
			t.Fatalf("region %d after eviction: %v, %v", i, value, err)
		}
	}
}
