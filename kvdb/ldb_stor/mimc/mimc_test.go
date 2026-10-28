package mimc

import (
	"bytes"
	"crypto/rand"
	"io"
	"path/filepath"
	"testing"
)

func TestMimcCore(t *testing.T) {
	fs, err := NewMimcCore(filepath.Join(t.TempDir(), "mimc.dat"), 128*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	if fs.String() != "#" {
		t.Fatal(fs.String())
	}

	checkPos := func(seek func(offset int64, whence int) (ret int64, err error), v0, v1, v2 int64) {
		v, _ := seek(0, 1)
		if v != v0 {
			t.Fatal(v)
		}
		v, _ = seek(0, 2)
		if v != v1 {
			t.Fatal(v)
		}
		v, _ = seek(0, 0)
		if v != v2 {
			t.Fatal(v)
		}
	}

	fp := fs.Create(nil)
	checkPos(fp.Seek, 0, 0, 0)

	fp.Write(make([]byte, 1024))
	checkPos(fp.Seek, 1024, 1024, 0)
	fp.Seek(128, 0)
	checkPos(fp.Seek, 128, 1024, 0)
	fp.Seek(128, 0)
	fp.Seek(10, 1)
	checkPos(fp.Seek, 128+10, 1024, 0)

	fp.Seek(30, 0)
	fp.WriteAt(make([]byte, 256), 256)
	checkPos(fp.Seek, 30, 1024, 0)

	rp := fs.Open(fp.Space)
	checkPos(rp.Seek, 0, 1024, 0)
	rp.Read(make([]byte, 512))
	checkPos(rp.Seek, 512, 1024, 0)
	rp.Seek(30, 0)
	rp.ReadAt(make([]byte, 256), 256)
	checkPos(rp.Seek, 30, 1024, 0)

	rp.Seek(128, 0)
	rp.Seek(10, 1)
	checkPos(rp.Seek, 128+10, 1024, 0)
	if fs.String() != "##" {
		t.Fatal(fs.String())
	}

	_, err = fp.Write(make([]byte, fs.ClusterSize*5))
	if err != nil {
		t.Fatal(err)
	}

	fs.MarkOccupy(2, false)
	fs.MarkOccupy(4, false)
	if fs.String() != "##_#_#" {
		t.Fatal(fs.String())
	}

	fakeData := make([]byte, fs.ClusterSize*5+172)
	rand.Read(fakeData)
	w := fs.Create(&Space{Offset: 75})
	w.Write(fakeData)
	if err != nil {
		t.Fatal(err)
	}
	if fs.String() != "##########" {
		t.Fatal(fs.String())
	}
	bak := make([]byte, fs.ClusterSize*5+172)
	r := fs.Open(w.Space)
	_, err = r.Read(bak)
	// err = fs.getData(chain, bak)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(bak, fakeData) != 0 {
		t.Errorf("not euqual")
	}

	fs.Remove(w.Space)
	if !fs.IsOccupy(w.Space.Clusters[0]) {
		t.Error("should not free")
	}
	if fs.IsOccupy(w.Space.Clusters[1]) {
		t.Error("should free")
	}
	if fs.String() != "####_#____" {
		t.Fatal(fs.String())
	}
	s2 := w.Space.Clone()
	s2.Offset = 0
	fs.Remove(s2)
	if fs.IsOccupy(s2.Clusters[0]) {
		t.Error("should free")
	}
	if fs.IsOccupy(s2.Clusters[1]) {
		t.Error("should free")
	}
	s0w := fs.WriteS0()
	s0w.Write(fakeData)
	if fs.Space0.String() != "cap: 719872 consumed: 655532 spare: 64340, clusters: [0 2 4 6 7 8], start: 66560" {
		t.Fatal(string(fs.Space0.String()))
	}

	rand.Read(fakeData)
	fs.WriteAll(fs.Space0, fakeData)

	s0r := fs.ReadS0()
	bakOverflow := make([]byte, fs.ClusterSize*5+172+5)
	ar, err := s0r.Read(bakOverflow)
	if ar != int(fs.ClusterSize)*5+172 {
		t.Error(ar)
	}
	if err != io.EOF {
		t.Error("should EOF")
	}

	if bytes.Compare(bakOverflow[:int(fs.ClusterSize)*5+172], fakeData) != 0 {
		t.Errorf("not euqual")
	}

	bakAll, err := io.ReadAll(fs.Open(fs.Space0))
	if err != nil {
		t.Error(err)
	}
	if len(bakAll) != int(fs.ClusterSize)*5+172 {
		t.Error(len(bakAll))
	}

	if bytes.Compare(bakAll, fakeData) != 0 {
		t.Errorf("not euqual")
	}

}
