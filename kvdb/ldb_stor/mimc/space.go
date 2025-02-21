package mimc

import (
	"fmt"
)

type Space struct {
	Clusters    []int64
	Offset      int64
	WritedLen   int64
	ClusterSize int64
	changeCB    func(*Space)
}

func (s *Space) SetChangeHook(changeCB func(*Space)) {
	s.changeCB = changeCB
}

func (s *Space) Flush() {
	if s.changeCB != nil {
		s.changeCB(s)
	}
}

func (s *Space) String() string {
	return fmt.Sprintf("cap: %v consumed: %v spare: %v, clusters: %v, start: %v", s.Cap(), s.WritedLen, s.Cap()-s.WritedLen, s.Clusters, s.Offset)
}

func (s *Space) Clone() *Space {
	return &Space{
		Clusters:    s.Clusters,
		Offset:      s.Offset,
		WritedLen:   s.WritedLen,
		ClusterSize: s.ClusterSize,
	}
}

func WriteVarInt64(w func(b byte) error, x int64) (err error) {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	for ux >= 0x80 {
		if err = w(byte(ux) | 0x80); err != nil {
			return err
		}
		ux >>= 7
	}
	return w(byte(ux))
}

func ReadVarInt64(r func() (byte, error)) (int64, error) {
	var ux uint64
	for i := uint(0); i < 70; i += 7 {
		b, err := r()
		if err != nil {
			return 0, err
		}
		ux |= uint64(b&0x7f) << i
		if b&0x80 == 0 {
			x := int64(ux >> 1)
			if ux&1 != 0 {
				x = ^x
			}
			return x, nil
		}
	}
	return 0, fmt.Errorf("invalid varint")
}

func (s *Space) Dump(w func(b byte) error) error {
	if s == nil {
		if err := w('E'); err != nil {
			return err
		}
		return nil
	}
	if err := w('S'); err != nil {
		return err
	}
	wi64 := func(i int64) error {
		return WriteVarInt64(w, i)
	}
	err := wi64(int64(len(s.Clusters)))
	if err != nil {
		return err
	}
	err = wi64(s.ClusterSize)
	if err != nil {
		return err
	}
	err = wi64(s.Offset)
	if err != nil {
		return err
	}
	err = wi64(s.WritedLen)
	if err != nil {
		return err
	}
	for _, v := range s.Clusters {
		err = wi64(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadSpace(r func() (byte, error)) (*Space, error) {
	ri64 := func() (int64, error) {
		return ReadVarInt64(r)
	}
	v, _ := r()
	if v == 'E' {
		return nil, nil
	}
	if v != 'S' {
		panic("invalid data")
	}
	cl, err := ri64()
	if err != nil {
		return nil, err
	}
	s := &Space{
		Clusters: make([]int64, 0),
	}
	s.ClusterSize, err = ri64()
	if err != nil {
		return nil, err
	}
	s.Offset, err = ri64()
	if err != nil {
		return nil, err
	}
	s.WritedLen, err = ri64()
	if err != nil {
		return nil, err
	}
	for ; cl > 0; cl-- {
		v, err := ri64()
		if err != nil {
			return nil, err
		}
		s.Clusters = append(s.Clusters, v)
	}
	return s, nil
}

func (s *Space) FullBlocks() []int64 {
	unFull := ((s.Offset + s.ClusterSize - 1) / s.ClusterSize)
	return s.Clusters[unFull:]
}

// how many bytes can this space hold
func (s *Space) Cap() int64 {
	return int64(len(s.Clusters))*s.ClusterSize - s.Offset
}

// how many bytes dose write accessed
func (s *Space) Used() int64 {
	return s.WritedLen
}

func (s *Space) Size() int64 {
	return s.WritedLen
}

func (s *Space) UpdateLen(dataLen int64) {
	if dataLen > s.WritedLen {
		s.WritedLen = dataLen
		if s.WritedLen > s.Cap() {
			panic(fmt.Errorf("BUG: %v > %v", s.WritedLen, s.Cap()))
		}
		if s.changeCB != nil {
			s.changeCB(s)
		}
	}
}

func (s *Space) AppendCluster(clusters []int64) {
	s.Clusters = append(s.Clusters, clusters...)
	if s.changeCB != nil {
		s.changeCB(s)
	}
}

// check if want write n byte in this space, how may extra space req
func (s *Space) NeedMore(dataEnd int64) (count int) {
	if dataEnd > s.Cap() {
		totalReq := ((dataEnd + s.Offset + s.ClusterSize - 1) / s.ClusterSize)
		return int(totalReq) - len(s.Clusters)
	}
	return 0
}

func (s *Space) overRead(dataEnd int64) (back int64) {
	if dataEnd > s.WritedLen {
		return dataEnd - s.WritedLen
	}
	return 0
}

func (s *Space) TargetOffset(dataStart, dataEnd int64) (
	clusterI int64,
	offset int64,
	canOperateSpace int64,
) {
	if dataEnd > s.Cap() {
		panic(fmt.Errorf("BUG: need check space before use: %v, %v, %v", dataEnd, s.Offset, s.Clusters))
	}
	clusterI = s.Clusters[((dataStart + s.Offset) / s.ClusterSize)]
	offset = (dataStart + s.Offset) % s.ClusterSize
	canOperateSpace = s.ClusterSize - offset
	if canOperateSpace > dataEnd-dataStart {
		canOperateSpace = dataEnd - dataStart
	}
	return
}
