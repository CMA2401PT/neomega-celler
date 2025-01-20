package low_io

import (
	"bytes"
	"celler"
	"celler/storage_io"
	"io"
	"sort"
)

type SimpleMemoryIO struct {
	Versions     map[celler.SubChunkPos]celler.ChunkVersion
	Datas        map[celler.SubChunkPos][]byte
	changed      bool
	SharedBuffer *bytes.Buffer
}

func NewSimpleMemoryIO() *SimpleMemoryIO {
	return &SimpleMemoryIO{
		Versions: make(map[celler.SubChunkPos]celler.ChunkVersion),
		Datas:    make(map[celler.SubChunkPos][]byte),
		changed:  false,
	}
}

func (io *SimpleMemoryIO) ClearChangeFlag() {
	io.changed = false
}

func (io *SimpleMemoryIO) Close() error { return nil }

func (io *SimpleMemoryIO) CurrentVersion() map[celler.SubChunkPos]celler.ChunkVersion {
	return io.Versions
}

type readCloserWrapper struct {
	io.Reader
}

func (readCloserWrapper) Close() error { return nil }

func (io *SimpleMemoryIO) GetReader(p celler.SubChunkPos) io.ReadCloser {
	bs := io.Datas[p]
	return readCloserWrapper{bytes.NewReader(bs)}
}

func (io *SimpleMemoryIO) Drop(p celler.SubChunkPos) {
	io.changed = true
	delete(io.Versions, p)
	delete(io.Datas, p)
}

type writeReciver struct {
	*bytes.Buffer
	onClose func(data []byte)
}

func (r *writeReciver) Close() error {
	r.onClose(r.Bytes())
	return nil
}

func (io *SimpleMemoryIO) GetWriter(p celler.SubChunkPos, v celler.ChunkVersion) io.WriteCloser {
	if io.SharedBuffer == nil {
		io.SharedBuffer = bytes.NewBuffer(nil)
	}
	io.SharedBuffer.Reset()
	return &writeReciver{
		io.SharedBuffer,
		func(data []byte) {
			io.changed = true
			io.Versions[p] = v
			io.Datas[p] = bytes.Clone(data)
		},
	}
}

type SimpleMemoryIOToBytes struct {
	SimpleMemoryIO
	genBytes []byte
	onClose  func(data []byte, change bool)
}

func (sio *SimpleMemoryIOToBytes) Close() error {
	if !sio.changed {
		if sio.onClose != nil {
			sio.onClose(sio.genBytes, false)
		}

		return nil
	}
	buf := sio.SharedBuffer
	buf.Reset()
	keyArr := make([]celler.SubChunkPos, 0, len(sio.Versions))
	for k := range sio.Versions {
		keyArr = append(keyArr, k)
	}
	sort.Slice(keyArr, func(i, j int) bool {
		return keyArr[i].Less(keyArr[j])
	})
	versionArr := make([]celler.ChunkVersion, 0, len(sio.Versions))
	dataArr := make([][]byte, 0, len(sio.Versions))
	for _, k := range keyArr {
		versionArr = append(versionArr, sio.Versions[k])
	}
	for _, k := range keyArr {
		dataArr = append(dataArr, sio.Datas[k])
	}
	storage_io.WriteArrayContigious(buf, keyArr)
	storage_io.WriteArrayWithoutCount(buf, versionArr)
	storage_io.WriteArrayWithFnWithoutCount(buf, dataArr, func(sw io.Writer, sd []byte) {
		storage_io.WriteArray(sw, sd)
	})
	sio.genBytes = buf.Bytes()
	sio.ClearChangeFlag()
	if sio.onClose != nil {
		sio.onClose(sio.genBytes, true)
	}
	return nil
}

func (sio *SimpleMemoryIOToBytes) Bytes() []byte {
	return sio.genBytes
}

func NewSimpleMemoryIOToBytes(data []byte) *SimpleMemoryIOToBytes {
	s := &SimpleMemoryIOToBytes{
		SimpleMemoryIO: *NewSimpleMemoryIO(),
		genBytes:       data,
	}
	s.SharedBuffer = bytes.NewBuffer(nil)

	if len(data) != 0 {
		buf := bytes.NewReader(data)
		keyArr := storage_io.ReadArrayContigious[celler.SubChunkPos](buf)
		versionArr := storage_io.ReadArrayWithoutCount[celler.ChunkVersion](buf, len(keyArr))
		dataArr := storage_io.ReadArrayWithFnWithoutCount(buf, func(sr io.Reader) []byte {
			return storage_io.ReadArray[byte](sr)
		}, len(keyArr))
		for i, k := range keyArr {
			s.Versions[k] = versionArr[i]
		}
		for i, k := range keyArr {
			s.Datas[k] = dataArr[i]
		}
	}
	return s
}
