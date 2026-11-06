# 维护 neomega-celler

本模块负责 Minecraft 原生存档的读取、编辑和写入，Go 模块路径为 `celler`。它保留原始记录及未解释的 NBT 字段，消费者的 API、格式范围与持久化规则见 [README.md](README.md)。工作区与依赖声明遵循 [根 AGENTS.md](../AGENTS.md)。

Bedrock 以 LevelDB 区块记录为原始单位，Java 以完整区块 NBT 为原始单位。只有调用者请求 section 时才解释方块层；修改层后必须经 `SetSection` 回到区块，再由世界句柄保存。

## 结构与边界

`bedrock` 管理原生区块记录、section 和世界元数据，`bedrock/leveldb` 配置数据库引擎。`java` 管理区块 NBT 与维度，`java/region` 处理 MCA 扇区、压缩与外置 MCC。`storage` 将世界文件访问接到目录或 PackFS，`layer_storage` 独立保存 16³ 个可比较值。`bedrock/mcworld` 只读访问 zip 中的原生基岩世界。没有 chunk/subchunk 索引的结构文件由 converter-core 的 `structure` 解析并写成原生存档。

模块依赖方块 IR、NBT 与 PackFS 基础库，不引入转换规则、邻接修复或渲染职责。方块版本属于 Bedrock 调色板条目，不能挪为 section 统一版本。

## 变更约束

保留未修改记录和未知 NBT 字段；不要为了方便读取而重建整份区块文档。层坐标布局是 X-Z-Y，Y 最快变化，磁盘编解码负责布局换算。Java 生物群系单独采用 Y-Z-X 的 4³ 采样。

不存在的区块或 section 用 `fs.ErrNotExist` 表达，损坏和不支持的格式返回错误，全空气层仍是有效数据。不要以默认空气隐藏解码失败。

Java 世界借用文件后端，关闭 world 后才能关闭 PackFS。修改刷新、提交和快照逻辑时，保持 README 描述的持久化顺序。world 及文件后端由同一 goroutine 使用；并行计算应接收独占区块数据。

## 验收与示例

从工作区根目录执行：

```sh
./check.sh neomega-celler
```

测试自行建立临时存档，覆盖多层与条目版本、未知 NBT 保留、负坐标、MCC 内外切换、PackFS 快照及导出目录后重新读取。修改编解码或持久化时，应在对应测试中验证原始字段和保存后重新读取的结果。

`examples/inspect-bedrock` 与 `examples/inspect-java` 是只读已有存档的线性示例。按 README 中的命令提供实际路径与存在的区块。示例修改后须复读中文注释，检查维度、坐标、所有权和关闭顺序是否自足，再实际运行。
