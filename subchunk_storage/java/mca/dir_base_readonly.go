package mca

import (
	"fmt"
	"os"
	"path/filepath"
)

type dirBaseRoStor struct {
	dir string
}

func (s *dirBaseRoStor) Close() error {
	return nil
}

func (s *dirBaseRoStor) IterRegions(cb func(RegionPos) (next bool)) {
	fs, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		baseName := f.Name()
		rp, err := FileNameToRegionPos(baseName)
		if err != nil {
			continue
		}
		if !cb(rp) {
			return
		}
	}
}

func (s *dirBaseRoStor) OpenRegion(rp RegionPos) (Reader, error) {
	baseName := RegionPosToFileName(rp)
	fullName := filepath.Join(s.dir, baseName)
	fp, err := os.OpenFile(fullName, os.O_RDONLY, 0755)
	return fp, err
}

func OpenDirReadOnly(dir string) (ReadOnlyStor, error) {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		if err != nil {
			return nil, err
		} else {
			return nil, fmt.Errorf("%v is not a dir", dir)
		}
	}
	return &dirBaseRoStor{
		dir: dir,
	}, nil
}
