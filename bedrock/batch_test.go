package bedrock

import (
	"bytes"
	"testing"

	"celler"
	nbt "neomega-nbt/presets/v1"
)

// 编码结果必须独占数据；批量新建和普通替换都要保留非区块键，并正确处理实体。
func TestEncodedChunksAndReplacement(t *testing.T) {
	world, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer world.Close()
	if err := world.DB.Put([]byte("custom-world-data"), []byte("kept"), nil); err != nil {
		t.Fatal(err)
	}
	var encoded []*EncodedChunk
	for x := range int32(3) {
		chunk := NewChunk(celler.ChunkPos{X: x}, 0)
		chunk.Records[string([]byte{TagVersion})] = []byte{42}
		chunk.Records[string([]byte{TagSubChunk, 0})] = []byte{1, 2, 3}
		actor := Actor{ID: [8]byte{byte(x + 1)}, NBT: nbt.Document{Value: nbt.Compound{"identifier": "minecraft:pig", "UniqueID": int64(x + 1)}}}
		result, err := EncodeChunk(chunk, []Actor{actor})
		if err != nil {
			t.Fatal(err)
		}
		encoded = append(encoded, result)
		chunk.Records[string([]byte{TagVersion})][0] = 99
		actor.NBT.Value["identifier"] = "changed"
	}
	if err := world.WriteChunks(encoded, WriteChunksOptions{NewWorld: true}); err != nil {
		t.Fatal(err)
	}
	for x := range int32(3) {
		pos := celler.ChunkPos{X: x}
		chunk, err := world.ReadChunk(0, pos)
		if err != nil || !bytes.Equal(chunk.Records[string([]byte{TagVersion})], []byte{42}) {
			t.Fatal(chunk, err)
		}
		actors, err := world.ReadChunkActors(0, pos)
		if err != nil || len(actors) != 1 || actors[0].Value["identifier"] != "minecraft:pig" {
			t.Fatal(actors, err)
		}
	}
	replacement := NewChunk(celler.ChunkPos{}, 0)
	replacement.Records[string([]byte{TagVersion})] = []byte{43}
	record, err := EncodeChunk(replacement, []Actor{})
	if err != nil {
		t.Fatal(err)
	}
	if err := world.WriteChunks([]*EncodedChunk{record}, WriteChunksOptions{}); err != nil {
		t.Fatal(err)
	}
	chunk, err := world.ReadChunk(0, celler.ChunkPos{})
	if err != nil || len(chunk.Records) != 1 || chunk.Records[string([]byte{TagVersion})][0] != 43 {
		t.Fatal(chunk, err)
	}
	actors, err := world.ReadChunkActors(0, celler.ChunkPos{})
	if err != nil || len(actors) != 0 {
		t.Fatal(actors, err)
	}
	data, err := world.DB.Get([]byte("custom-world-data"), nil)
	if err != nil || string(data) != "kept" {
		t.Fatal(data, err)
	}
}
