# Remote model benchmark input

> 请填写本文件中的配置项，但不要把 API Key、Token、密码或 Cookie 写进来。
> Key 只放在本机环境变量中。未填写的可选项保持原样。

## 1. Provider 信息（必填）

| 字段 | 填写值 |
| --- | --- |
| Provider 名称 | `请填写` |
| API 协议 | `OpenAI-compatible` / `Anthropic-compatible` / `其他` |
| API Base URL | `https://api.agnes-ai.cn/v1` |
| Model ID | `agnes-2.5-flash` |
| API Key 环境变量名 | `AGNES_API_KEY` |
| Chat Completions 路径 | `/chat/completions` |
| Models 路径 | `/models` |

### 说明

- `API Base URL` 只填写地址，不要附加 API Key。
- 如果服务不是 OpenAI-compatible，请填写“其他”，并补充请求格式说明。
- `Model ID` 必须填写服务实际接受的模型 ID，不要只填写宣传名称。
- API Key 只填写环境变量名，例如 `AGNES_API_KEY`，不要填写 Key 本身。

## 2. 模型信息

| 字段 | 填写值 |
| --- | --- |
| 模型正式名称 | `请填写` |
| 模型版本或快照 | `请填写` |
| Provider 返回的模型 ID | `请填写` |
| 上下文长度 | `请填写，例如 32768` |
| 是否支持工具调用 | `是` / `否` / `未确认` |
| 是否支持 Seed | `是` / `否` / `未确认` |
| 是否支持 Temperature=0 | `是` / `否` / `未确认` |
| 是否支持流式响应 | `是` / `否` / `未确认` |
| 模型文档 URL | `https://请填写` |

## 3. 认证和网络

| 字段 | 填写值 |
| --- | --- |
| API Key 环境变量名 | `AGNES_API_KEY` |
| 是否需要额外 Header | `否` / `是，见下方` |
| 额外 Header 名称 | `请填写名称，不要填写敏感值` |
| 代理要求 | `无` / `请填写` |
| 请求超时（秒） | `120` |
| 是否允许产生 API 费用 | `是` / `否` |
| 速率限制 | `请填写，例如 60 RPM` |

> 额外 Header 的敏感值不要写入本文件。只填写环境变量名，例如：
>
> ```text
> Header: X-Custom-Key
> Value source: AGNES_CUSTOM_HEADER
> ```

## 4. 公平测试参数

以下参数默认用于 HNO、Agno、LangGraph 三个框架：

| 参数 | 默认值 | 是否修改 |
| --- | ---: | --- |
| Temperature | `0` | `否` / `改为：` |
| Seed | `42` | `否` / `不支持` / `改为：` |
| Max output tokens | `128` | `否` / `改为：` |
| Warmup runs | `3` | `否` / `改为：` |
| Measured runs | `100` | `否` / `改为：` |
| 并发度 | `8` | `否` / `改为：` |
| 是否启用流式测试 | `否` | `是` / `否` |

## 5. 测试场景

### 场景 A：固定问答

默认 Prompt：

```text
Reply with exactly: REMOTE_MODEL_OK
```

期望结果：

```text
REMOTE_MODEL_OK
```

是否启用：`是`

### 场景 B：工具调用

工具名称：`add`

工具 Schema：

```json
{
  "type": "function",
  "function": {
    "name": "add",
    "description": "Add two numbers together",
    "parameters": {
      "type": "object",
      "properties": {
        "a": {
          "type": "number",
          "description": "First number"
        },
        "b": {
          "type": "number",
          "description": "Second number"
        }
      },
      "required": ["a", "b"]
    }
  }
}
```

Prompt：

```text
Use the add tool to calculate 25 + 17.
After the tool returns, reply with exactly: RESULT_42
```

期望结果：

```text
RESULT_42
```

是否启用：`是`

### 场景 C：两轮上下文

第一轮：

```text
Remember this code exactly: BLUE-42. Reply with ACK only.
```

第二轮：

```text
What code did I ask you to remember? Reply with the code only.
```

期望结果：

```text
BLUE-42
```

是否启用：`是`

### 场景 D：自定义业务场景（可选）

场景名称：`请填写`

业务背景：

```text
请填写
```

固定 Prompt：

```text
请填写
```

成功判定：

```text
请填写明确的判定规则
```

## 6. Provider Smoke Test 信息（可选）

请在本机运行测试，但不要把返回的 API Key 或 Authorization Header 粘贴到这里。

```text
GET /health：请填写状态
GET /models：请填写状态
POST /chat/completions：请填写状态
```

实际返回的模型 ID：

```text
请填写
```

Provider 的响应格式备注：

```text
请填写
```

## 7. 费用和发布限制

| 项目 | 填写值 |
| --- | --- |
| 单次请求费用 | `请填写或未知` |
| 是否有每日额度 | `请填写或未知` |
| 是否允许运行 3 次预热 + 100 次正式样本 | `是` / `否` |
| 是否允许重复复测 | `是` / `否` |
| 结果是否允许写入公开文档 | `是` / `否` |
| 需要隐藏的字段 | `请填写字段名，不要填写敏感值` |

## 8. 环境变量检查

请在运行 benchmark 的同一个终端中设置 Key，例如 PowerShell：

```powershell
$env:AGNES_API_KEY = "在这里设置真实 Key，但不要写入本文件或聊天"
$env:AGNES_BASE_URL = "https://请填写/v1"
$env:AGNES_MODEL = "请填写模型 ID"
```

代码只检查环境变量是否存在，不会把值写入：

- 原始 JSON；
- Markdown 报告；
- 日志；
- Git；
- 文档页面。

## 9. 你填完后需要确认的事项

- [ ] 已填写 API Base URL
- [ ] 已填写 Model ID
- [ ] 已填写 API Key 环境变量名
- [ ] Key 已设置在本机环境变量中
- [ ] 没有把 Key、Token 或密码写进本文件
- [ ] 已确认 API 协议
- [ ] 已确认是否支持工具调用
- [ ] 已确认可以进行预热和重复测试
- [ ] 已确认测试可能产生的费用或额度消耗
