package mca

import (
	"os"
	"path/filepath"
	"sync"
)

var _ ReadWriteStor = &dirBaseRWStor{}

type dirBaseRWStor struct {
	*dirBaseRoStor
}

func (s *dirBaseRWStor) OpenRegion(rp RegionPos) (RW, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.fhCache[rp]; ok {
		return w.File, nil
	}
	baseName := RegionPosToFileName(rp)
	fullName := filepath.Join(s.dir, baseName)
	fp, err := os.OpenFile(fullName, os.O_CREATE|os.O_RDWR, 0755)
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

func OpenDirReadWrite(dir string) (ReadWriteStor, error) {
	os.MkdirAll(dir, 0755)
	return &dirBaseRWStor{
		&dirBaseRoStor{
			dir:     dir,
			fhCache: map[RegionPos]*WF{},
			mu:      sync.RWMutex{},
		},
	}, nil
}
