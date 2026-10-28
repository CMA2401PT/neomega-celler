package test

import (
	"celler/kvdb"
	"path/filepath"
	"testing"
)

func TestMemoryLevelDB(t *testing.T) {
	db, err := kvdb.NewReadWriteLevelDB("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	testKVDB(t, db)
}

func TestDirLevelDB(t *testing.T) {

	db, err := kvdb.NewReadWriteLevelDB(filepath.Join(t.TempDir(), "dbDir"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	testKVDB(t, db)
}

func TestMimcLevelDB(t *testing.T) {

	db, err := kvdb.NewReadWriteLevelDB(filepath.Join(t.TempDir(), "db.nomg"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	testKVDB(t, db)
}
