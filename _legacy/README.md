# 历史实现

`dup` 是 2025-02-21 的 `rebuild project` 提交从旧 `subchunk_storage` 移出的实现。当前实现使用 `kvdb` 和新的 `subchunk_storage`。`cli_dup` 依赖重构前已删除的接口。

这些源码保留供查阅，不作为当前库使用。Go 的 `./...` 会跳过以下划线开头的目录，因此归档代码不会参与构建和测试。
