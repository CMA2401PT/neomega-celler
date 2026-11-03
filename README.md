# neomega-celler

`neomega-celler` 提供 Minecraft Bedrock 和 Java 原生存档的读取、编辑与写入。Bedrock 使用本机目录中的 LevelDB 数据库；Java 使用本机目录或 PackFS 中的 MCA 区域文件。Go 模块名为 `celler`，导入路径使用 `celler/...`。

存档先以原始记录或完整 NBT 文档读入，再按需要解释其中的方块和其他数据。编辑一个 section 后，把它写回所属区块，最后保存区块。Section 是 16×16×16 的空间区域。这样的读取方式保留了尚未解释的记录与字段，也让调用者可以直接处理原始数据。

消费者可以从下方的原生存档 API 和可运行示例开始；模块结构、变更约束与验收见 [AGENTS.md](AGENTS.md)。

## 从世界文件到层存储

两种存档各自提供原生区块模型。内存中的方块层使用 `layer_storage`，它以调色板、平铺数组或稀疏映射保存 4096 个值。

```text
bedrock.World → 本机 LevelDB → 区块记录 → bedrock.Section → 多个方块层
java.World   → 目录 / PackFS → MCA/MCC → java.Section    → 一个方块层
```

各包按数据的含义和存放方式划分：

| 包 | 职责 |
| --- | --- |
| `celler` | 区块坐标和绝对方块坐标 |
| `layer_storage` | 独立的 16³ 内存存储及实现之间的复制 |
| `bedrock` | 原生区块记录、section、多层方块、方块实体、生物群系及 `level.dat` |
| `bedrock/leveldb` | 使用现有 LevelDB 引擎打开本机数据库，配置 Bedrock 压缩方式 |
| `java` | 原生区块 NBT、section、世界元数据和维度目录 |
| `java/region` | MCA 槽位、扇区分配、压缩记录和外置 MCC 文件 |
| `storage` | 文件后端接口、本机目录与 PackFS 适配 |

NBT 文档读写使用 `neomega-nbt/presets/v1`。`neomega-blocks/ir` 提供方块名称与属性的文本表示。`neomega-packfs` 提供容器、提交和快照。

## 方块怎样表示

层内坐标使用 `layer_storage.Position`。`layer_storage.At(x, y, z)` 取坐标的低四位，也适用于负数绝对坐标。平铺数组采用 `(x << 8) | (z << 4) | y`，即 Y 最快变化。各存档编解码器负责磁盘布局与该内存布局之间的排列。

Java 的方块层是 `Palette[ir.IRBlock]`。属性保持 Java 的字符串表示，例如 `minecraft:oak_slab ["type"="bottom","waterlogged"="true"]`。

Bedrock 的每一层是 `Palette[bedrock.Block]`。`Block.State` 保存 IR，`Block.Version` 保存该 palette 条目的版本号。版本号属于条目，同一个 section 内可以出现不同版本。`Section.Layers` 按磁盘顺序保存所有层，层数来自记录本身。

解码得到的 IR 已经 tidy，即名称和属性具有规范的文本形式。调用者新增方块时应提供有效、已经 tidy 的 IR。方块名称与状态直接来自存档，调色板中可以保留自定义方块。Palette 的第一个值是默认值，可能是任意方块。

`layer_storage` 的四种实现适用于不同访问方式：

- `Palette`：重复值较多时，用局部调色板和压缩索引减少内存。
- `Plain`：平铺保存 4096 个值，适合连续运算。
- `Map`：只保存非默认位置，适合稀疏数据。
- `PlaceHolder`：用一个值表示均匀层。其 `Set` 不改变数据，`CopyFrom` 只取得源默认值。

`CopyFrom` 复制层数据。`ConvertPalette` 生成独立的索引数组并映射 palette 值，因此修改结果不会影响来源。层存储接受可比较的 Go 值，包括 Bedrock 的方块状态与版本组合。

## Bedrock：按记录读取和编辑

`bedrock.Open` 接收包含 `db` 和 `level.dat` 的世界目录。维度使用原生数值 ID。`ReadChunk` 返回自有字节切片，`Chunk.Records` 按坐标和维度之后的键后缀保存记录。

`Section(y)` 只解码指定高度的 section。修改返回的层后，调用 `SetSection` 将结果放回区块。`WriteChunk` 用一个 LevelDB batch 替换该区块已识别的记录集合；未改动记录沿用原有字节，其他数据库键保持原值。`DeleteSection` 删除一条 section 记录，`DeleteChunk` 删除该区块已识别的记录。

```go
world, err := bedrock.Open(directory, bedrock.Options{})
if err != nil {
    return err
}
defer world.Close()

chunk, err := world.ReadChunk(0, celler.ChunkPos{X: -2, Z: 3})
if err != nil {
    return err
}
section, err := chunk.Section(-4)
if err != nil {
    return err
}
// block 的 State 与 Version 由调用者按来源版本提供。
section.Layers[0].Blocks.Set(layer_storage.At(1, 2, 3), block)
if err := chunk.SetSection(section); err != nil {
    return err
}
return world.WriteChunk(chunk, &opt.WriteOptions{Sync: true})
```

这里的 `opt` 来自 `github.com/df-mc/goleveldb/leveldb/opt`。`Sync: true` 要求该 batch 持久化后返回。传入 `nil` 使用引擎默认写入策略。世界元数据通过 `ReadLevel`、`WriteLevel` 访问，并保留文件头版本。

`ScanChunks` 基于数据库快照遍历区块。回调取得的字节在迭代器前进后仍然有效，也可以在回调中写入世界。数据库的其他记录、范围迭代、快照和批量操作可直接通过 `World.DB` 使用。

方块实体通过 `BlockEntities` 和 `SetBlockEntities` 读写，保留其原始位置、ID 和 NBT 类型。`Biomes2D` 对应二维高度与生物群系记录；`Biomes3D` 保留 16³ 数值 palette 和“沿用上一 section”的标记。三维记录首个 section 的 Y 由维度高度定义决定，记录本身不携带它。

## Java：编辑完整 NBT 中的 section

`java.OpenDirectory` 打开本机世界目录。`java.Open` 接收文件后端；可写后端实现 `storage.Writer`，只读后端可以是 `os.DirFS` 或 PackFS 快照视图。

世界中的维度以相对目录表示：主世界为 `java.Overworld`，下界为 `java.Nether`，末地为 `java.End`。自定义维度使用 `java.Dimension("dimensions/namespace/name")`。`Terrain`、`Entities`、`POI` 分别选择对应的 region 目录。

`ReadChunk` 返回完整的 `Chunk.NBT`。`Section(y)` 解码方块与生物群系；修改后调用 `SetSection`。它会更新该 section 的已解释字段，并保留其他 NBT。方块实体和其他区块数据仍在 `Chunk.NBT.Value` 中，使用原生字段名称与类型访问。

```go
chunk, err := world.ReadChunk(java.Overworld, position)
if err != nil {
    return err
}
section, err := chunk.Section(-4)
if err != nil {
    return err
}
section.Blocks.Set(layer_storage.At(1, 2, 3), ir.IRBlock("minecraft:stone []"))
if err := chunk.SetSection(section); err != nil {
    return err
}
if err := world.WriteChunk(java.Overworld, chunk); err != nil {
    return err
}
return world.Commit()
```

`Section.Blocks == nil` 表示缺少方块字段，例如仅保存光照的 section。Java 生物群系使用独立的 4×4×4 采样数组，顺序是 Y-Z-X。`SectionYs` 从实际记录枚举高度。

`ReadRaw` 和 `WriteRaw` 访问带时间戳的压缩记录，适合保留现有压缩数据，或在计算任务中提前完成压缩。`Scan` 枚举实际存在的槽位。`ReadLevel` 和 `WriteLevel` 处理 gzip 压缩的 `level.dat`。

`NewChunk` 只初始化坐标、格式版本和 section 列表。创建完整区块时，由调用者提供目标存档需要的生成状态等字段。

## Java 存档放入 PackFS

PackFS 内部保持原有世界目录结构。容器根目录可以直接包含 `level.dat`、`region`、`entities`、`poi` 和各维度目录。使用 `storage.Pack{PackFS: container}` 打开可写世界：

```go
world, err := java.Open(
    storage.Pack{PackFS: container},
    java.Options{MaxOpenRegions: 16},
)
```

`World` 借用容器。调用者负责打开和关闭容器，以及创建、读取和恢复快照。region 缓存有容量限制，默认最多保留 16 个打开的文件。

写入过程分为两个层次：

1. `world.Flush()` 将所有 region 的数据和头表写入文件后端。
2. `world.Commit()` 先执行 Flush，再提交后端。PackFS 此时发布整组文件的新状态；目录后端同步修改过的文件及父目录。

`world.Close()` 刷新并释放 region 句柄。需要明确持久化时，应先成功调用 `world.Commit()`。本机目录的同步不具有 PackFS 的多文件原子发布能力。

创建快照前调用 `world.Flush()`，然后调用 `container.Snapshot(name)`，因为 Snapshot 自身会提交容器。关闭容器之前先关闭 world；PackFS 的 Close 也会提交容器。恢复快照之前关闭 world 及其他文件句柄，恢复后重新打开 world，避免继续使用旧 region 头表。

一个 world 及其文件后端由同一个 goroutine 使用。区块解码、计算和压缩可以在独立任务中完成，再把独占的结果交给文件所有者写入。

## 格式与错误约定

Bedrock 方块编解码支持 persistent subchunk 版本 1、8、9，以及带 `name`、`states`、`version` 的 palette 条目。区块记录层可以保留原始字节，具体内容在请求解码时检查。

Java section 编辑支持现代 `sections` / `block_states` / `biomes` 结构及 64 位整数内不跨字打包的 palette。Region 层读取 gzip、zlib、未压缩及 Java LZ4Block 压缩记录；普通写入采用 zlib，大记录使用 `.mcc` 外置文件。原始记录接口可以保留已有压缩内容。

不存在的区块或 section 返回 `fs.ErrNotExist`。全空气是一个正常的数据值。损坏记录与不支持的编码返回错误，不通过默认方块掩盖问题。NBT 文档编辑保留根名和标签类型；重新编码 Go map 后，Compound 字段顺序可能变化。

读取后返回的数据由调用者持有。提取 section 后，应通过 `SetSection` 应用对方块层和生物群系的修改。嵌套 NBT 使用普通 Go map、slice 和 array；同时处理同一份文档时，由调用者安排所有权。

## 运行示例

本地开发使用 Go 1.25.5。将 `neomega-celler`、`neomega-blocks`、`neomega-nbt` 和 `neomega-packfs` 放在同级目录，供模块的本地替换引用。

以下命令在父工作区目录执行。坐标参数是区块坐标。示例只读取已有存档并打印 section 信息：

```sh
go run ./neomega-celler/examples/inspect-bedrock \
  -world /path/to/bedrock-world -dimension 0 -x -2 -z 3

go run ./neomega-celler/examples/inspect-java \
  -world /path/to/java-world -dimension . -x -2 -z 3

go run ./neomega-celler/examples/inspect-java \
  -pack /path/to/world.pack -dimension DIM-1 -x -2 -z 3

```

完整代码见 [Bedrock 检查示例](examples/inspect-bedrock/main.go) 和 [Java 检查示例](examples/inspect-java/main.go)。路径需替换为自己的已有存档，区块必须实际存在。
