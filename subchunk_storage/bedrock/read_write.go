package bedrock

import (
	"celler"
	"celler/kvdb"
	"celler/layer_storage"
	"celler/subchunk"
)

type BedrockStorageReadWrite[T ~uint32] struct {
	*BedrockStorageReadOnly[T]
	RWDB                      kvdb.KVDB
	EncodeSubChunkChunkBlocks func(buf Writer, Y int32, blocks *layer_storage.Palette[T])
}

func NewReadWriteBedrockStorage[T ~uint32](
	DB kvdb.KVDB,
	Dim int32,
	DefaultBlockID T,
	BlockDecode func(nbt []byte) (T, error),
	BlockEncode func(T) ([]byte, error),
) *BedrockStorageReadWrite[T] {
	chunkEncoder := &ChunkEncoder[T]{
		BlockEncode: BlockEncode,
	}
	p := &BedrockStorageReadWrite[T]{
		BedrockStorageReadOnly:    NewReadOnlyBedrockStorage(DB, Dim, DefaultBlockID, BlockDecode),
		RWDB:                      DB,
		EncodeSubChunkChunkBlocks: chunkEncoder.EncodeSubChunkChunkBlocks,
	}
	return p
}

func (p *BedrockStorageReadWrite[T]) Drop(position celler.SubChunkPos) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := p.index(position.ChunkPos)
	p.RWDB.Delete(append(key, keySubChunkData, uint8(position.Y)))

	// keyFinalisation & keyVersion is related to the full chunk
	// p.RWDB.Delete(append(key, keyFinalisation))
	// p.RWDB.Delete(append(key, keyVersion))

	chunkNbts := p.loadChunkNBT(position.ChunkPos)
	chunkNbts.Drop(position.Y)
	p.updateChunkNBT(position.ChunkPos, chunkNbts)
}

func (p *BedrockStorageReadWrite[T]) Set(position celler.SubChunkPos, data *subchunk.Simple[
	T, uint32,
	*layer_storage.Palette[T],
	*layer_storage.PlaceHolder[uint32],
	*layer_storage.Map[celler.SNBT],
]) {
	if data.Version() == 0 {
		p.Drop(position)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	key := p.index(position.ChunkPos)
	p.RWDB.Put(append(key, keyVersion), []byte{chunkVersion})
	p.RWDB.Put(append(key, keyFinalisation), []byte{2, 0, 0, 0})
	p.saveSubChunkNBT(position, data.BlockSNBTStorage)
	p.saveSubChunkBlocks(position, data.BlockIDStorage)
}

// func (p *BedrockStorageReadOnly[T]) Get(position celler.SubChunkPos) *subchunk.Simple[
// 	T, uint32,
// 	*layer_storage.Palette[T],
// 	*layer_storage.PlaceHolder[uint32],
// 	*layer_storage.Map[celler.SNBT],
// ] {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	blocks, err := p.loadSubChunkBlocks(position)
// 	if err != nil {
// 		return nil
// 	}
// 	if blocks == nil {
// 		return nil
// 	}
// 	snbt := p.loadSubChunkNBT(position)
// 	return subchunk.MakeSimpleSubChunk(
// 		1,
// 		nil,
// 		blocks,
// 		layer_storage.MakePlaceHolder(uint32(0)),
// 		snbt,
// 	)
// }

// func (s *BedrockStorageReadOnly[T]) GetOrEmpty(pos celler.SubChunkPos) *subchunk.Simple[
// 	T, uint32,
// 	*layer_storage.Palette[T],
// 	*layer_storage.PlaceHolder[uint32],
// 	*layer_storage.Map[celler.SNBT],
// ] {
// 	current := s.Get(pos)
// 	if current.Version() == 0 {
// 		current = &subchunk.Simple[
// 			T, uint32,
// 			*layer_storage.Palette[T],
// 			*layer_storage.PlaceHolder[uint32],
// 			*layer_storage.Map[celler.SNBT]]{
// 			CurrentVersion: 0, MetaData: nil,
// 			BlockIDStorage:     layer_storage.New(s.DefaultBlock).Palette(),
// 			BlockAttachStorage: layer_storage.New(uint32(0)).PlaceHolder(),
// 			BlockSNBTStorage:   layer_storage.New(celler.SNBT("")).Map(),
// 		}
// 	}
// 	return current
// }

// func (p *BedrockStorageReadOnly[T]) IterAll(cb func(celler.SubChunkPos) (next bool)) {
// 	for kv := range p.ReadOnlyDB.IterAll {
// 		key := kv.Key()
// 		if p.Dim == 0 {
// 			if len(key) != (8+2) || key[8] != '/' {
// 				continue
// 			}
// 		} else {
// 			if len(key) != (12+2) || key[12] != '/' {
// 				continue
// 			}
// 			dm := binary.LittleEndian.Uint32(key[8:12])
// 			if dm != uint32(p.Dim) {
// 				continue
// 			}
// 		}
// 		x := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key)))
// 		z := celler.ChunkUnit(int32(binary.LittleEndian.Uint32(key[4:])))
// 		var y celler.ChunkUnit
// 		if p.Dim == 0 {
// 			y = celler.ChunkUnit(int8(key[9]))
// 		} else {
// 			y = celler.ChunkUnit(int8(key[13]))
// 		}
// 		pos := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: x, Z: z}, Y: y}
// 		if !cb(pos) {
// 			break
// 		}
// 	}
// }

// func (p *BedrockStorageReadOnly[T]) QueryChange(position celler.SubChunkPos, version celler.ChunkVersion) celler.SubChunkChange[T, uint32] {
// 	return p.Get(position).QueryChange(version)
// }
