package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"celler/storage"
)

// 结构转换在临时目录内提交，文件可能已关闭、改名，或仍由 region 缓存持有。
// 此测试在 Windows 上同时覆盖文件同步权限与父目录同步的差异。
func TestDirectoryCommitBeforePublishingWorld(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "结构文件 (worldedit)")
	root := filepath.Join(parent, ".structure-test", "world")
	directory, err := storage.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = directory.MkdirAll("region"); err != nil {
		t.Fatal(err)
	}
	if err = storage.WriteFile(directory, "level.dat.tmp", []byte("level")); err != nil {
		t.Fatal(err)
	}
	if err = directory.Rename("level.dat.tmp", "level.dat"); err != nil {
		t.Fatal(err)
	}
	region, err := directory.OpenFile("region/r.0.0.mca", os.O_RDWR|os.O_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	defer region.Close()
	if _, err = region.WriteAt([]byte("region"), 0); err != nil {
		t.Fatal(err)
	}
	if err = directory.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = region.Close(); err != nil {
		t.Fatal(err)
	}
	// 提交成功后，空提交也应成功，随后才能将整个临时世界发布为同名目录。
	if err = directory.Commit(); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(parent, "城堡")
	if err = os.Rename(root, output); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"level.dat": "level", "region/r.0.0.mca": "region"} {
		data, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(name)))
		if err != nil || string(data) != want {
			t.Fatalf("%s: %q, %v", name, data, err)
		}
	}
}
