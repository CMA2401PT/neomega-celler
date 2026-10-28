package bedrock

import (
	"bytes"
	"celler"
	"celler/internal/nbtutil"
	"celler/layer_storage"
	"fmt"
	"math"

	nbt "neomega-nbt/presets/v1"

	"github.com/df-mc/goleveldb/leveldb"
)

type chunkSnbtWithChangeMark struct {
	Snbts  map[celler.ChunkBlockPos]celler.SNBT
	Change bool
}

func (c *chunkSnbtWithChangeMark) GetSubChunkSNBTs(y celler.ChunkUnit) func(func(celler.SubChunkBlockPos, celler.SNBT) bool) {
	if c == nil {
		return func(func(celler.SubChunkBlockPos, celler.SNBT) bool) {}
	}
	return func(cb func(celler.SubChunkBlockPos, celler.SNBT) (next bool)) {
		for p, s := range c.Snbts {
			if !y.InRange(int(p.Y)) {
				continue
			}
			if !cb(celler.SubChunkBlockPos{X: p.X, Y: uint8(p.Y & 0xf), Z: p.Z}, s) {
				return
			}
		}
	}
}

func (c *chunkSnbtWithChangeMark) Drop(y celler.ChunkUnit) {
	if c == nil {
		return
	}
	toDrop := []celler.ChunkBlockPos{}
	for p := range c.Snbts {
		if !y.InRange(int(p.Y)) {
			continue
		}
		toDrop = append(toDrop, p)
	}
	for _, p := range toDrop {
		delete(c.Snbts, p)
	}
	c.Change = c.Change || len(toDrop) != 0
}

func (c *chunkSnbtWithChangeMark) Update(y celler.ChunkUnit, snbts *layer_storage.Map[celler.SNBT]) {
	toDrop := []celler.ChunkBlockPos{}
	for cp := range c.Snbts {
		if !y.InRange(int(cp.Y)) {
			continue
		}
		ip := celler.SubChunkBlockPos{X: cp.X, Y: uint8(cp.Y & 0xf), Z: cp.Z}
		if snbts.Get(ip) == "" {
			toDrop = append(toDrop, cp)
		}
	}
	for _, p := range toDrop {
		delete(c.Snbts, p)
	}
	c.Change = c.Change || len(toDrop) != 0
	for ip, data := range snbts.IterNonDefault {
		cp := celler.ChunkBlockPos{X: ip.X, Y: int16(ip.Y) | int16(y)<<4, Z: ip.Z}
		if c.Snbts[cp] != data {
			c.Change = true
			c.Snbts[cp] = data
		}
	}
}

func newChunkSnbtWithChangeMark() *chunkSnbtWithChangeMark {
	return &chunkSnbtWithChangeMark{
		Snbts:  make(map[celler.ChunkBlockPos]celler.SNBT),
		Change: false,
	}
}

func (p *BedrockStorageReadOnly[T]) loadChunkNBT(position celler.ChunkPos) (chunkSnbts *chunkSnbtWithChangeMark) {
	v, ok := p.SNBTCache.Get(position)
	if ok {
		return v
	}
	defer func() {
		p.SNBTCache.Add(position, chunkSnbts)
	}()

	key := p.index(position)
	data, err := p.ReadOnlyDB.Get(append(key, keyBlockEntities))

	if err == leveldb.ErrNotFound {
		return nil
	}
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	chunkSnbts = newChunkSnbtWithChangeMark()
	buf := bytes.NewBuffer(data)

	for buf.Len() != 0 {
		blockNbt, err := nbtutil.Read(buf, nbt.EncodingLittleEndian)
		if err != nil {
			return chunkSnbts
		}

		x := nbtutil.Coordinate(blockNbt, "x", math.MaxInt)
		y := nbtutil.Coordinate(blockNbt, "y", math.MaxInt16)
		z := nbtutil.Coordinate(blockNbt, "z", math.MaxInt)
		if !position.X.InRange(x) || !position.Z.InRange(z) || y > math.MaxInt16 || y < math.MinInt16 {
			fmt.Printf("block nbt %v @ %v %v %v -> out of chunk range\n", blockNbt, x, y, z)
			continue
		}
		delete(blockNbt, "x")
		delete(blockNbt, "y")
		delete(blockNbt, "z")
		snbt, err := nbtutil.ToSNBT(blockNbt)
		if err != nil {
			fmt.Printf("encode %v @ %v %v %v -> snbt err: %v\n", x, y, z, blockNbt, err)
			continue
		}
		chunkSnbts.Snbts[celler.ChunkBlockPos{
			X: uint8(x & 0xf), Z: uint8(z & 0xf), Y: int16(y),
		}] = celler.SNBT(snbt)
	}
	return chunkSnbts
}

func (p *BedrockStorageReadOnly[T]) loadSubChunkNBT(position celler.SubChunkPos) (
	snbts *layer_storage.Map[celler.SNBT],
) {
	chunkSNBTs := p.loadChunkNBT(position.ChunkPos)
	snbts = layer_storage.New(celler.SNBT("")).Map()
	for cp, snbt := range chunkSNBTs.GetSubChunkSNBTs(position.Y) {
		snbts.Set(cp, snbt)
	}
	return snbts
}

func (p *BedrockStorageReadWrite[T]) updateChunkNBT(position celler.ChunkPos, chunkSnbts *chunkSnbtWithChangeMark) {
	key := p.index(position)
	if chunkSnbts == nil {
		p.RWDB.Delete(append(key, keyBlockEntities))
		return
	}
	if !chunkSnbts.Change {
		return
	}
	defer func() { chunkSnbts.Change = false }()
	if len(chunkSnbts.Snbts) == 0 {
		p.RWDB.Delete(append(key, keyBlockEntities))
		return
	}

	buf := bytes.NewBuffer(nil)
	for bp, snbt := range chunkSnbts.Snbts {
		X := int32(position.X)<<4 | int32(bp.X&0xf)
		Z := int32(position.Z)<<4 | int32(bp.Z&0xf)
		Y := int32(bp.Y)
		blockNBT, err := nbtutil.FromSNBT(string(snbt))
		if err != nil {
			fmt.Printf("snbt decode %v @ %v %v %v -> nbt err: %v\n", string(snbt), X, Y, Z, err)
			continue
		}
		if blockNBT == nil {
			fmt.Printf(" %v @ %v %v %v is not map[string]any\n", string(snbt), X, Y, Z)
			continue
		}
		blockNBT["x"] = X
		blockNBT["y"] = Y
		blockNBT["z"] = Z
		raw, err := nbtutil.Encode(blockNBT, nbt.EncodingLittleEndian)
		if err != nil {
			fmt.Printf("encode %v @ %v %v %v -> nbt err: %v\n", blockNBT, X, Y, Z, err)
			continue
		}
		buf.Write(raw)
	}
	p.RWDB.Put(append(key, keyBlockEntities), buf.Bytes())
}

func (p *BedrockStorageReadWrite[T]) saveSubChunkNBT(position celler.SubChunkPos, snbts *layer_storage.Map[celler.SNBT]) {
	chunkSNBTs := p.loadChunkNBT(position.ChunkPos)
	if chunkSNBTs == nil {
		chunkSNBTs = newChunkSnbtWithChangeMark()
		p.SNBTCache.Add(position.ChunkPos, chunkSNBTs)
	}
	chunkSNBTs.Update(position.Y, snbts)
	p.updateChunkNBT(position.ChunkPos, chunkSNBTs)
}
