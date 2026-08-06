---
title: "Agent 文件工具不能只靠路径前缀：HNO 如何用根句柄做沙盒"
description: "HNO 通过显式能力、Go root-bound 文件系统句柄、读写限额与独立部署隔离层，约束 Agent 的文件读写。"
date: 2026-08-06
lastUpdated: 2026-08-06
author: HNO Team
category: 安全工程
tags:
  - AI Agent
  - Agent 安全
  - 沙盒
  - 文件 I/O
  - Go
head:
  - - meta
    - name: keywords
      content: "AI Agent 文件沙盒, 安全 Agent 文件 I/O, Go os.Root, Agent 工具安全, 路径名约束"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "Agent 文件工具不能只靠路径前缀：HNO 如何用根句柄做沙盒"
  - - meta
    - property: og:description
      content: "HNO 如何通过显式根目录和 root-bound 文件系统操作约束 Agent 文件工具。"
  - - meta
    - property: article:published_time
      content: "2026-08-06T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/zh/blog/sandboxed-file-io
---

# Agent 文件工具不能只靠路径前缀：HNO 如何用根句柄做沙盒

能读取报告、检查模板、生成产物的 Agent 很有价值；能解析任意宿主机路径的 Agent，则是在等待安全边界失效。

这两种实现很容易被混淆，因为它们暴露的函数名几乎相同：`read_file`、`write_file`、`create_file`。真正重要的问题不是函数叫什么，而是：**当不可信模型给出一个路径后，文件系统最终授权了什么？**

HNO 的新沙盒化文件 I/O 通过显式读写能力、根相对 Agent 路径，以及 Go 的根句柄绑定文件系统操作来回答这个问题。

<img src="/diagrams/sandboxed-file-io.svg" alt="HNO 沙盒化文件 I/O 架构：Agent 工具调用经过能力策略和 root-bound os.Root 句柄，最终到达独立的读根和写根。" />

## 路径前缀白名单的陷阱

第一版实现通常是这样的：

1. 将不可信路径转为绝对路径；
2. 将它与允许目录进行比较；
3. 对该路径调用 `os.ReadFile` 或 `os.WriteFile`。

它能挡住不少明显的 `../` 情况，但对于 Agent 运行时仍然不够。符号链接可以改变已检查名称最终解析的位置；检查之后、不受限制的 I/O 调用之前，文件系统状态也可能发生变化。Windows 还存在盘符相对路径、UNC、设备名和备用数据流等特殊形式。

这里的结论不是输入校验没有用，而是：校验应该用于尽早拒绝与清晰报错；真正执行文件操作的原语必须承担安全边界。

## 架构：先判能力，再用根句柄强制执行

HNO 用显式根目录创建 sandbox：

```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./inputs", "./templates"),
    file.WithWriteRoot("./workspace"),
    file.WithMaxReadBytes(10<<20),
    file.WithMaxWriteBytes(5<<20),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()

readWriteTools := file.NewWithSandbox(sandbox)
generationTools := filegen.NewWithSandbox(sandbox)
```

调用流分为三层：

1. **Toolkit 层**提供熟悉的 Agent 工具。`file.FileTools` 负责读取、列目录、元数据、PPTX 提取、写入和删除；`filegen` 负责生成产物。
2. **Sandbox 策略层**判断一次请求是否具备读或写能力，拒绝不安全的根相对名称，应用字节上限，控制覆盖，并可以输出审计记录。
3. **`os.Root` 句柄层**执行真正的 `Open`、`Stat`、`OpenFile`、`Mkdir`、`OpenRoot` 和 `Remove`。一个已经解析的路径不能通过指向根外的符号链接逃离该句柄代表的目录树。

Agent 传入的路径不是宿主机路径，而是相对于配置根目录的名称，例如 `reports/summary.md`。

## 读和写是不同的授权

读根不会悄悄变成写权限，写根也不会悄悄变成读权限。这使得更小、更准确的 Agent 能力组合成为可能：

- 审阅 Agent 可以读取 `./inputs` 和 `./templates`，但无法修改它们；
- 生成 Agent 只能写入 `./workspace`；
- 每次运行可使用私有的按租户工作区，而不暴露宿主机 checkout 或用户主目录；
- 没有配置根目录的 sandbox 默认拒绝所有操作。

多个读根按配置顺序检查，第一个包含目标根相对名称的读根会被使用。写根只有一个，从而让生成产物落到唯一且明确的目标位置。

## 生成文件还需要第二道覆盖确认

`filegen.create_file` 有 `overwrite` 工具参数，因为创建新产物和替换已有文件是实质不同的操作。

Sandbox 只有在以下两个独立条件同时满足时才允许替换：

- 工具调用明确写了 `overwrite: true`；并且
- 宿主机配置了 `file.WithAllowOverwrite(true)`。

这防止模型仅仅因为应用在技术上“可以覆盖”就替换文件。默认情况下，新文件通过排他创建。为兼容 Go 1.24，获授权的覆盖采用 root-bound 截断并写入，而不是原子 rename；关键发布流程应先生成和校验新产物，再由更高层、受控的发布步骤进行提升或替换。

## Sandbox 不能替代什么

此能力约束的是**文件工具的路径解析**，并不会让任意代码执行变得安全。它不限制出站网络、CPU、内存、进程、挂载点，也不会自动消除授权根目录中已经存在的特权资源。

对于不可信或多租户工作负载，应把文件 sandbox 放进纵深防御体系：

- 每次运行或每个租户使用私有卷/目录树；
- 最小权限服务账号和文件系统 ACL；
- 对进程使用容器、虚拟机或专用执行 sandbox；
- 内存、CPU、进程数和磁盘配额；
- egress 控制与持久审计日志。

这个边界说明与实现本身同样重要。路径 sandbox 擅长在仔细选择根目录后防止路径逃逸；它不是把敏感宿主机文件系统挂载给不可信执行环境的理由。

## 可测试的安全契约

回归套件覆盖的不是只有 happy path，而是完整操作契约：fail-closed 启动、能力分离、路径穿越、绝对路径与 Windows 特殊路径、符号链接逃逸、最终链接目标、读写上限、父目录创建、审计事件，以及 `filegen` 的覆盖规则。

运行聚焦检查：

```bash
go test ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/... -v
go vet ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/...
```

完整 API 契约、部署建议与架构讲解请见[沙盒化文件 I/O 指南](/zh/guide/sandboxed-file-io)。