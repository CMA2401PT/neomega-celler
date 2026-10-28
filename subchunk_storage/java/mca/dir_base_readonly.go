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
	onClose   func()
	closeOnce sync.Once
	closeErr  error
}

// Closing a region must also invalidate the directory's handle cache.
func (w *WF) Close() error {
	w.closeOnce.Do(func() { w.closeErr = w.File.Close(); w.onClose() })
	return w.closeErr
}

type dirBaseRoStor struct {
	dir     string
	fhCache map[RegionPos]*WF
	mu      sync.RWMutex
}

func (s *dirBaseRoStor) Close() error {
	s.mu.RLock()
	handles := make([]*WF, 0, len(s.fhCache))
	for _, w := range s.fhCache {
		handles = append(handles, w)
	}
	s.mu.RUnlock()
	for _, w := range handles {
		if err := w.Close(); err != nil {
			return err
		}
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
		return w, nil
	}
	baseName := RegionPosToFileName(rp)
	fullName := filepath.Join(s.dir, baseName)
	fp, err := os.OpenFile(fullName, os.O_RDONLY, 0755)
	if err != nil {
		return nil, err
	}
	s.fhCache[rp] = &WF{
		File: fp,
		onClose: func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.fhCache, rp)
		},
	}
	return s.fhCache[rp], nil
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
