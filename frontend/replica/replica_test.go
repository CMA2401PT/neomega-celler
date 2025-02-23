package replica

import (
	"fmt"
	"testing"
)

func TestNewReplicaWriter(t *testing.T) {
	writer := NewReplicaWriter()
	if writer == nil {
		t.Fatal("Expected non-nil ReplicaWriter")
	}

	// Check initial state of the writer
	if writer.Cx != 0 || writer.Cy != 0 || writer.Cz != 0 {
		t.Errorf("Expected initial Cx, Cy, Cz to be 0, got Cx: %d, Cy: %d, Cz: %d", writer.Cx, writer.Cy, writer.Cz)
	}
	if len(writer.cache) != CacheSize {
		t.Errorf("Expected cache size to be %d, got %d", CacheSize, len(writer.cache))
	}
}

func TestReplicaWriter_SetCmd(t *testing.T) {
	writer := NewReplicaWriter()

	// Set initial values for Cx, Cy, Cz
	writer.Cx = 10
	writer.Cy = 20
	writer.Cz = 30

	// First call to SetCmd - should set a short command
	writer.SetCmd(10, 20, 30, 1)
	if writer.Cmd == 0 {
		t.Fatal("Expected non-zero Cmd after SetCmd")
	}

	// Test handling of fast repeat
	initialCmd := writer.Cmd
	writer.SetCmd(10, 20, 30, 1)
	if writer.Cmd == initialCmd {
		t.Fatal("Expected Cmd to change after fast repeat")
	}
}

func TestReplicaWriter_Close(t *testing.T) {
	writer := NewReplicaWriter()

	// Simulate some writes
	writer.SetCmd(10, 20, 30, 1)
	writer.SetCmd(11, 21, 31, 2)

	// Close the writer and get the result
	result := writer.Close()

	// Verify that the result is not empty
	if len(result) == 0 {
		t.Fatal("Expected non-empty result from Close")
	}
}

func TestReplicaReader_IterCmds(t *testing.T) {
	// Simulate some commands in the writer
	writer := NewReplicaWriter()
	writer.SetCmd(10, 20, 30, 1)
	writer.SetCmd(11, 21, 31, 2)
	data := writer.Close()

	// Create a new reader from the written data
	reader := NewReplicaReader(data)

	// Iterate through the commands and check the position and block id
	expectedPositions := []P{{10, 20, 30}, {11, 21, 31}}
	expectedBlockIds := []uint16{1, 2}

	var idx int
	reader.IterCmds(func(p P, blkId uint16) bool {
		if p != expectedPositions[idx] || blkId != expectedBlockIds[idx] {
			t.Errorf("Unexpected command at index %d: got P{X:%d, Y:%d, Z:%d}, blkId: %d, expected P{X:%d, Y:%d, Z:%d}, blkId: %d",
				idx, p.X, p.Y, p.Z, blkId, expectedPositions[idx].X, expectedPositions[idx].Y, expectedPositions[idx].Z, expectedBlockIds[idx])
		}
		idx++
		return true
	})

	// Ensure that we have processed all commands
	if idx != len(expectedPositions) {
		t.Errorf("Expected to process %d commands, but processed %d", len(expectedPositions), idx)
	}
}

func TestReplicaReader_MoveAndSetCmds(t *testing.T) {
	// Simulate a sequence of move commands followed by block setting
	writer := NewReplicaWriter()
	writer.SetCmd(1, 2, 3, 10) // move
	writer.SetCmd(2, 3, 4, 20) // move
	writer.SetCmd(1, 2, 3, 30) // set block
	data := writer.Close()

	// Create a new reader from the written data
	reader := NewReplicaReader(data)

	// Expected positions and block IDs
	expectedCommands := []struct {
		p     P
		blkId uint16
	}{
		{P{1, 2, 3}, 10}, // move to 1,2,3
		{P{2, 3, 4}, 20}, // move to 2,3,4
		{P{1, 2, 3}, 30}, // set block at 1,2,3 with block id 30
	}

	var idx int
	reader.IterCmds(func(p P, blkId uint16) bool {
		if p != expectedCommands[idx].p || blkId != expectedCommands[idx].blkId {
			t.Errorf("Unexpected command at index %d: got P{X:%d, Y:%d, Z:%d}, blkId: %d, expected P{X:%d, Y:%d, Z:%d}, blkId: %d",
				idx, p.X, p.Y, p.Z, blkId, expectedCommands[idx].p.X, expectedCommands[idx].p.Y, expectedCommands[idx].p.Z, expectedCommands[idx].blkId)
		}
		idx++
		return true
	})

	// Ensure that we have processed all commands
	if idx != len(expectedCommands) {
		t.Errorf("Expected to process %d commands, but processed %d", len(expectedCommands), idx)
	}
}

func TestReplicaWriter_ComplexCommandSequence(t *testing.T) {
	writer := NewReplicaWriter()

	// 构造复杂的命令序列

	// 第一组命令：移动命令
	writer.SetCmd(1, 2, 3, 10) // move
	writer.SetCmd(2, 3, 4, 20) // move
	writer.SetCmd(3, 4, 5, 20) // move
	writer.SetCmd(4, 5, 6, 20) // move

	// 第二组命令：设置块命令，使用重复命令
	writer.SetCmd(5, 6, 7, 50) // set block with block id 50
	writer.SetCmd(6, 7, 8, 60) // set block with block id 60
	writer.SetCmd(7, 8, 9, 70) // set block with block id 70
	writer.SetCmd(7, 8, 9, 70) // set block with block id 70 (repeat)

	// 第三组命令：移动命令
	writer.SetCmd(8, 9, 10, 80)  // move
	writer.SetCmd(9, 10, 11, 90) // move

	// 第四组命令：快速重复命令
	writer.SetCmd(10, 11, 12, 100) // set block with block id 100
	writer.SetCmd(10, 11, 12, 100) // set block with block id 100 (repeat)
	writer.SetCmd(10, 11, 12, 100) // set block with block id 100 (repeat)

	// 第五组命令：设置块并移动
	writer.SetCmd(11, 12, 13, 110) // set block with block id 110
	writer.SetCmd(12, 13, 14, 120) // move
	writer.SetCmd(13, 14, 15, 130) // set block with block id 130

	// 获取写入的数据
	data := writer.Close()
	fmt.Println(len(data))

	// 创建一个新的 ReplicaReader 来读取刚才写入的数据
	reader := NewReplicaReader(data)

	// 验证读出的命令顺序和内容
	expectedCommands := []struct {
		p     P
		blkId uint16
	}{
		// 第一组移动命令
		{P{1, 2, 3}, 10},
		{P{2, 3, 4}, 20},
		{P{3, 4, 5}, 20},
		{P{4, 5, 6}, 20},

		// 第二组设置块命令和重复命令
		{P{5, 6, 7}, 50},
		{P{6, 7, 8}, 60},
		{P{7, 8, 9}, 70},
		{P{7, 8, 9}, 70}, // 重复命令

		// 第三组移动命令
		{P{8, 9, 10}, 80},
		{P{9, 10, 11}, 90},

		// 第四组快速重复命令
		{P{10, 11, 12}, 100},
		{P{10, 11, 12}, 100}, // 重复命令
		{P{10, 11, 12}, 100}, // 重复命令

		// 第五组设置块和移动命令
		{P{11, 12, 13}, 110},
		{P{12, 13, 14}, 120},
		{P{13, 14, 15}, 130},
	}

	var idx int
	reader.IterCmds(func(p P, blkId uint16) bool {
		if p != expectedCommands[idx].p || blkId != expectedCommands[idx].blkId {
			t.Errorf("Unexpected command at index %d: got P{X:%d, Y:%d, Z:%d}, blkId: %d, expected P{X:%d, Y:%d, Z:%d}, blkId: %d",
				idx, p.X, p.Y, p.Z, blkId, expectedCommands[idx].p.X, expectedCommands[idx].p.Y, expectedCommands[idx].p.Z, expectedCommands[idx].blkId)
		}
		idx++
		return true
	})

	// 确保所有命令都被正确读取
	if idx != len(expectedCommands) {
		t.Errorf("Expected to process %d commands, but processed %d", len(expectedCommands), idx)
	}
}

func TestReplicaWriter_ComplexCommandSequenceWithDump(t *testing.T) {
	writer := NewReplicaWriter()

	cmds := []struct {
		p     P
		blkId uint16
	}{
		// 第一组移动命令
		{P{1, 2, 3}, 10},
		{P{2, 3, 4}, 20},
		{P{3, 4, 5}, 20},
		{P{4, 5, 6}, 20},

		// 第二组设置块命令和重复命令
		{P{5, 6, 7}, 50},
		{P{6, 7, 8}, 60},
		{P{7, 8, 9}, 70},
		{P{7, 8, 9}, 70}, // 重复命令

		// 第三组移动命令
		{P{8, 9, 10}, 80},
		{P{9, 10, 11}, 90},

		// 第三组移动命令
		{P{208, 209, 210}, 80},
		{P{209, 210, 211}, 90},

		// 第四组快速重复命令
		{P{10, 11, 12}, 100},
		{P{10, 11, 12}, 100}, // 重复命令
		{P{10, 11, 12}, 100}, // 重复命令

		// 第五组设置块和移动命令
		{P{11, 12, 13}, 110},
		{P{12, 13, 14}, 120},
		{P{13, 14, 15}, 130},
	}

	// 验证读出的命令顺序和内容

	expectValue := map[P]uint16{}
	for _, c := range cmds {
		expectValue[P{X: c.p.X, Y: c.p.Y, Z: c.p.Z}] = c.blkId
		writer.SetCmd(c.p.X, c.p.Y, c.p.Z, c.blkId)
	}
	// 获取写入的数据
	data := writer.Close()
	fmt.Println(len(data))

	// 创建一个新的 ReplicaReader 来读取刚才写入的数据
	reader := NewReplicaReader(data)

	var idx int

	rec := reader.Dump()
	for cp, iter := range rec.Iter {
		for icp, blkId := range iter {
			p := P{X: cp.X<<4 | icp.X, Y: cp.Y<<4 | icp.Y, Z: cp.Z<<4 | icp.Z}
			if blkId != expectValue[p] {
				t.Errorf("Unexpected blkID %v at %v, expected: %d", blkId, p, expectValue[p])
			} else {
				idx++
			}
		}
	}

	// 确保所有命令都被正确读取
	if idx != len(expectValue) {
		t.Errorf("Expected to process %d commands, but processed %d", len(expectValue), idx)
	}
}
