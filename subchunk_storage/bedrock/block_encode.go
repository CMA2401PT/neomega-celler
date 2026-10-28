package bedrock

import (
	"celler/internal/nbtutil"
	"fmt"
	"neomega-blocks/ir"
	blocks "neomega-blocks/presets/block-converts/v1"
	nbt "neomega-nbt/presets/v1"
	"sync"
)

const chunkVersion = 27

var encodeCache sync.Map // map[uint32][]byte

// DefaultBlockEncode accepts a canonical Bedrock block hash, not a runtime ID.
// The returned bytes are cached and must be treated as read-only.
func DefaultBlockEncode(id uint32) ([]byte, error) {
	if value, ok := encodeCache.Load(id); ok {
		return value.([]byte), nil
	}
	block, ok := blocks.BedrockHashToBlockStrict(id)
	if !ok {
		air, err := DefaultBlockEncode(ir.IRBlock("minecraft:air []").Hash())
		if err != nil {
			return nil, err
		}
		return air, fmt.Errorf("unknown Bedrock block hash: %d", id)
	}
	raw, err := nbtutil.Encode(map[string]any{
		"name":   block.NameWithNameSpace(),
		"states": block.States().GoMap(),
	}, nbt.EncodingLittleEndian)
	if err != nil {
		return nil, err
	}
	encodeCache.Store(id, raw)
	return raw, nil
}
