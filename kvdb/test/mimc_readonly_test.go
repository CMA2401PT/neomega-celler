package test

import (
	"bytes"
	"celler/kvdb"
	"celler/kvdb/ldb_stor/mimc"
	"os"
	"path/filepath"
	"testing"
)

func TestMimcExtensionAndReadOnlyOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.mimc")
	db, err := kvdb.NewReadWriteLevelDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := kvdb.NewReadOnlyLevelDB(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ro.Get([]byte("key"))
	if err != nil || !bytes.Equal(got, []byte("value")) {
		t.Fatalf("%q: %v", got, err)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing.mimc")
	if _, err := kvdb.NewReadOnlyLevelDB(missing); !os.IsNotExist(err) {
		t.Fatalf("missing file: %v", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("read-only open created a missing database")
	}
}

func TestMimcInvalidHeaderReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.mimc")
	if err := os.WriteFile(path, make([]byte, 128*1024), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := mimc.NewReadOnlyMimcCore(path, 128*1024); err == nil {
		t.Fatal("invalid header was accepted")
	}
}
