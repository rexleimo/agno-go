# EvoLink 媒体智能体（Media Agents）

## 概览
本文演示如何在 HNO 中使用 EvoLink Provider，通过 `pkg/hno/models/evolink/*` 生成文本、图片与视频。

## 环境准备
- 设置环境变量：
```bash
export EVO_API_KEY=sk-your-evolink-key
export EVO_BASE_URL=https://api.evolink.ai # 可选，默认即为该值
```

## 支持的图像模型
所有图像接口都必须显式设置 `model` 字段，可选值如下：

| Model ID | 能力说明 | 典型场景 |
| --- | --- | --- |
| `gpt-4o-image` | GPT-4o 多模态渲染，支持参考图与遮罩 | 高保真创作、故事板 |
| `doubao-seedream-4.0` | Seedream 4.0（Doubao） | 二次元、插画风 |
| `gemini-2.5-flash-image` | Nano Banana（Gemini Flash） | 快速草图、低延迟创意 |
| `qwen-image-edit` | Qwen 图像编辑 | 遮罩编辑、修复 |
| `wan2.5-text-to-image` | Wan2.5 文生图 | 写实照片、写实场景 |
| `wan2.5-image-to-image` | Wan2.5 图生图 | 参考图变体、风格迁移 |

可通过模型配置默认值或 `InvokeRequest.Extra["model"]` 动态切换。

## 支持的视频模型
视频接口同样必须提供 `model` 字段，当前可选：

| Model ID | 能力说明 | 典型场景 |
| --- | --- | --- |
| `veo3.1-fast` | Veo 3.1 Fast 文/图生视频 | 快速预览、创意草稿 |
| `sora-2` | Sora-2 基础版 | 高质量电影感片段 |
| `sora-2-pro` | Sora-2 Pro | 更强画质与控制能力 |
| `wan2.5-text-to-video` | Wan2.5 文生视频 | 写实视频生成 |
| `wan2.5-image-to-video` | Wan2.5 图生视频 | 参考图驱动的视频 |
| `doubao-seedance-1.0-pro-fast` | Seedance 1.0 Pro Fast | Doubao 系列，动感舞蹈风格 |

视频模型提示：
- `image_urls`（数组）用于提供参考帧。Wan2.5 图生视频必须提供 1 张；Veo 最多支持 2 张；Sora/Seedance 最多 1 张。
- `duration` 取值因模型而异：Sora `10/15` 秒，Wan2.5 `5/10` 秒，Seedance `2–12` 秒。留空则采用官方默认。
- `quality` 只在 Sora-2-Pro（`standard|high`）与 Seedance（`720p|1080p`）生效。
- `remove_watermark` 在 Sora 默认为 `true`（去水印），若需保留请传 `false`。
- `aspect_ratio` 也因模型不同而限制。Wan2.5 图生视频由参考图自动决定，将忽略该字段。
- `callback_url` 需为 HTTPS，EvoLink 会在任务完成/失败/取消时回调。

## 文本（Chat Completions）
```go
package main

import (
  "context"
  "fmt"
  "os"

  evoText "github.com/rexleimo/agno-go/pkg/hno/models/evolink/text"
  "github.com/rexleimo/agno-go/pkg/hno/models"
  "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
  m, _ := evoText.New("evo-gpt-4o", evoText.Config{APIKey: os.Getenv("EVO_API_KEY"), BaseURL: os.Getenv("EVO_BASE_URL"), Temperature: 0.6})
  resp, _ := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{ types.NewUserMessage("写一段 3 个镜头的海底城市分镜") }})
  fmt.Println(resp.Content)
}
```

## 图片（GPT-4O Images）
支持尺寸：`1:1`、`2:3`、`3:2`、`1024x1024`、`1024x1536`、`1536x1024`。`n ∈ {1,2,4}`。最多 5 个参考图。当提供 `mask_url` 时，必须仅有 1 张参考图，且遮罩必须为 PNG。

```go
package main

import (
  "context"
  "fmt"
  "os"

  evoImg "github.com/rexleimo/agno-go/pkg/hno/models/evolink/image"
  "github.com/rexleimo/agno-go/pkg/hno/models"
  "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
  m, _ := evoImg.New("evo-gpt-4o-images", evoImg.Config{APIKey: os.Getenv("EVO_API_KEY"), BaseURL: os.Getenv("EVO_BASE_URL"), Size: "1:1", N: 2, Model: evoImg.ModelGPT4O})
  resp, _ := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{ types.NewUserMessage("由珊瑚构成的未来主义城市") }})
  fmt.Printf("task: %s, data: %#v\n", resp.ID, resp.Metadata.Extra["data"])
}
```

## 视频（Sora-2）
Sora 支持 `16:9`/`9:16` 宽高比与 `10`/`15` 秒时长，允许传入单张 `image_urls` 参考图以及 HTTPS `callback_url`。默认会去除水印，如需保留请传 `remove_watermark=false`。

```go
package main

import (
  "context"
  "fmt"
  "os"

  evoVid "github.com/rexleimo/agno-go/pkg/hno/models/evolink/video"
  "github.com/rexleimo/agno-go/pkg/hno/models"
  "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
  v, _ := evoVid.New("evo-sora-2", evoVid.Config{APIKey: os.Getenv("EVO_API_KEY"), BaseURL: os.Getenv("EVO_BASE_URL"), AspectRatio: "9:16", DurationSeconds: 15, Model: evoVid.ModelSora2})
  req := &models.InvokeRequest{
    Messages: []*types.Message{ types.NewUserMessage("清晨海洋生物学家与蝠鲼同游") },
    Extra: map[string]interface{}{
      "image_urls": []string{"https://assets.example.com/keyframes/manta.png"},
    },
  }
  resp, _ := v.Invoke(context.Background(), req)
  fmt.Printf("task: %s, status: %v\n", resp.ID, resp.Metadata.Extra["status"])
}
```

## 注意事项
- 生成的资源可能在 24 小时后过期，必要时请自行持久化。
- EvoLink 有严格的内容审核；Webhook 回调仅支持 HTTPS。
- 参考文档：GPT-4O 图片：https://docs.evolink.ai/en/api-manual/image-series/gpt-4o/gpt-4o-image-generation。本地 Sora-2 HTML 参考：`/home/rex/下载/Sora-2 视频生成 - EvoLink.html`。
