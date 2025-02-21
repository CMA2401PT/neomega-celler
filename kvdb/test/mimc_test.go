package test

import (
	"bytes"
	"celler/kvdb"
	"fmt"
	"os"
	"testing"
)

func TestMimc(t *testing.T) {
	// cpy stage
	srcDB, err := kvdb.NewReadWriteLevelDB("./newUTB")
	if err != nil {
		panic(err)
	}
	datas := map[string][]byte{}
	for kv := range srcDB.IterAll {
		datas[string(kv.Key())] = bytes.Clone(kv.Value())
	}
	srcDB.Close()

	os.Remove("utb.nomg")
	dstDB, err := kvdb.NewReadWriteLevelDB("utb.nomg")
	if err != nil {
		panic(err)
	}
	fmt.Println("copying")
	for k, v := range datas {
		dstDB.Put([]byte(k), v)
	}

	checkSame := func(DB kvdb.ReadOnlyKVDB) {
		fmt.Println("checking")
		for k, v := range datas {
			if backV, _ := DB.Get([]byte(k)); !bytes.Equal(v, backV) {
				t.Fatal("not equal")
			}
		}
		for kv := range DB.IterAll {
			k := string(kv.Key())
			if !bytes.Equal(datas[k], kv.Value()) {
				t.Fatal("not equal")
			}
		}
		fmt.Println("check complete")
	}
	checkSame(dstDB)
	dstDB.Close()

	fmt.Println("reopening")

	// dstDB, err = NewReadWriteMimcLevelDB("utb.mimc")
	// if err != nil {
	// 	panic(err)
	// }
	// checkSame()
	// dstDB.Close()

	fmt.Println("shrinking")
	kvdb.ShrinkMimcLevelDB("utb.nomg")
	dstDB, err = kvdb.NewReadWriteLevelDB("utb.nomg")
	if err != nil {
		panic(err)
	}
	checkSame(dstDB)

	dstDB.Close()

	fmt.Println("open read only")
	dstDBRo, err := kvdb.NewReadWriteLevelDB("utb.nomg")
	if err != nil {
		panic(err)
	}
	checkSame(dstDBRo)
	dstDB.Close()
	kvdb.ShrinkMimcLevelDB("utb.nomg")
}

func TestSmallestMimc(t *testing.T) {
	dstDB, err := kvdb.NewReadWriteLevelDB("small.nomg")
	if err != nil {
		panic(err)
	}
	defer dstDB.Close()
	dstDB.Put([]byte{0, 0, 1}, make([]byte, 1024))
}
