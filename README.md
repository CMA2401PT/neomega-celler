# neomega-celler

`neomega-celler` 围绕 `LayerStorage` 构建面向不同场景的 Minecraft 方块存储，并让这些存储互相转换。`LayerStorage` 是一个子区块内某一类值的存储模型。子区块是 16×16×16 的方块区域。仓库里的其他存储结构都围绕这个模型组织。

`LayerStorage` 像网络里的 IP 层，是整个设计的“窄腰”：上面可以封装 Java 存档、Bedrock 数据库和同步机制，下面可以使用复杂的调色板位压缩，中间的接口保持简单、稳定。Go 模块名是 `celler`，导入路径使用 `celler/...`。

## 为什么需要一个窄腰

同一份方块数据会以多种形态存在。保存时需要对接 Bedrock 的 LevelDB 数据库或 Java 的 MCA 区域文件。运算时需要方便随机读写的内存数组。方块实体数据很稀疏，适合只保存少量非默认值。同步时又需要把远端的数据落到本地选择的存储实现上。

如果这些形态两两对接，每增加一种形态，就要补写多条转换路径。通过 `LayerStorage`，每种形态只需接入同一个模型。磁盘容器可以跨越许多区块，但拆到这里，每一层处理的范围始终是 16×16×16。

这也把问题分成了两部分：围绕同一模型选择合适的存储方案，以及通过 `CopyFrom` 在不同实现之间搬运数据。

## LayerStorage 怎么工作

根包的 [layer_storage.go](layer_storage.go) 定义了这组能力。读取只需要知道默认值，以及哪些位置与默认值不同。写入负责初始化和从其他层复制。随机访问在此基础上提供 `Get`、`Set`，供逐点读取和修改使用。

```go
type LayerStorageRead[D any] interface {
    Default() D
    IterNonDefault(fn func(SubChunkBlockPos, D) (next bool))
}

type LayerStorageWrite[D any] interface {
    InitWithDefault(data D)
    CopyFrom(LayerStorageRead[D])
}
```

“一个默认值 + 所有非默认点”足以描述整层。目标实现先用默认值填满 4096 个位置，再逐点写入差异，就能重建源数据。因此，`CopyFrom` 只需要 `LayerStorageRead[D]`，无需了解源实现的数组、字典或压缩布局。这里复制的是相同值类型 `D` 的数据。值本身的含义需要双方约定一致。

`Palette`、`Plain`、`Map` 的 `CopyFrom` 对已知类型选择专门的路径。例如，`Palette` 到 `Palette` 直接复制压缩数组和调色板。遇到未知类型时，它们用 `InitWithDefault` 和逐点 `Set` 完成通用复制。快路径只影响效率。

因此，新实现只要提供读接口，就能成为现有通用实现的复制来源。它再提供同样的通用 `CopyFrom`，就能接收其他实现的数据。扩展时无需编写两两之间的转换器。只表示统一值的 `PlaceHolder` 有单独的限制，见下文。

## 窄腰之下：按场景选择实现

方块实体的结构化数据在这里用 SNBT 保存，即 Minecraft NBT 数据格式的文本表示。为了在相同的空间范围内选择不同的时间、内存开销，[layer_storage](layer_storage) 提供四种实现：

- `Palette` 用调色板和位压缩紧凑保存重复值，与 Bedrock、Java 的磁盘调色板结构对应。
- `Plain` 用平铺数组保存 4096 个值，可以直接按下标随机读写，适合运算。
- `Map` 只保存非默认点，适合方块实体 SNBT 等稀疏数据。
- `PlaceHolder` 用一个值表示整层，两个存档适配器都用它表示统一为 0 的附加值层。

`PlaceHolder.Set` 不改变数据。它的 `CopyFrom` 只取源默认值，因此无法保留非均匀层的逐点差异。需要保存这些差异时，应选择其他实现。

## 窄腰之上：组合、保存和同步

`SubChunk` 描述一个子区块的全部方块信息。它没有把每个格子的所有信息打包成一个值，而是按信息种类拆成三层 `LayerStorage`：方块 ID、方块实体 SNBT 和附加值。这与 Bedrock 和 Java 存档一致，普通方块和带 NBT 的方块本来就分开存放。

拆开的好处是每层可以按自己的疏密选择实现。方块 ID 每个格子都有，适合 `Palette`。SNBT 只出现在少数方块上，适合 `Map`。附加值层预留给将来非常稀疏的方块注释信息，目前两个适配器都用 `PlaceHolder` 填 0。

`subchunk.Simple` 的三个层类型参数都只要求满足 `celler.Storage`，即可随机读写的 `LayerStorage`，所以每层用哪种实现由实例化的一方决定。`SubChunk` 还带有版本号和元数据，供同步判断新旧。

为了管理多个子区块，`SubChunkStorage` 按 `SubChunkPos` 存取它们。这个位置由水平区块坐标和子区块高度组成。`subchunk_storage/bedrock` 通过 `kvdb` 对接 LevelDB。`subchunk_storage/java` 通过 `subchunk_storage/java/mca` 对接 MCA 区域文件。两个适配器各自处理自己的格式与层数据之间的编解码，因此 Java 和 Bedrock 之间无需再写一套专用的逐块转换逻辑。

同步也使用这条边界。`SubChunkChange` 可以携带完整的 `Base` 子区块或逐块增量。`subchunk.Simple.ApplyChange` 接收比本地版本更新的 `Base` 时，会复制版本号和元数据，并对三层分别调用 `CopyFrom`。远端的层实现可以与本地不同。

需要压缩传输方块命令时，`frontend/replica` 提供坐标与 `uint16` 值的命令编解码。它目前是独立的传输工具，尚未直接接入 `SubChunkChange`。

## 格式之间的转换：Java → Bedrock

转换只需遍历源子区块，把三层复制到目标实现，再写回目标存储。反方向也遵循相同的步骤。默认 Java 适配器和 Bedrock 转换器都使用规范化 Bedrock 方块哈希，因此默认配置下复制 ID 时无需再次翻译。

下面的完整程序从 Java 的 `region` 目录读取子区块，并写入内存 LevelDB。运行环境和命令见“运行例子”。

```go
package main

import (
    "fmt"
    "log"
    "os"

    "celler"
    "celler/kvdb"
    "celler/subchunk_storage/bedrock"
    "celler/subchunk_storage/java"
    "celler/subchunk_storage/java/mca"
    "neomega-blocks/ir"
)

func copyJavaToBedrock(
    source *java.JavaStorageReadOnly,
    target *bedrock.BedrockStorageReadWrite[uint32],
) int {
    copied := 0
    source.IterAll(func(pos celler.SubChunkPos) bool {
        src := source.Get(pos)
        if src == nil || src.Version() == 0 {
            return true
        }
        dst := target.GetOrEmpty(pos)
        dst.BlockIDStorage.CopyFrom(src.BlockID())
        dst.BlockAttachStorage.CopyFrom(src.BlockAttach())
        dst.BlockSNBTStorage.CopyFrom(src.BlockSNBT())
        // Set 将版本号 0 当作删除，所以也要复制源的非零版本号。
        dst.CurrentVersion = src.Version()
        target.Set(pos, dst)
        copied++
        return true
    })
    return copied
}

func main() {
    if len(os.Args) != 2 {
        log.Fatal("usage: convert <java-region-directory>")
    }
    mcaDB, err := mca.NewReadOnlyMCAKvFromDir(os.Args[1])
    if err != nil {
        log.Fatal(err)
    }
    defer mcaDB.Close()
    source := java.NewReadOnlyJavaStorage(mcaDB)

    db, err := kvdb.NewReadWriteLevelDB("")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    air := ir.IRBlock("minecraft:air []").Hash()
    target := bedrock.NewReadWriteBedrockStorage(
        db,
        0, // Bedrock 主世界维度。
        air,
        bedrock.DefaultBlockDecode,
        bedrock.DefaultBlockEncode,
    )
    fmt.Printf("copied %d subchunks\n", copyJavaToBedrock(source, target))
}
```

Java 的 `IterAll` 为每个磁盘区块枚举 `Y = -4..19` 的候选位置，所以代码通过 `Get` 跳过不存在的子区块。`GetOrEmpty` 在目标缺失时创建版本号为 0 的子区块，在目标已有数据时返回现有数据。逐层 `CopyFrom` 会覆盖目标层的内容。

同步场景可以用 `ApplyChange(celler.SubChunkChange[uint32, uint32]{Base: src})` 完成三层复制，并携带版本号和元数据。但它只接受版本号更大的 `Base`。存档转换要覆盖已有目标时，直接复制三层可以避免同步的新旧判断。存档适配器不会持久化这里的同步版本号和元数据。

## 方块 ID 从哪里来

基础类型允许调用者选择自己的 `~uint32` 方块 ID。默认 Bedrock 转换器和 Java 适配器使用 `neomega-blocks/presets/block-converts/v1` 的规范化 Bedrock 方块哈希。它不是游戏 runtime ID，也不能直接当作旧版 `UIBI` 使用。

输入方块缺少属性时，默认转换会使用 `neomega-blocks` 的 canonical 逻辑补齐属性，再获得规范化方块的哈希。NBT 和 SNBT 的读写使用 `neomega-nbt/presets/v1`。需要自己的 ID 体系时，Bedrock 构造函数可接收自定义的 `BlockDecode`、`BlockEncode` 回调，以转换磁盘里的单个方块 NBT。

## 运行例子

本地开发需要 Go 1.25.5 或以上。将 `neomega-celler`、`neomega-blocks`、`neomega-nbt` 放在同一级目录，供 `go.mod` 中的 `replace` 使用。如果使用父目录的 `go.work`，需要把这三个模块都加入工作区。

要运行上面的转换程序，将代码保存为 `/tmp/celler-convert.go`，在仓库根目录执行：

```sh
go run /tmp/celler-convert.go /path/to/java-world/region
```

内存 LevelDB 会随程序结束释放。需要保存到磁盘时，将 `kvdb.NewReadWriteLevelDB("")` 的空路径换成目标数据库目录。该程序复制的是适配器支持的方块数据，完整存档的保留范围见“适配器的局限”。

如果想先验证单个方块的保存和读回，可以使用下面的内存 LevelDB 例子。它通过 `ChangeBlock` 修改数据并推进版本号，再经过 Bedrock 编解码写入和读取。

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
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    air := ir.IRBlock("minecraft:air []").Hash()
    stone, ok := blocks.ConvertFromBedrock(ir.IRBlock("minecraft:stone []"))
    if !ok {
        log.Fatal("stone is missing from the block table")
    }
    storage := bedrock.NewReadWriteBedrockStorage(
        db,
        0,
        air,
        bedrock.DefaultBlockDecode,
        bedrock.DefaultBlockEncode,
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

将代码保存为 `/tmp/celler-basic.go`，在仓库根目录执行：

```sh
go run /tmp/celler-basic.go
```

最后一行应为 `true`。

写入 Java 存档时，用 `mca.NewReadWriteMCAKvFromDir` 打开 `region` 目录，再交给 `java.NewReadWriteJavaStorage`。修改子区块后必须调用 `Set` 标记待写入数据。结束时先调用 Java 存储的 `Close` 刷出缓存，再关闭 MCA 数据库。传入的底层数据库由调用者管理。

## 适配器的局限

这些适配器面向方块和方块实体数据。Java 编码器会重建区块 NBT，丢失原有的生物群系、光照、实体和计划刻等信息。Bedrock 解码器只保留第一层方块，写入时也只生成这一层。因此，读写往返不是完整存档的无损编辑。

共享的方块 ID 约定解决了层数据复制时的 ID 对接，但两版的方块实体 SNBT 仍有各自的语义。直接复制 SNBT 不等于完成两版方块实体格式的转换。

旧存储接口没有统一的错误返回通道。例如，读取失败可能表现为没有数据。需要检查原始解码错误时，应直接使用相应的 `ChunkDecoder`。Java 写入者的 `Close` 会返回写出错误，调用方需要检查。

## 测试

在仓库根目录执行：

```sh
go test ./...
go test -race ./...
go vet ./...
```

测试自行生成临时 LevelDB、MCA 和区块数据，无需外部世界样本。测试覆盖方块与 SNBT 往返、负坐标、层实现之间的复制、调色板、MCA 缓存淘汰及底层存储。
