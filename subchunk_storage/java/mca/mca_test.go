package mca

// func TestMCA(t *testing.T) {
// 	stor, err := stor.OpenDirReadOnly("../../data/region")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for rp := range stor.IterRegions {
// 		fmt.Println(rp)
// 	}

// 	mcaKV := NewReadOnlyMCA(stor)
// 	data, err := mcaKV.Get(celler.ChunkPos{1, 1})
// 	fmt.Println(len(data), err)

// 	for chunkPos := range mcaKV.IterAll {
// 		fmt.Println(chunkPos)
// 	}

// 	// rd, err := stor.OpenRegion(regionPos{1, -2})
// 	// fmt.Println(rd, err)
// 	// rr, err := newRoRegion(rd)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }
// 	// data, err := rr.GetChunkData(inRegionPos{Cx: 24, Cz: 10})
// 	// fmt.Println(data, err)
// 	// fmt.Println(rr)
// }
