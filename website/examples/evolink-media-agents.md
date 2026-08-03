# EvoLink Media Agents

## Overview
This example shows how to use the EvoLink provider inside Agno-Go to generate text, images, and video via `pkg/hno/models/evolink/*`.

## Setup
- Export env variables:
```bash
export EVO_API_KEY=sk-your-evolink-key
export EVO_BASE_URL=https://api.evolink.ai # optional, defaults to this value
```

## Supported image models
All EvoLink image endpoints require an explicit `model` identifier. The provider currently exposes:

| Model ID | Description | Typical use |
| --- | --- | --- |
| `gpt-4o-image` | GPT-4o multi-modal image generation | High-fidelity creative renders, supports references + mask |
| `doubao-seedream-4.0` | Seedream 4.0 (Doubao) | Stylized illustration, anime aesthetics |
| `gemini-2.5-flash-image` | Nano Banana (Gemini Flash) | Fast drafts, low-latency ideation |
| `qwen-image-edit` | Qwen image editing | Mask-based edits, restoration |
| `wan2.5-text-to-image` | Wan2.5 TTI | Photorealistic text-to-image |
| `wan2.5-image-to-image` | Wan2.5 I2I | Reference-driven variation / style transfer |

Use `InvokeRequest.Extra["model"]` or the model config default to switch among them.

## Supported video models
All video endpoints also require a `model` value matching the EvoLink docs:

| Model ID | Description | Typical use |
| --- | --- | --- |
| `veo3.1-fast` | Veo 3.1 Fast text/image to video | Fast previews with Veo 3.1 |
| `sora-2` | Sora-2 base model | High-fidelity cinematic clips |
| `sora-2-pro` | Sora-2 Pro | Extended capabilities + quality |
| `wan2.5-text-to-video` | Wan2.5 TTV | Photorealistic videos from prompts |
| `wan2.5-image-to-video` | Wan2.5 ITV | First-frame/reference driven videos |
| `doubao-seedance-1.0-pro-fast` | Seedance 1.0 Pro Fast | Stylized, dance-like motion (Doubao) |

Video model quick tips:
- `image_urls` (array) provide reference frames. Wan2.5 ITV requires exactly one URL; Veo accepts up to two; Sora/Seedance accept at most one.
- `duration` depends on the model: Sora `{10,15}`, Wan2.5 `{5,10}`, Seedance `2–12`. Leave it empty to use EvoLink defaults.
- `quality` applies to Sora-2-Pro (`standard|high`) and Seedance (`720p|1080p`).
- `remove_watermark` defaults to `true` on Sora variants—set the extra to `false` to keep the watermark.
- `aspect_ratio` has model-specific limits. Wan2.5 image-to-video derives its ratio from the input and ignores the field altogether.
- `callback_url` must be HTTPS; EvoLink POSTs completion/failure/cancel events there.

## Text (Chat Completions)
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
  resp, _ := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{ types.NewUserMessage("Write a 3-shot underwater city storyboard") }})
  fmt.Println(resp.Content)
}
```

## Images (GPT-4O Images)
Allowed sizes: `1:1`, `2:3`, `3:2`, `1024x1024`, `1024x1536`, `1536x1024`. `n ∈ {1,2,4}`. Up to 5 references. If `mask_url` is provided, exactly one reference is required and the mask must be a PNG.

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
  resp, _ := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{ types.NewUserMessage("A futuristic city made of coral") }})
  fmt.Printf("task: %s, data: %#v\n", resp.ID, resp.Metadata.Extra["data"])
}
```

## Video (Sora-2)
Sora accepts aspect ratios `16:9` or `9:16`, durations `10` or `15` seconds, an optional single `image_urls` entry for image-to-video, and an optional HTTPS `callback_url`. Watermarks are removed by default; send `remove_watermark=false` when you need to keep them.

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
    Messages: []*types.Message{ types.NewUserMessage("A marine biologist swims with manta rays at dawn") },
    Extra: map[string]interface{}{
      "image_urls": []string{"https://assets.example.com/keyframes/manta.png"},
    },
  }
  resp, _ := v.Invoke(context.Background(), req)
  fmt.Printf("task: %s, status: %v\n", resp.ID, resp.Metadata.Extra["status"])
}
```

## Notes
- Assets may expire after 24h; persist them if needed.
- EvoLink enforces strict moderation and HTTPS-only webhooks.
- Reference docs: GPT-4O images: https://docs.evolink.ai/en/api-manual/image-series/gpt-4o/gpt-4o-image-generation. Local Sora-2 HTML reference: `/home/rex/下载/Sora-2 视频生成 - EvoLink.html`.
