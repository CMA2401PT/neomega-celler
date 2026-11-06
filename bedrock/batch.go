package bedrock

import (
	"bytes"
	"fmt"
	"io/fs"

	"celler"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/df-mc/goleveldb/leveldb/util"
	nbt "neomega-nbt/presets/v1"
)

// EncodedChunk 独立持有一根柱子的原生键值记录，可从计算任务交给数据库写入者。
// 构建后只读；不依赖传入 Chunk、Actor 或 NBT 的生命周期。
type EncodedChunk struct {
	Pos       celler.ChunkPos
	Dimension int32
	data      leveldb.Batch
}

// Size 返回编码批次字节数，用于约束等待写入的记录总量。
func (c *EncodedChunk) Size() int { return len(c.data.Dump()) }

// EncodeChunk 只编码，不访问世界。actors 为 nil 时不改实体索引；非 nil 空列表删除索引。
// 不同任务可以并行调用，但输入区块及实体 NBT 必须由各任务独占。
func EncodeChunk(chunk *Chunk, actors []Actor) (*EncodedChunk, error) {
	encoded := &EncodedChunk{Pos: chunk.Pos, Dimension: chunk.Dimension}
	prefix := ChunkPrefix(chunk.Pos, chunk.Dimension)
	for suffix, value := range chunk.Records {
		key := append(bytes.Clone(prefix), []byte(suffix)...)
		pos, dimension, parsed, ok := ParseChunkKey(key)
		if !ok || pos != chunk.Pos || dimension != chunk.Dimension || parsed != suffix {
			return nil, fmt.Errorf("invalid chunk record suffix %x", suffix)
		}
		encoded.data.Put(key, value)
	}
	if actors != nil {
		ids := make([]byte, 0, len(actors)*8)
		for _, actor := range actors {
			data, err := nbt.EncodeDocument(actor.NBT, nbt.EncodingLittleEndian)
			if err != nil {
				return nil, err
			}
			encoded.data.Put(append([]byte("actorprefix"), actor.ID[:]...), data)
			ids = append(ids, actor.ID[:]...)
		}
		key := append([]byte("digp"), prefix...)
		if len(ids) == 0 {
			encoded.data.Delete(key)
		} else {
			encoded.data.Put(key, ids)
		}
	}
	return encoded, nil
}

// WriteChunksOptions 区分新建世界的追加导入与已有区块的替换。
type WriteChunksOptions struct {
	// NewWorld 要求这些坐标尚未写入区块，因此无需扫描和删除旧记录。
	NewWorld     bool
	WriteOptions *opt.WriteOptions
}

// WriteChunks 在同一个数据库 batch 中提交区块与实体。调用者负责限制批次大小，
// 并保证每批内的坐标与维度组合不重复。
// 普通模式沿用 WriteChunk 的旧记录清理规则；NewWorld 跳过清理扫描。
// 合并 batch 会复制编码字节，提交期间须为这份副本预留内存。
func (w *World) WriteChunks(chunks []*EncodedChunk, options WriteChunksOptions) error {
	if w.closed {
		return fs.ErrClosed
	}
	if w.readOnly {
		return fs.ErrPermission
	}
	batch := new(leveldb.Batch)
	for _, chunk := range chunks {
		if !options.NewWorld {
			iter := w.DB.NewIterator(util.BytesPrefix(ChunkPrefix(chunk.Pos, chunk.Dimension)), nil)
			for iter.Next() {
				pos, dimension, _, ok := ParseChunkKey(iter.Key())
				if ok && pos == chunk.Pos && dimension == chunk.Dimension {
					batch.Delete(iter.Key())
				}
			}
			err := iter.Error()
			iter.Release()
			if err != nil {
				return err
			}
		}
		if err := chunk.data.Replay(batch); err != nil {
			return err
		}
	}
	return w.DB.Write(batch, options.WriteOptions)
}
