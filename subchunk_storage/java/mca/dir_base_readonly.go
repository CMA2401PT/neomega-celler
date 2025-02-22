package mca

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var _ ReadOnlyStor = &dirBaseRoStor{}

type WF struct {
	*os.File
	onClose func()
}

type dirBaseRoStor struct {
	dir     string
	fhCache map[RegionPos]*WF
	mu      sync.RWMutex
}

func (s *dirBaseRoStor) Close() error {
	for _, w := range s.fhCache {
		w.Close()
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.fhCache[rp]; ok {
		return w.File, nil
	}
	baseName := RegionPosToFileName(rp)
	fullName := filepath.Join(s.dir, baseName)
	fp, err := os.OpenFile(fullName, os.O_RDONLY, 0755)
	s.fhCache[rp] = &WF{
		File: fp,
		onClose: func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.fhCache, rp)
		},
	}
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
		dir:     dir,
		fhCache: map[RegionPos]*WF{},
		mu:      sync.RWMutex{},
	}, nil
}
