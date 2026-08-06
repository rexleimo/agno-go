---
title: 沙盒化文件 I/O
description: 使用显式根目录、根句柄绑定的文件操作、大小限制和审计能力，限制 Agent 的文件访问权限。
outline: deep
---

# 沙盒化文件 I/O

把 `read_file` 或 `write_file` 交给 Agent，本质上是在授予能力，而不只是提供一个方便的函数。HNO 的沙盒化文件 I/O 将这项能力限制在明确配置的文件系统根目录中，并让实际文件操作绑定到这些根目录。

<img src="/diagrams/sandboxed-file-io.svg" alt="HNO 沙盒化文件 I/O 架构图：Agent 工具调用经过能力策略和 root-bound os.Root 句柄，最终只能访问隔离的读根和写根。" />

::: warning 该能力的边界
这是一种**路径名约束**，不是完整的操作系统沙盒。它阻止文件工具解析到配置根目录以外的位置，包括通过逃逸性符号链接解析到外部的位置；但它不会隔离进程、限制网络，也不能自动消除挂载点、已有硬链接、设备文件或已存在于授权根目录中的其他特权资源。
:::

## 为什么仅做路径白名单不够

用 `filepath.Abs`、`filepath.Rel` 或字符串前缀来检查路径，确实可以拒绝明显的 `../` 输入，但它本身不是安全边界：

- 符号链接可能改变一个已检查路径最终解析的位置；
- 先验证名称、再调用 `os.ReadFile` 或 `os.WriteFile` 会留下检查与使用之间的竞争窗口（TOCTOU）；
- 绝对路径行为和进程工作目录会让同一调用方输入出现不同解释；
- Windows 还存在盘符相对路径、UNC、设备名和备用数据流等特殊形式。

因此，HNO 会验证 Agent 可见的路径名称，**同时**使用 Go 的 `os.OpenRoot` 和 `*os.Root` 执行真实操作。操作系统收到的是与已打开目录句柄绑定的根相对名称，而不是一个不受限制的宿主机路径。

## 能力模型

`file.NewSandbox` 默认 fail-closed。没有配置根目录的 sandbox 仍可以被构造，但所有读写操作都会被拒绝，直到显式授予相应能力。

| 能力 | 配置方式 | 授予的权限 |
| --- | --- | --- |
| 读取 | `file.WithReadRoots("./inputs", "./templates")` | 在一个或多个按顺序检查的根目录下读取、列目录、查询元数据和提取 PPTX。 |
| 写入 | `file.WithWriteRoot("./workspace")` | 在一个根目录下创建、写入、删除文件和创建目录；不自动授予读取权限。 |
| 读取上限 | `file.WithMaxReadBytes(n)` | 单次普通文件读取可返回的最大字节数。默认 10 MiB。 |
| 写入上限 | `file.WithMaxWriteBytes(n)` | 单次写入可接收的最大字节数。默认 5 MiB。 |
| 覆盖 | `file.WithAllowOverwrite(true)` | 允许文件工具替换一个已有的普通文件。默认关闭。 |
| 审计 | `file.WithAudit(fn)` | 每次成功的沙盒操作后接收一条 `AuditEntry`。 |

读根和写根被刻意分离。报告 Agent 可以读取源材料但不能修改它；生成 Agent 可以写产物但不会自动获得查看其他工作区文件的权限。

## 为 Agent 配置沙盒

对于模型、用户或任何不可信来源提供路径的 Agent，请使用 `NewWithSandbox`。在沙盒模式下，Agent 传入的路径必须始终相对配置根目录；不要把宿主机绝对路径传给沙盒化工具。

```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./agent-inputs", "./shared-templates"),
    file.WithWriteRoot("./agent-workspace"),
    file.WithMaxReadBytes(10<<20),
    file.WithMaxWriteBytes(5<<20),
    file.WithAudit(func(entry file.AuditEntry) {
        log.Printf("sandbox op=%s path=%s size=%d at=%s",
            entry.Operation, entry.Path, entry.Size, entry.At.Format(time.RFC3339))
    }),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close() // 所有 Agent / 工具工作结束后再关闭。

fileTools := file.NewWithSandbox(sandbox)
fileGenTools := filegen.NewWithSandbox(sandbox)

ag, err := agent.New(agent.Config{
    Model: model,
    Toolkits: []toolkit.Toolkit{
        fileTools,
        fileGenTools,
    },
})
if err != nil {
    log.Fatal(err)
}
```

当工具调用可能并发执行时，审计回调应写入一个并发安全的审计接收端。审计记录包含操作名、根相对路径、大小和时间戳，不包含文件内容。

如果多个 Toolkit 共用同一个 `Sandbox`，由调用方统一在所有工作结束后调用一次 `sandbox.Close()`。单独关闭其中一个 Toolkit 会关闭其底层 sandbox，进而使共享它的其他 Toolkit 不可再用。

## 一次工具调用如何经过沙盒

1. 模型请求 `read_file`、`write_file`、`create_file` 或 `create_directory` 等工具操作。
2. Toolkit 检查 sandbox 配置。传入 `nil` sandbox 时会 fail-closed，而不会退回到不受限制的宿主机 I/O。
3. Sandbox 规范化并拒绝不安全的 Agent 路径：空路径、绝对路径、`..` 穿越、NUL 字节，以及 Windows 上的盘符相对路径、保留设备名和备用数据流等形式。
4. Sandbox 按请求的能力选择读根，或选择唯一的写根。
5. 真实操作通过 `*os.Root` 执行（`Open`、`Stat`、`OpenFile`、`Mkdir`、`OpenRoot`、`Remove`）。符号链接只能在解析结果仍位于已打开根目录内时被跟随。
6. 读取和写入大小上限生效；成功写入会同步到文件系统；成功操作可触发审计记录。

验证步骤用于尽早拒绝格式错误的名称并提供清晰错误信息；真正防止已验证名称在之后解析到根目录外的边界，是 `*os.Root` 的实际操作。

写入时，如预检查观察到最终符号链接，sandbox 会拒绝该目标。这属于纵深防御，
不是跨平台的原子 no-follow 原语：如果另一个进程能修改授权写根，它可以在检查后
替换一个仍在根目录内的条目。`*os.Root` 依旧阻止替换逃出根目录，但不能保证根内
目标的身份。存在这种并发写入方时，请使用每次运行独立的私有根目录或更高层的锁。

## Toolkit 行为

### `file.FileTools`

`file.NewWithSandbox(sandbox)` 提供以下根相对工具调用：

| 工具调用 | 所需能力 | 说明 |
| --- | --- | --- |
| `read_file(path)` | 读取 | 仅读取普通文件，并执行读取上限。 |
| `list_files(path)` | 读取 | 列出一个目录；使用 `"."` 表示根目录。 |
| `file_exists(path)` | 读取 | 在读根内成功解析名称时返回元数据。 |
| `read_pptx(path)` | 读取 | 先打开沙盒化文件句柄，再解析 ZIP/PPTX 内容。 |
| `write_file(path, content)` | 写入 | 按需在写根内部创建父目录。 |
| `delete_file(path)` | 写入 | 删除写根内的文件或空目录。 |

对 `write_file` 而言，新目标默认以排他方式创建。替换已有普通文件只有在 sandbox 配置了 `WithAllowOverwrite(true)` 时才允许。

### `filegen.FileGenToolkit`

`filegen.NewWithSandbox(sandbox)` 复用同一个写根能力来生成产物：

| 工具调用 | 路径参数 | 已有目标的行为 |
| --- | --- | --- |
| `create_file` | `file_path` | 必须同时满足工具调用中的 `overwrite: true` 和 sandbox 的 `WithAllowOverwrite(true)`，才可覆盖已有文件。 |
| `create_directory` | `dir_path` | 按需创建父目录；目标已经存在时返回错误。 |
| `generate_from_template` | 无 | 仅在内存中做模板替换，不访问文件系统。 |

这种双重确认规则确保：模型不能仅因为宿主开启了通用覆盖策略就替换文件；调用方传入 `overwrite: true` 也不能绕过保守的 sandbox 策略。

## 路径约定与示例

当写根设置为 `file.WithWriteRoot("./agent-workspace")` 时，Agent 侧参数的含义如下：

| 参数 | 结果 |
| --- | --- |
| `"reports/summary.md"` | 允许，解析到 `./agent-workspace` 下。 |
| `"../secrets.txt"` | 作为路径穿越被拒绝。 |
| `"/etc/passwd"` 或 `"C:\\Windows\\win.ini"` | 作为绝对路径或平台特殊路径被拒绝。 |
| `"out/report.txt"`，且 `out` 是指向根外的符号链接 | 由根绑定操作拒绝。 |
| 用 `"."` 作为写入或创建目标 | 被拒绝，因为根目录本身不是文件目标。 |

配置多个读根时，HNO 按配置顺序检查，并选择第一个包含目标根相对名称的根目录。这样可以授权一个只读模板目录和一个输入目录，却不把其中任一目录暴露为可写位置。

## 覆盖写入的语义

Go 1.24 的根绑定 API 没有可用于此场景的 root-bound 原子替换接口。因此，获得明确覆盖授权后的已有文件使用安全的 root-bound 截断并写入语义，而不是原子 replace。

这意味着应用不应把 `WithAllowOverwrite(true)` 用于必须保留旧版本直到新版本完整落盘的关键文件。对此类需求，请在私有工作区中生成新产物、经过校验后由更高层的受控发布流程处理替换。

## 兼容性与迁移

| 构造函数 | 推荐用途 |
| --- | --- |
| `file.New()` | 不受限制的旧行为。仅用于调用方永远不会传入不可信路径的可信程序。 |
| `file.NewWithBaseDir(baseDir)` | 兼容构造函数：把读写都限制在一个根目录，同时保留旧的根内绝对路径和覆盖行为。 |
| `file.NewWithSandbox(sandbox)` | 新的 Agent 文件工具首选。路径为根相对路径，能力显式配置。 |
| `filegen.New()` | 不受限制的旧文件生成，仅限可信程序。 |
| `filegen.NewWithSandbox(sandbox)` | Agent 生成文件产物的首选构造函数。 |

一个实用的迁移方式是：为每次运行或每个租户配置独立工作区卷作为写根；把任何共享输入显式配置为读根。不要把仓库 checkout、用户主目录或大范围挂载卷当作默认写根。

## 安全边界与部署建议

Sandbox 保护的是**文件 Toolkit 能解析哪些名称**，并不会让任意代码执行变得安全。对于多租户、不可信代码或高保障工作负载，请继续叠加部署级控制：

- 每个租户或运行使用私有卷/目录树；
- 宿主机 ACL 和低权限服务账号；
- 对执行进程使用容器、虚拟机或专门 sandbox runtime；
- CPU、内存、进程数和磁盘配额限制；
- 出站网络白名单或 egress 隔离；
- 按数据分级设置持久审计日志及保留策略。

`os.Root` 本身不会阻止文件系统边界穿越（例如 Linux bind mount），也不会撤销授权根目录中已经存在的特殊文件或硬链接所代表的访问能力。应把选定的根目录及其外围部署环境一起视为安全策略的一部分。

本文所述安全保证面向原生 Go 平台。Go 标准库明确说明 `GOOS=js` 上的 `os.Root` 存在符号链接校验的 TOCTOU 竞争风险，因此不能在该平台上把它当作安全边界。

## 验证覆盖

回归测试覆盖了 fail-closed 配置、读写根分离、路径穿越和绝对路径拒绝、符号链接逃逸、已检测到的最终符号链接、大小上限、父目录创建、覆盖策略、审计事件、Windows 特殊路径，以及 sandboxed `filegen` 创建流程。

运行聚焦验证：

```bash
go test ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/... -v
go vet ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/...
```

进一步阅读：[Tools 指南](/zh/guide/tools)、[Tools API 参考](/zh/api/tools)，以及[关于根绑定文件沙盒的工程文章](/zh/blog/sandboxed-file-io)。
