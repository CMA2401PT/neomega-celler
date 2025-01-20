package subchunk

import (
	"celler"
	"fmt"
	"io"
)

type Versioned[
	BlockIDT ~uint32, AttachT ~uint32,
	BlockStorageT interface {
		celler.Storage[BlockIDT]
		celler.In
		celler.Out
	},
	AttachStorageT interface {
		celler.Storage[AttachT]
		celler.In
		celler.Out
	},
	SNBTStorageT interface {
		celler.Storage[celler.SNBT]
		celler.In
		celler.Out
	},
] struct {
	Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]
	versionData []celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT]
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) QueryChange(cacheVersion celler.ChunkVersion) celler.SubChunkDataChange[BlockIDT, AttachT] {
	if cacheVersion >= vs.CurrentVersion {
		// no change
		return celler.SubChunkDataChange[BlockIDT, AttachT]{}
	}
	if cacheVersion > 0 && cacheVersion >= vs.versionData[0].BeforeVersion {
		deltas := []celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT]{}
		for _, d := range vs.versionData {
			if d.BeforeVersion >= cacheVersion {
				deltas = append(deltas, d)
			}
		}
		return celler.SubChunkDataChange[BlockIDT, AttachT]{
			BlockDeltas: deltas,
		}
	}
	return celler.SubChunkDataChange[BlockIDT, AttachT]{
		Base: &vs.Simple,
	}
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) appendChange(beforeVersion celler.ChunkVersion, delta celler.SubChunkBlockDelta[BlockIDT, AttachT]) {
	vs.versionData = append(vs.versionData, celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT]{
		SubChunkBlockDelta: delta,
		BeforeVersion:      beforeVersion,
		AfterVersion:       vs.CurrentVersion,
	})
	if len(vs.versionData) > 256 {
		vs.versionData = vs.versionData[128:]
	}
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlock(delta celler.SubChunkBlockDelta[BlockIDT, AttachT]) {
	beforeVersion := vs.CurrentVersion
	vs.Simple.ChangeBlock(delta)
	vs.appendChange(beforeVersion, delta)
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlockWithVersion(delta celler.SubChunkBlockDelta[BlockIDT, AttachT], newVersion celler.ChunkVersion) {
	beforeVersion := vs.CurrentVersion
	vs.Simple.ChangeBlockWithVersion(delta, newVersion)
	vs.appendChange(beforeVersion, delta)
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ApplyChange(change celler.SubChunkDataChange[BlockIDT, AttachT]) {
	if change.Base != nil {
		if change.Base.Version() > vs.CurrentVersion {
			vs.CurrentVersion = change.Base.Version()
			vs.MetaData = map[string]string{}
			for k, v := range change.Base.Meta() {
				vs.MetaData[k] = v
			}
			vs.BlockIDStorage.CopyFrom(change.Base.BlockID())
			vs.BlockAttachStorage.CopyFrom(change.Base.BlockAttach())
			vs.BlockSNBTStorage.CopyFrom(change.Base.BlockSNBT())
			vs.versionData = make([]celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT], 0)
		}
	}
	if len(change.BlockDeltas) > 0 {
		for _, delta := range change.BlockDeltas {
			if delta.BeforeVersion < vs.CurrentVersion {
				continue
			}
			vs.ChangeBlockWithVersion(delta.SubChunkBlockDelta, delta.AfterVersion)
		}
	}
}

type changeStoreT[BlockIDT ~uint32, AttachT ~uint32] struct {
	celler.SubChunkBlockDeltaNoSNBT[BlockIDT, AttachT]
	BeforeVersion celler.ChunkVersion
	AfterVersion  celler.ChunkVersion
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Dump(w io.Writer) {
	vs.Simple.Dump(w)
	// changeNoSNBT := make([]changeStoreT[BlockIDT, AttachT], len(vs.versionData))
	// changeSNBT := make([]celler.SNBT, len(vs.versionData))
	// for i, delta := range vs.versionData {
	// 	changeNoSNBT[i] = changeStoreT[BlockIDT, AttachT]{
	// 		delta.SubChunkBlockDeltaNoSNBT,
	// 		delta.BeforeVersion,
	// 		delta.AfterVersion,
	// 	}
	// 	changeSNBT[i] = delta.SNBT
	// }
	// storage_io.WriteArrayContigious(w, changeNoSNBT)
	// storage_io.WriteArray(w, changeSNBT)
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Load(r io.Reader) {
	vs.Simple.Load(r)
	vs.versionData = nil
	// changeNoSNBT := storage_io.ReadArrayContigious[changeStoreT[BlockIDT, AttachT]](r)
	// changeSNBT := storage_io.ReadArrayContigious[celler.SNBT](r)
	// versionData := make([]celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT], len(changeSNBT))
	// for i, d := range changeNoSNBT {
	// 	versionData[i] = celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT]{
	// 		SubChunkBlockDelta: celler.SubChunkBlockDelta[BlockIDT, AttachT]{
	// 			SubChunkBlockDeltaNoSNBT: d.SubChunkBlockDeltaNoSNBT,
	// 			SNBT:                     changeSNBT[i],
	// 		},
	// 		BeforeVersion: changeNoSNBT[i].BeforeVersion,
	// 		AfterVersion:  changeNoSNBT[i].AfterVersion,
	// 	}
	// }
	// vs.versionData = versionData
}

func MakeVersionedSubChunk[
	BlockIDT ~uint32, AttachT ~uint32,
	BlockStorageT interface {
		celler.Storage[BlockIDT]
		celler.In
		celler.Out
	},
	AttachStorageT interface {
		celler.Storage[AttachT]
		celler.In
		celler.Out
	},
	SNBTStorageT interface {
		celler.Storage[celler.SNBT]
		celler.In
		celler.Out
	},
](
	CurrentVersion celler.ChunkVersion,
	MetaData map[string]string,
	BlockIDStorage BlockStorageT,
	BlockAttachStorage AttachStorageT,
	BlockSNBTStorage SNBTStorageT,
) *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT] {
	if CurrentVersion == 0 {
		panic(fmt.Errorf("version cannot be 0"))
	}
	if MetaData == nil {
		MetaData = make(map[string]string)
	}
	return &Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]{
		Simple: Simple[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]{
			CurrentVersion:     CurrentVersion,
			MetaData:           MetaData,
			BlockIDStorage:     BlockIDStorage,
			BlockAttachStorage: BlockAttachStorage,
			BlockSNBTStorage:   BlockSNBTStorage,
		},
		versionData: make([]celler.SubChunkBlockDeltaWithVersion[BlockIDT, AttachT], 0),
	}
}
