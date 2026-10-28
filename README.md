# neomega-celler

`neomega-celler` 把 Minecraft 方块数据组织成可以读取、修改、保存和同步的子区块。子区块是一个 16×16×16 的方块区域。内存里的子区块保存方块 ID、附加值和方块实体 SNBT，存储适配器负责把它们转成 Bedrock LevelDB 或 Java MCA 中的数据。

这样，修改方块的代码可以围绕同一种内存表示编写。只有读写存档的边界需要处理不同的磁盘格式。Go 模块名仍是 `celler`，因此导入路径使用 `celler/...`。

## 方块 ID 从哪里来

子区块的基础类型允许调用者选择自己的 `~uint32` 方块 ID。默认 Bedrock 转换器和 Java 适配器使用当前 `neomega-blocks/presets/block-converts/v1` 的 **规范化 Bedrock 方块哈希**。它不是游戏 runtime ID，也不能直接当成旧版 `UIBI` 数据使用。

输入方块可能缺少属性。默认转换先通过 blocks 的 canonical 逻辑补齐属性，再获得哈希。NBT 和 SNBT 的读写使用 `neomega-nbt/presets/v1`。

## 从内存里的方块到存档

为了让业务代码只处理方块，库分成三层：

- `layer_storage` 保存一个子区块里的某一类值。`Palette` 适合大量重复方块，`Map` 适合稀疏的方块实体，`PlaceHolder` 表示整层相同。
- `subchunk` 把这些层组合起来，并用版本号描述变化。版本号为 0 表示没有数据；修改时使用 `ChangeBlock` 可以同时推进版本。
- `subchunk_storage` 负责保存子区块。Bedrock 通过 `kvdb` 访问 LevelDB；Java 通过 `subchunk_storage/java/mca` 访问区域文件。

需要传递变化时，根包的 `SubChunkChange` 描述全量或增量更新，`frontend/replica` 提供相关命令的编码与读取。

## 运行一个例子

本地开发需要 Go 1.25.5 或以上，并让 `neomega-celler`、`neomega-blocks`、`neomega-nbt` 位于同一级目录。`go.mod` 的 `replace` 指向这两个相邻仓库。使用父目录的 `go.work` 时，需要把三个模块都加入其中。

下面使用内存 LevelDB，经过 Bedrock 编解码保存并读回一个方块。把代码保存为 `examples/basic/main.go`，在仓库根目录运行 `go run ./examples/basic`。

```go
package main

import (
    "fmt"
    "log"

    "celler"
    "celler/kvdb"
    "celler/subchunk_storage/bedrock"
    "neomega-blocks/ir"
    blocks "neomega-blocks/presets/block-converts/v1"
)

func main() {
    db, err := kvdb.NewReadWriteLevelDB("")
    if err != nil { log.Fatal(err) }
    defer db.Close()

    air := ir.IRBlock("minecraft:air []").Hash()
    stone, ok := blocks.ConvertFromBedrock(ir.IRBlock("minecraft:stone []"))
    if !ok { log.Fatal("stone is missing from the block table") }
    storage := bedrock.NewReadWriteBedrockStorage(
        db, 0, air, bedrock.DefaultBlockDecode, bedrock.DefaultBlockEncode,
    )
    position := celler.SubChunkPos{ChunkPos: celler.ChunkPos{X: -2, Z: 3}, Y: -4}
    local := celler.ToSubChunkBlockPos(1, 2, 3)
    chunk := storage.GetOrEmpty(position)
    change := celler.BlockChange[uint32, uint32]{}.SetPos(local).SetID(stone.Hash())
    chunk.ChangeBlock(change)
    storage.Set(position, chunk)
    fmt.Println(storage.Get(position).BlockID().Get(local) == stone.Hash())
}
```

访问磁盘 LevelDB 时，将空路径换成数据库目录。Java 使用 `mca.NewReadWriteMCAKvFromDir` 打开 `region` 目录，再交给 `java.NewReadWriteJavaStorage`。修改 Java 子区块后必须调用 `Set` 标记待写入数据。结束时先调用 Java 存储的 `Close` 刷出缓存，再关闭传入的 MCA 数据库。传入的底层数据库由调用者管理。

这些适配器面向方块和方块实体数据。Java 编码器会重建区块 NBT，不保留原有的生物群系、光照、实体、计划刻等完整世界信息；Bedrock 解码器只保留第一层方块。因此不能把这里的读写往返视为完整存档的无损编辑。旧存储接口也没有统一的错误返回通道，需要检查原始编解码错误的调用方应直接使用相应的 decoder。

## 扩展与验证

需要使用自己的方块 ID 时，Bedrock 构造函数接收 `BlockDecode`、`BlockEncode` 回调。它们把磁盘里的单个方块 NBT 与 ID 互相转换。需要接入新的存储后端时，可以实现 `kvdb.KVDB`，或者围绕根包的子区块读写接口提供适配器。

在仓库根目录执行：

```sh
go test ./...
go test -race ./...
go vet ./...
```

测试自行生成临时 LevelDB、MCA 和区块数据，不需要外部世界样本。测试覆盖方块与 SNBT 往返、负坐标、调色板、MCA 缓存淘汰，以及底层存储。

## 历史代码的位置

`_legacy/dup` 保存重构前的存储实现，`_legacy/cli_dup` 保存依赖旧接口的 CLI。2025-02-21 的 `b6ff3f2` 提交把旧实现移到 `dup`，并建立了当前的 `kvdb`、`subchunk_storage` 结构。当前模块没有引用这些旧实现；`_legacy` 的目录名让 Go 的 `./...` 自动跳过它们。它们的保留用途和来源见 [_legacy/README.md](_legacy/README.md)。
