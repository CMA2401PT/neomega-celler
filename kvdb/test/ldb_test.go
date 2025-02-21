package test

import (
	"celler/kvdb"
	"os"
	"testing"
)

func TestMemoryLevelDB(t *testing.T) {
	db, err := kvdb.NewReadWriteLevelDB("")
	if err != nil {
		t.Fatal(err)
	}
	testKVDB(t, db)
}

func TestDirLevelDB(t *testing.T) {
	os.RemoveAll("dbDir")
	db, err := kvdb.NewReadWriteLevelDB("dbDir")
	if err != nil {
		t.Fatal(err)
	}
	testKVDB(t, db)
}

func TestMimcLevelDB(t *testing.T) {
	os.RemoveAll("db.nomg")
	db, err := kvdb.NewReadWriteLevelDB("db.nomg")
	if err != nil {
		t.Fatal(err)
	}
	testKVDB(t, db)
}
