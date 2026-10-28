package bedrock

import (
	"celler/internal/nbtutil"
	"fmt"
	"neomega-blocks/ir"
	blocks "neomega-blocks/presets/block-converts/v1"
	nbt "neomega-nbt/presets/v1"
	"sync"
)

var decodeCache sync.Map // map[string]uint32

// DefaultBlockDecode returns the canonical Bedrock block hash used by presets/v1.
func DefaultBlockDecode(rawNBT []byte) (uint32, error) {
	key := string(rawNBT)
	if value, ok := decodeCache.Load(key); ok {
		return value.(uint32), nil
	}
	value, err := nbtutil.Decode(rawNBT, nbt.EncodingLittleEndian)
	if err != nil {
		return 0, err
	}
	name, ok := value["name"].(string)
	if !ok {
		return 0, fmt.Errorf("block NBT is missing its name")
	}
	states, _ := value["states"].(map[string]any)
	block, err := ir.NewIRBlockFromGo(name, states)
	if err != nil {
		return 0, err
	}
	canonical, ok := blocks.ConvertFromBedrock(block)
	if !ok {
		return 0, fmt.Errorf("unknown Bedrock block: %s", block)
	}
	id := canonical.Hash()
	decodeCache.Store(key, id)
	return id, nil
}
