package mimc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"sync"
)

// const ClusterSize = 128 * (1 << 10)

type MimcCore struct {
	fp           *os.File
	readOnly     bool
	emptyCluster []byte
	occupyTable  []byte
	currsize     int64
	metaMu       sync.RWMutex
	ClusterSize  int64
	Space0       *Space
}

func (fs *MimcCore) Close() {
	if !fs.readOnly {
		fs.fp.Sync()
	}
	fs.fp.Close()
}
func (fs *MimcCore) shrink() {
	totalCluster := fs.currentCluster()
	orig := totalCluster
	for i := fs.currentCluster() - 1; i > 0; i-- {
		if !fs.isOccupy(i) {
			totalCluster -= 1
		} else {
			break
		}
	}
	fmt.Printf("shrink from %v(%v) -> %v(%v)\n", orig, orig*fs.ClusterSize/(1024*1024), totalCluster, totalCluster*fs.ClusterSize/(1024*1024))
	fs.fp.Seek(0, 0)
	fs.fp.Truncate(totalCluster * fs.ClusterSize)
	fs.currsize = totalCluster * fs.ClusterSize
	fs.fp.Sync()
}

func (fs *MimcCore) Shrink() {
	fs.metaMu.Lock()
	defer fs.metaMu.Unlock()
	fs.shrink()
}

func (fs *MimcCore) MapAndShrink(cs []int64) map[int64]int64 {
	cs = append(cs, 0)
	slices.Sort(cs)
	cs = slices.Compact(cs)
	used := int64(len(cs))
	fs.metaMu.Lock()
	defer fs.metaMu.Unlock()
	if used >= fs.currentCluster() {
		return nil
	}
	p := 0
	avail := []int64{}
	i := int64(0)
	for ; i < used; i++ {
		if i < cs[p] {
			avail = append(avail, i)
		} else if i == cs[p] {
			p += 1
		} else {
			for i > cs[p] && p < int(used) {
				p += 1
			}
		}
	}
	cMap := map[int64]int64{}
	buf := make([]byte, fs.ClusterSize)
	for i, origCi := range cs[p:] {
		newCi := avail[i]
		cMap[origCi] = newCi
		fmt.Printf("move %v -> %v\n", origCi, newCi)
		n, err := fs.fp.ReadAt(buf, origCi*fs.ClusterSize)
		if err != nil {
			panic(err)
		}
		if n != int(fs.ClusterSize) {
			panic(fmt.Errorf("%v!=%v", n, int(fs.ClusterSize)))
		}
		n, err = fs.fp.WriteAt(buf, newCi*fs.ClusterSize)
		if err != nil {
			panic(err)
		}
		if n != int(fs.ClusterSize) {
			panic(fmt.Errorf("%v!=%v", n, int(fs.ClusterSize)))
		}
		err = fs.markOccupy(origCi, false, false)
		if err != nil {
			panic(err)
		}
		err = fs.markOccupy(newCi, true, false)
		if err != nil {
			panic(err)
		}
	}
	fs.shrink()
	return cMap
}

func (fs *MimcCore) init() error {
	fs.emptyCluster = make([]byte, fs.ClusterSize)
	fs.occupyTable = make([]byte, fs.ClusterSize/2)
	fs.metaMu = sync.RWMutex{}
	// check if is just created
	var err error
	fs.currsize, err = fs.fp.Seek(0, 2)
	if err != nil {
		return err
	}
	s0SpaceStart := int64((fs.ClusterSize / 2) + 16)
	s0SpaceMax := int64(1024 - 16)
	updateSpace0 := func(s0 *Space) {
		buf := bytes.NewBuffer(nil)
		err := s0.Dump(buf.WriteByte)
		if err != nil {
			panic(err)
		}
		err = fs.WriteSingleCluster(0, s0SpaceStart, buf.Bytes())
		if err != nil {
			panic(err)
		}
	}
	if fs.CurrentSize() == 0 {
		if fs.readOnly {
			return fmt.Errorf("empty MIMC file")
		}
		ci, err := fs.extendCluster()
		if err != nil {
			return err
		}
		if ci != 0 {
			panic(fmt.Errorf("BUG: %v", ci))
		}
		err = fs.markOccupy(0, true, false)
		if err != nil {
			return err
		}
		sz := []byte{'2', '4', '0', '1', 0, 0, 0, 0, 0, 0, 0, 0}
		binary.LittleEndian.PutUint64(sz[4:], uint64(fs.ClusterSize))
		fs.WriteSingleCluster(0, fs.ClusterSize/2, sz)
		updateSpace0(&Space{
			Clusters:    []int64{0},
			Offset:      s0SpaceStart + s0SpaceMax,
			WritedLen:   0,
			ClusterSize: fs.ClusterSize,
			changeCB:    updateSpace0,
		})
	} else {
		sz := make([]byte, 12)
		if err := fs.ReadSingleCluster(0, fs.ClusterSize/2, sz); err != nil {
			return err
		}
		if string(sz[:4]) != "2401" || binary.LittleEndian.Uint64(sz[4:]) != uint64(fs.ClusterSize) {
			return fmt.Errorf("invalid MIMC file header")
		}
	}
	// fmt.Println("current clusters:", fs.currentCluster())
	fs.ReadSingleCluster(0, 0, fs.occupyTable)
	s0D := make([]byte, s0SpaceMax)
	fs.ReadSingleCluster(0, s0SpaceStart, s0D)
	s0B := bytes.NewBuffer(s0D)
	fs.Space0, err = LoadSpace(s0B.ReadByte)
	if err != nil {
		return err
	}
	fs.Space0.SetChangeHook(updateSpace0)
	return nil
}

func (fs *MimcCore) WriteS0() *WriteHandle {
	return fs.Create(fs.Space0)
}

func (fs *MimcCore) ReadS0() *ReadHandle {
	return fs.Open(fs.Space0)
}

func (fs *MimcCore) CurrentSize() int64 {
	fs.metaMu.RLock()
	defer fs.metaMu.RUnlock()
	return fs.currentSize()
}

func (fs *MimcCore) currentSize() int64 {
	return fs.currsize
}

func (fs *MimcCore) CurrentCluster() int64 {
	return fs.CurrentSize() / fs.ClusterSize
}

func (fs *MimcCore) currentCluster() int64 {
	return fs.currentSize() / fs.ClusterSize
}

func (fs *MimcCore) isOccupy(ci int64) bool {
	return (fs.occupyTable[ci/8] & (1 << (ci % 8))) != 0
}

func (fs *MimcCore) IsOccupy(ci int64) bool {
	fs.metaMu.RLock()
	defer fs.metaMu.RUnlock()
	return (fs.occupyTable[ci/8] & (1 << (ci % 8))) != 0
}

func (fs *MimcCore) markOccupy(ci int64, occupy bool, lock bool) error {
	if lock {
		fs.metaMu.Lock()
	}
	if ci >= fs.currentCluster() {
		panic(fmt.Errorf("BUG: mark %v >= %v", ci, fs.currentCluster()))
	}
	if occupy == fs.isOccupy(ci) {
		if lock {
			fs.metaMu.Unlock()
		}
		return nil
	}
	mb := byte(0)
	if occupy {
		mb = fs.occupyTable[ci/8] | (1 << (ci % 8))
	} else {
		mb = fs.occupyTable[ci/8] & (^(1 << (ci % 8)))
	}
	fs.occupyTable[ci/8] = mb
	if lock {
		fs.metaMu.Unlock()
	}
	n, err := fs.fp.WriteAt([]byte{mb}, ci/8)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("BUG: want write 1 byte @ %v, actual write: %v", ci/8, n)
	}
	return nil
}

func (fs *MimcCore) MarkOccupy(ci int64, occupy bool) error {
	return fs.markOccupy(ci, occupy, true)
}

func (fs *MimcCore) String() string {
	occupyStr := ""
	for i := int64(0); i < fs.CurrentCluster(); i++ {
		if fs.IsOccupy(i) {
			occupyStr += "#"
		} else {
			occupyStr += "_"
		}
	}
	return occupyStr
}

func (fs *MimcCore) WriteSingleCluster(ci int64, offset int64, data []byte) error {
	if int64(len(data)+int(offset)) > fs.ClusterSize {
		panic(fmt.Errorf("BUG: %v>%v", len(data), fs.ClusterSize))
	}
	n, err := fs.fp.WriteAt(data, ci*fs.ClusterSize+offset)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("BUG: want write %v, actual write: %v", len(data), n)
	}
	return nil
}

func (fs *MimcCore) ReadSingleCluster(ci int64, offset int64, data []byte) error {
	if int64(len(data)+int(offset)) > fs.ClusterSize {
		panic(fmt.Errorf("BUG: %v>%v", len(data), int(ci)))
	}
	if ci >= fs.CurrentCluster() {
		panic(fmt.Errorf("BUG: %v>%v", ci, fs.CurrentCluster()))
	}
	n, err := fs.fp.ReadAt(data, ci*fs.ClusterSize+offset)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("BUG: want read %v, actual write: %v", len(data), n)
	}
	return nil
}

func (fs *MimcCore) extendCluster() (int64, error) {
	fs.metaMu.Lock()
	defer fs.metaMu.Unlock()
	ci := fs.currentCluster()
	size := fs.currentSize()
	if ci == int64(len(fs.occupyTable)*8) {
		return 0, fmt.Errorf("size limit reached")
	}
	n, err := fs.fp.WriteAt(fs.emptyCluster, size)
	if err != nil {
		return 0, err
	}
	if n != int(fs.ClusterSize) {
		return 0, fmt.Errorf("fail to append cluster, actual append: %v", n)
	}
	fs.currsize += fs.ClusterSize
	fs.markOccupy(ci, true, false)
	return ci, nil
}

func (fs *MimcCore) AllocClusters(clustersReq int) ([]int64, error) {
	out := []int64{}
	fs.metaMu.Lock()
	for i := int64(0); i < fs.currentCluster(); i++ {
		if clustersReq == 0 {
			break
		}
		if !fs.isOccupy(i) {
			err := fs.markOccupy(i, true, false)
			if err != nil {
				return nil, err
			}
			out = append(out, i)
			clustersReq -= 1
		}
	}
	fs.metaMu.Unlock()
	if clustersReq == 0 {
		return out, nil
	}
	for clustersReq > 0 {
		ci, err := fs.extendCluster()
		if err != nil {
			return nil, err
		}
		out = append(out, ci)
		clustersReq -= 1
	}
	return out, nil
}

func (fs *MimcCore) Open(space *Space) *ReadHandle {
	if space == nil {
		space = &Space{}
	}
	if space.Clusters == nil {
		space.Clusters = make([]int64, 0)
	}
	if space.ClusterSize == 0 {
		space.ClusterSize = fs.ClusterSize
	} else {
		if space.ClusterSize != fs.ClusterSize {
			panic(fmt.Errorf("%v !=%v", space.ClusterSize, fs.ClusterSize))
		}
	}
	return &ReadHandle{
		fs:    fs,
		Space: space,
		p:     0,
	}
}

func (fs *MimcCore) WriteAll(space *Space, data []byte) {
	fs.Create(space).Write(data)
}

func (fs *MimcCore) Create(space *Space) *WriteHandle {
	if space == nil {
		space = &Space{}
	}
	if space.Clusters == nil {
		space.Clusters = make([]int64, 0)
	}
	if space.ClusterSize == 0 {
		space.ClusterSize = fs.ClusterSize
	} else {
		if space.ClusterSize != fs.ClusterSize {
			panic(fmt.Errorf("%v !=%v", space.ClusterSize, fs.ClusterSize))
		}
	}
	return &WriteHandle{
		fs:    fs,
		Space: space,
		p:     0,
	}
}

func (fs *MimcCore) Remove(space *Space) {
	if space == nil {
		return
	}
	if space.ClusterSize == 0 {
		space.ClusterSize = fs.ClusterSize
	} else {
		if space.ClusterSize != fs.ClusterSize {
			panic(fmt.Errorf("%v !=%v", space.ClusterSize, fs.ClusterSize))
		}
	}
	for _, b := range space.FullBlocks() {
		fs.MarkOccupy(b, false)
	}
}

func NewMimcCore(path string, clusterSize int64) (*MimcCore, error) {
	return openMimcCore(path, clusterSize, false)
}

// NewReadOnlyMimcCore opens an existing file without creating or modifying it.
func NewReadOnlyMimcCore(path string, clusterSize int64) (*MimcCore, error) {
	return openMimcCore(path, clusterSize, true)
}

func openMimcCore(path string, clusterSize int64, readOnly bool) (*MimcCore, error) {
	flags := os.O_CREATE | os.O_RDWR
	if readOnly {
		flags = os.O_RDONLY
	}
	fp, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return nil, err
	}
	mfs := &MimcCore{fp: fp, ClusterSize: clusterSize, readOnly: readOnly}
	if err := mfs.init(); err != nil {
		fp.Close()
		return nil, err
	}
	return mfs, nil
}
