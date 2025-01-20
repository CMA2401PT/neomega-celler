package grid

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestZip(t *testing.T) {
	outbuf := bytes.NewBuffer(nil)
	w := zip.NewWriter(outbuf)
	fw, err := w.Create("a")
	if err != nil {
		panic(err)
	}
	randomData := make([]byte, 1<<20)
	rand.Read(randomData)
	fw.Write(randomData)
	w.Close()
	fmt.Println(len(outbuf.Bytes()))

	outbuf = bytes.NewBuffer(nil)
	w = zip.NewWriter(outbuf)

	for i := 0; i < 100; i++ {
		fw, err = w.Create(fmt.Sprintf("%v", i))
		if err != nil {
			panic(err)
		}
		fw.Write(randomData)
	}
	w.Close()
	fmt.Println(len(outbuf.Bytes()))
}
