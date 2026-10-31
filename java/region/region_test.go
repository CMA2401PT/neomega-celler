package region

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"celler"
	"celler/storage"
	"github.com/pierrec/lz4/v4"
)

func TestIndependentRegionHeaderAndExternalRecord(t *testing.T) {
	directory := t.TempDir()
	data := make([]byte, 3*sectorSize)
	// Chunk (-1,-33) occupies local slot (31,31) in region (-1,-2).
	binary.BigEndian.PutUint32(data[1023*4:], 2<<8|1)
	binary.BigEndian.PutUint32(data[sectorSize+1023*4:], 1234)
	binary.BigEndian.PutUint32(data[2*sectorSize:], 1)
	data[2*sectorSize+4] = 128 | Uncompressed
	if err := os.WriteFile(filepath.Join(directory, "r.-1.-2.mca"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "c.-1.-33.mcc"), []byte("native external payload"), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(os.DirFS(directory), ".", celler.ChunkPos{X: -1, Z: -2}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	record, err := r.Read(celler.ChunkPos{X: -1, Z: -33})
	if err != nil {
		t.Fatal(err)
	}
	if string(record.Data) != "native external payload" || record.Timestamp != 1234 || record.Compression != Uncompressed {
		t.Fatal(record)
	}
	if err = r.Write(celler.ChunkPos{X: -1, Z: -33}, record); !errors.Is(err, fs.ErrPermission) {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	// A partial header must not be repaired or replaced by a writable open.
	short := []byte{1, 2, 3}
	if err = os.WriteFile(filepath.Join(directory, "r.0.0.mca"), short, 0644); err != nil {
		t.Fatal(err)
	}
	backend, err := storage.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Open(backend, ".", celler.ChunkPos{}, true); err == nil {
		t.Fatal("accepted truncated header")
	}
	remaining, err := os.ReadFile(filepath.Join(directory, "r.0.0.mca"))
	if err != nil || !bytes.Equal(remaining, short) {
		t.Fatal("modified damaged header", err)
	}
}

func TestCompressionFormatsAndTruncation(t *testing.T) {
	plain := bytes.Repeat([]byte("known payload"), 100)
	zlibRecord, err := Compress(plain, 0)
	if err != nil {
		t.Fatal(err)
	}
	var gzipData bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipData)
	if _, err = gzipWriter.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err = gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	compressed := make([]byte, lz4.CompressBlockBound(len(plain)))
	n, err := lz4.CompressBlock(plain, compressed, nil)
	if err != nil || n == 0 {
		t.Fatal("LZ4 test compression failed", err)
	}
	// Build Java's block framing directly; its end marker is a zero-size block.
	frame := make([]byte, 21+n+21)
	copy(frame, "LZ4Block")
	frame[8] = 0x21
	binary.LittleEndian.PutUint32(frame[9:], uint32(n))
	binary.LittleEndian.PutUint32(frame[13:], uint32(len(plain)))
	copy(frame[21:], compressed[:n])
	copy(frame[21+n:], "LZ4Block")
	frame[21+n+8] = 0x11
	for _, record := range []Record{zlibRecord, {Compression: Gzip, Data: gzipData.Bytes()}, {Compression: Uncompressed, Data: plain}, {Compression: LZ4, Data: frame}} {
		decoded, err := record.Decode()
		if err != nil || !bytes.Equal(decoded, plain) {
			t.Fatalf("compression %d: %v", record.Compression, err)
		}
	}
	if _, err = (Record{Compression: LZ4, Data: frame[:len(frame)-1]}).Decode(); err == nil {
		t.Fatal("accepted truncated LZ4 trailer")
	}
	if _, err = (Record{Compression: Zlib, Data: zlibRecord.Data[:4]}).Decode(); err == nil {
		t.Fatal("accepted truncated zlib")
	}
}
