// Package mcworld 只读打开 .mcworld。
// 文件本身是一个 zip，里面是基岩世界。这里不把条目写到临时目录，LevelDB 直接读 zip。
package mcworld

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"celler/bedrock"
	db "github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

// World 借用 zip 里的基岩世界。Close 先关数据库，再放开 zip 文件。
type World struct {
	bedrock *bedrock.World
	file    *os.File
	storage *zipStorage
}

// Open 打开一个 .mcworld。世界可以在压缩包根上，也可以在唯一的一层目录里。
func Open(filename string) (_ *World, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	opened := &World{file: file}
	defer func() {
		if err != nil {
			opened.Close()
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, err
	}
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		name := path.Clean(strings.TrimPrefix(entry.Name, "/"))
		if name == "." || strings.HasSuffix(entry.Name, "/") {
			continue
		}
		entries[name] = entry
	}
	root, err := worldRoot(entries)
	if err != nil {
		return nil, err
	}
	opened.storage = newZipStorage(file, dbEntries(entries, root))
	database, err := db.Open(opened.storage, &opt.Options{
		ReadOnly:       true,
		ErrorIfMissing: true,
		Compression:    opt.FlateCompression,
		BlockSize:      16 * opt.KiB,
	})
	if err != nil {
		return nil, err
	}
	opened.bedrock = bedrock.NewReadOnly(database, &zipFiles{root: root, files: entries})
	return opened, nil
}

// Bedrock 返回可供区块读取使用的世界。关闭权仍在 World 上。
func (w *World) Bedrock() *bedrock.World { return w.bedrock }

func (w *World) Close() error {
	var err error
	if w.bedrock != nil {
		err = w.bedrock.Close()
		w.bedrock = nil
	}
	if w.storage != nil {
		err = errors.Join(err, w.storage.Close())
		w.storage = nil
	}
	if w.file != nil {
		err = errors.Join(err, w.file.Close())
		w.file = nil
	}
	return err
}

// worldRoot 返回世界在压缩包里的目录前缀，世界在根上时为空串。
// level.dat 或 db/ 出现在同一层才算一个世界，出现两个世界就报错。
func worldRoot(entries map[string]*zip.File) (string, error) {
	if hasWorld(entries, "") {
		return "", nil
	}
	prefixes := map[string]struct{}{}
	for name := range entries {
		dir, _, ok := strings.Cut(name, "/")
		if ok && dir != "" {
			prefixes[dir] = struct{}{}
		}
	}
	var found string
	for prefix := range prefixes {
		if !hasWorld(entries, prefix+"/") {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("mcworld: more than one world in archive")
		}
		found = prefix + "/"
	}
	if found == "" {
		return "", fmt.Errorf("mcworld: archive has no bedrock world")
	}
	return found, nil
}

// hasWorld 检查 prefix 这一层是否像基岩世界：有 level.dat，或 db/ 下有 LevelDB 文件。
func hasWorld(entries map[string]*zip.File, prefix string) bool {
	if entries[prefix+"level.dat"] != nil || entries[prefix+"db/CURRENT"] != nil {
		return true
	}
	for name := range entries {
		base, ok := strings.CutPrefix(name, prefix+"db/")
		if ok && !strings.Contains(base, "/") && (strings.HasSuffix(base, ".ldb") || strings.HasSuffix(base, ".sst")) {
			return true
		}
	}
	return false
}

// dbEntries 取出世界 db/ 下的顶层文件，键是去掉前缀的文件名。
// LevelDB 只读 CURRENT、MANIFEST 和 *.ldb/*.sst 这些顶层文件，子目录不参与。
func dbEntries(entries map[string]*zip.File, root string) map[string]*zip.File {
	out := map[string]*zip.File{}
	prefix := root + "db/"
	for name, file := range entries {
		base, ok := strings.CutPrefix(name, prefix)
		if ok && base != "" && !strings.Contains(base, "/") {
			out[base] = file
		}
	}
	return out
}

// zipFiles 只为 level.dat 这类世界文件提供读取。db 由 LevelDB 存储单独打开。
type zipFiles struct {
	root  string
	files map[string]*zip.File
}

func (z *zipFiles) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	file := z.files[z.root+name]
	if file == nil {
		return nil, fs.ErrNotExist
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	return &zipFile{ReadCloser: reader, name: name, size: int64(file.UncompressedSize64)}, nil
}

type zipFile struct {
	io.ReadCloser
	name string
	size int64
}

func (f *zipFile) Stat() (fs.FileInfo, error) {
	return zipInfo{name: path.Base(f.name), size: f.size}, nil
}

type zipInfo struct {
	name string
	size int64
}

func (i zipInfo) Name() string       { return i.name }
func (i zipInfo) Size() int64        { return i.size }
func (i zipInfo) Mode() os.FileMode  { return 0444 }
func (i zipInfo) ModTime() time.Time { return time.Time{} }
func (i zipInfo) IsDir() bool        { return false }
func (i zipInfo) Sys() any           { return nil }
