package simple

import (
	"bytes"
	"celler"
	"celler/kvdb"
	"encoding/binary"
)

type subChunk[BlockIDT ~uint32, AttachT ~uint32] interface {
	celler.SubChunk[BlockIDT, AttachT]
	celler.In
	celler.Out
}

type SimpleKV[BlockIDT ~uint32, AttachT ~uint32, Data subChunk[BlockIDT, AttachT]] struct {
	DB       kvdb.KVDB
	NewEmpty func() Data
	prefix   byte
}

func NewSimpleKV[BlockIDT ~uint32, AttachT ~uint32, Data subChunk[BlockIDT, AttachT]](
	DB kvdb.KVDB,
	NewEmpty func() Data,
	prefix byte,
) *SimpleKV[BlockIDT, AttachT, Data] {
	return &SimpleKV[BlockIDT, AttachT, Data]{
		DB:       DB,
		NewEmpty: NewEmpty,
	}
}

func (SimpleKV[BlockIDT, AttachT, Data]) flip(o []byte) {
	x0, x1, x2, x3, z0, z1, z2, z3 := o[0], o[1], o[2], o[3], o[4], o[5], o[6], o[7]
	xz0 := (z0 & 1) | (x0&1)<<1 | (z0&2)<<1 | (x0&2)<<2 | (z0&4)<<2 | (x0&4)<<3 | (z0&8)<<3 | (x0&8)<<4
	xz1 := (z0&16)>>4 | (x0&16)>>3 | (z0&32)>>3 | (x0&32)>>2 | (z0&64)>>2 | (x0&64)>>1 | (z0&128)>>1 | (x0 & 128)
	o[0], o[1], o[2], o[3], o[4], o[5], o[6], o[7] = x3, z3, x2, z2, x1, z1, xz1, xz0
}

func (SimpleKV[BlockIDT, AttachT, Data]) unflip(o []byte) {
	x3, z3, x2, z2, x1, z1, xz1, xz0 := o[0], o[1], o[2], o[3], o[4], o[5], o[6], o[7]
	z0 := (xz0 & 1) | (xz0&4)>>1 | (xz0&16)>>2 | (xz0&64)>>3 | (xz1&1)<<4 | (xz1&4)<<3 | (xz1&16)<<2 | (xz1&64)<<1
	x0 := (xz0&2)>>1 | (xz0&8)>>2 | (xz0&32)>>3 | (xz0&128)>>4 | (xz1&2)<<3 | (xz1&8)<<2 | (xz1&32)<<1 | (xz1 & 128)
	o[0], o[1], o[2], o[3], o[4], o[5], o[6], o[7] = x0, x1, x2, x3, z0, z1, z2, z3
}

// func (SimpleKV[BlockIDT, AttachT, Data]) compress(inArr []byte) []byte {
// 	if len(inArr) == 0 {
// 		return nil
// 	}
// 	last := inArr[0]
// 	repeat := uint8(0)
// 	cl := 2
// 	for _, v := range inArr {
// 		if repeat < 255 && v == last {
// 			repeat += 1
// 		} else {
// 			cl += 2
// 			last = v
// 			repeat = 1
// 		}
// 	}

// 	out := make([]byte, cl)
// 	out[0] = inArr[0]
// 	cl = 2
// 	last = inArr[0]
// 	repeat = uint8(0)
// 	for _, v := range inArr {
// 		if repeat < 255 && v == last {
// 			repeat += 1
// 		} else {
// 			out[cl-2] = last
// 			out[cl-1] = repeat
// 			cl += 2
// 			last = v
// 			repeat = 1
// 		}
// 	}
// 	out[cl-2] = last
// 	out[cl-1] = repeat

// 	if len(inArr) > 2048 && len(inArr) < 1024*3 {
// 		fmt.Println(inArr)
// 		os.Exit(0)
// 	}

// 	if len(inArr) < (1024 + 512) {
// 		rawSize += len(inArr)
// 		newSize += len(out)
// 		fmt.Println(float64(newSize) / float64(rawSize))
// 	}

// 	return out
// }

// func (SimpleKV[BlockIDT, AttachT, Data]) decompress(inArr []byte) []byte {
// 	if len(inArr) == 0 {
// 		return nil
// 	}
// 	ol := 0
// 	for i := 1; i < len(inArr); i += 2 {
// 		ol += int(inArr[i])
// 	}
// 	out := make([]byte, ol)
// 	p := 0
// 	for i := 0; i < len(inArr); i += 2 {
// 		v := inArr[i]
// 		r := inArr[i+1]
// 		for ; r > 0; r-- {
// 			out[p] = v
// 			p++
// 		}
// 	}
// 	return out
// }

func (p *SimpleKV[BlockIDT, AttachT, Data]) posKey(pos celler.SubChunkPos) []byte {
	x, z := uint32(pos.X), uint32(pos.Z)
	b := make([]byte, 10)
	b[0] = p.prefix
	binary.LittleEndian.PutUint32(b[1:], x)
	binary.LittleEndian.PutUint32(b[5:], z)
	b[9] = uint8(int8(pos.Y))
	p.flip(b[1:])
	return b
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) Get(position celler.SubChunkPos) (d Data) {
	key := p.posKey(position)
	data, err := p.DB.Get(key)
	if err != nil {
		return d
	}
	if data == nil {
		return d
	}
	reciver := p.NewEmpty()
	reciver.Load(bytes.NewReader(data))
	return reciver
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) posFromKey(key []byte) celler.SubChunkPos {
	p.unflip(key)
	x := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key[1:])))
	z := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key[5:])))
	y := celler.ChunkUnit(int8(key[9]))
	pos := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: x, Z: z}, Y: y}
	return pos
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) IterAll(cb func(celler.SubChunkPos) (next bool)) {
	for kv := range p.DB.IterAll {
		key := (kv.Key())
		if len(key) != 10 {
			continue
		}
		if key[0] != p.prefix {
			continue
		}
		pos := p.posFromKey(key)
		if !cb(pos) {
			break
		}
	}
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[BlockIDT, AttachT] {
	return p.Get(position).QueryChange(version)
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) Drop(position celler.SubChunkPos) {
	p.DB.Delete(p.posKey(position))
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) Set(position celler.SubChunkPos, data Data) {
	if data.Version() == 0 {
		p.Drop(position)
		return
	}
	buf := bytes.NewBuffer(nil)
	data.Dump(buf)
	p.DB.Put(p.posKey(position), buf.Bytes())
}

func (s *SimpleKV[BlockIDT, AttachT, Data]) GetOrEmpty(pos celler.SubChunkPos) Data {
	current := s.Get(pos)
	if current.Version() == 0 {
		current = s.NewEmpty()
	}
	return current
}

func (p *SimpleKV[BlockIDT, AttachT, Data]) ApplyChange(position celler.SubChunkPos, change celler.SubChunkChange[BlockIDT, AttachT]) {
	if change.Valid {
		return
	}
	if change.Drop {
		p.Drop(position)
	}
	init := p.GetOrEmpty(position)
	init.ApplyChange(change)
	p.Set(position, init)
}
