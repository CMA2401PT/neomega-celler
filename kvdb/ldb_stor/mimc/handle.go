package mimc

import (
	"fmt"
	"io"
)

type ReadHandle struct {
	Space         *Space
	p             int64
	fs            *MimcCore
	CloseCallback func() error
}

func (rh *ReadHandle) Close() error {
	if rh.CloseCallback != nil {
		return rh.CloseCallback()
	}
	return nil
}

func (rh *ReadHandle) Seek(offset int64, whence int) (int64, error) {
	b := int64(0)
	if whence == 2 {
		b = rh.Space.Size()
	} else if whence == 1 {
		b = rh.p
	}
	rh.p = b + offset
	return rh.p, nil
}

func (rh *ReadHandle) Read(p []byte) (n int, err error) {
	pStart := int64(0)
	pEnd := int64(len(p))
	dataEnd := rh.p + pEnd
	if back := rh.Space.overRead(dataEnd); back != 0 {
		pEnd -= back
		dataEnd -= back
		err = io.EOF
	}
	n = int(pEnd)

	if dataEnd <= rh.p {
		return 0, io.EOF
	}

	if rh.Space.NeedMore(dataEnd) != 0 {
		panic(fmt.Errorf("%v > %v", dataEnd, rh.Space.Cap()))
	}

	for pStart != pEnd {
		ci, coff, readSpace := rh.Space.TargetOffset(rh.p, dataEnd)
		rh.fs.ReadSingleCluster(ci, coff, p[pStart:pStart+readSpace])
		pStart += readSpace
		rh.p += readSpace
	}
	return
}

func (rh *ReadHandle) ReadAt(p []byte, off int64) (n int, err error) {
	ptr := off
	pStart := int64(0)
	pEnd := int64(len(p))
	dataEnd := ptr + pEnd

	if back := rh.Space.overRead(dataEnd); back != 0 {
		pEnd -= back
		dataEnd -= back
		err = io.EOF
	}
	n = int(pEnd)

	if dataEnd <= ptr {
		return 0, io.EOF
	}

	if rh.Space.NeedMore(dataEnd) != 0 {
		panic(fmt.Errorf("%v > %v", dataEnd, rh.Space.Cap()))
	}

	for pStart != pEnd {
		ci, coff, readSpace := rh.Space.TargetOffset(ptr, dataEnd)
		rh.fs.ReadSingleCluster(ci, coff, p[pStart:pStart+readSpace])
		pStart += readSpace
		ptr += readSpace
	}
	return
}

type WriteHandle struct {
	Space         *Space
	p             int64
	fs            *MimcCore
	CloseCallback func() error
}

func (rh *WriteHandle) Close() error {
	if rh.CloseCallback != nil {
		return rh.CloseCallback()
	}
	return nil
}

func (rh *WriteHandle) Seek(offset int64, whence int) (int64, error) {
	b := int64(0)
	if whence == 2 {
		b = rh.Space.Size()
	} else if whence == 1 {
		b = rh.p
	}
	rh.p = b + offset
	return rh.p, nil
}

func (rh *WriteHandle) Sync() error {
	return rh.fs.fp.Sync()
}

func (rh *WriteHandle) Write(p []byte) (n int, err error) {
	pStart := int64(0)
	pEnd := int64(len(p))
	dataEnd := rh.p + pEnd
	clustersAppend := rh.Space.NeedMore(dataEnd)
	if clustersAppend > 0 {
		newClusters, err := rh.fs.AllocClusters(clustersAppend)
		if err != nil {
			return 0, err
		}
		rh.Space.AppendCluster(newClusters)
		if len(newClusters) < clustersAppend {
			return 0, fmt.Errorf("cannot alloacte enough clusters")
		}
	}

	for pStart != pEnd {
		ci, coff, writeSpace := rh.Space.TargetOffset(rh.p, dataEnd)
		rh.fs.WriteSingleCluster(ci, coff, p[pStart:pStart+writeSpace])
		pStart += writeSpace
		rh.p += writeSpace
		rh.Space.UpdateLen(rh.p)
	}
	return len(p), nil
}

func (rh *WriteHandle) WriteAt(p []byte, offset int64) (n int, err error) {
	ptr := offset
	pStart := int64(0)
	pEnd := int64(len(p))
	dataEnd := ptr + pEnd
	clustersAppend := rh.Space.NeedMore(dataEnd)
	if clustersAppend > 0 {
		newClusters, err := rh.fs.AllocClusters(clustersAppend)
		if err != nil {
			return 0, err
		}
		rh.Space.AppendCluster(newClusters)
		if len(newClusters) < clustersAppend {
			return 0, fmt.Errorf("cannot alloacte enough clusters")
		}
	}

	for pStart != pEnd {
		ci, coff, writeSpace := rh.Space.TargetOffset(ptr, dataEnd)
		rh.fs.WriteSingleCluster(ci, coff, p[pStart:pStart+writeSpace])
		pStart += writeSpace
		ptr += writeSpace
		rh.Space.UpdateLen(ptr)
	}
	return len(p), nil
}
