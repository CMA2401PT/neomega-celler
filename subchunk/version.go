package subchunk

import (
	"celler"
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
	versionData []celler.VersionedBlockChange[BlockIDT, AttachT]
}

func (sc *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) Version() celler.ChunkVersion {
	if sc == nil {
		return 0
	}
	return sc.Simple.CurrentVersion
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) QueryChange(cacheVersion celler.ChunkVersion) celler.SubChunkChange[BlockIDT, AttachT] {
	if vs == nil {
		return celler.SubChunkChange[BlockIDT, AttachT]{
			Valid: cacheVersion == 0,
			Drop:  true,
		}
	}
	if cacheVersion == vs.CurrentVersion {
		// no change
		return celler.SubChunkChange[BlockIDT, AttachT]{
			Valid: true,
		}
	}
	if cacheVersion >= vs.CurrentVersion {
		// no change
		return celler.SubChunkChange[BlockIDT, AttachT]{}
	}
	if cacheVersion > 0 && len(vs.versionData) > 0 && cacheVersion >= vs.versionData[0].BeforeVersion {
		deltas := []celler.VersionedBlockChange[BlockIDT, AttachT]{}
		for _, d := range vs.versionData {
			if d.BeforeVersion >= cacheVersion {
				deltas = append(deltas, d)
			}
		}
		return celler.SubChunkChange[BlockIDT, AttachT]{
			BlockChanges: deltas,
		}
	}
	return celler.SubChunkChange[BlockIDT, AttachT]{
		Base: &vs.Simple,
	}
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) appendChange(beforeVersion celler.ChunkVersion, delta celler.BlockChange[BlockIDT, AttachT]) {
	vs.versionData = append(vs.versionData, celler.VersionedBlockChange[BlockIDT, AttachT]{
		BlockChange:   delta,
		BeforeVersion: beforeVersion,
		AfterVersion:  vs.CurrentVersion,
	})
	if len(vs.versionData) > 256 {
		vs.versionData = vs.versionData[128:]
	}
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlock(delta celler.BlockChange[BlockIDT, AttachT]) {
	beforeVersion := vs.CurrentVersion
	vs.Simple.ChangeBlock(delta)
	vs.appendChange(beforeVersion, delta)
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ChangeBlockWithVersion(delta celler.BlockChange[BlockIDT, AttachT], newVersion celler.ChunkVersion) {
	beforeVersion := vs.CurrentVersion
	vs.Simple.ChangeBlockWithVersion(delta, newVersion)
	vs.appendChange(beforeVersion, delta)
}

func (vs *Versioned[BlockIDT, AttachT, BlockStorageT, AttachStorageT, SNBTStorageT]) ApplyChange(change celler.SubChunkChange[BlockIDT, AttachT]) {
	if change.Valid {
		return
	}
	if change.Drop {
		panic("drop must be handled by subchunk holder, not subchunk itself")
	}
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
			vs.versionData = make([]celler.VersionedBlockChange[BlockIDT, AttachT], 0)
		}
	}
	if len(change.BlockChanges) > 0 {
		for _, delta := range change.BlockChanges {
			if delta.BeforeVersion < vs.CurrentVersion {
				continue
			}
			vs.ChangeBlockWithVersion(delta.BlockChange, delta.AfterVersion)
		}
	}
}

type changeStoreT[BlockIDT ~uint32, AttachT ~uint32] struct {
	celler.BlockChange[BlockIDT, AttachT]
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
		versionData: make([]celler.VersionedBlockChange[BlockIDT, AttachT], 0),
	}
}
