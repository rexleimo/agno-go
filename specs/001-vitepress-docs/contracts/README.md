# Contracts: 官方文档站（用户文档）

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`

本目录用于描述与官方文档站相关的“文档契约”，而非新增运行时代码的 API 契约。其目标是确保：

- 文档页面结构与导航在 en/zh/ja/ko 四种语言中保持一致；  
- 文档中展示的 HTTP 接口与行为与 Go AgentOS 现有契约与 fixtures 一致；  
- 所有示例代码遵守“不出现维护者本机绝对路径”的约束。

## 文件一览

- `docs-site-openapi.yaml`：以 OpenAPI 形式建模“文档可见的 HTTP 接口与主要页面”，用于校对文档中出现的 URL 与请求/响应形状。  

未来如需细化，可在本目录下增加：

- 针对关键页面（例如 Quickstart、供应商矩阵、高级案例）的结构约束文件；  
- 与 `specs/001-go-agno-rewrite/contracts/fixtures/` 对齐的示例请求/响应片段。  

