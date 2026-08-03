package media

import (
    "bytes"
    "image"
    "image/color"
    "image/png"
    "testing"
)

func TestAnalyzeImage_PNG(t *testing.T) {
    // build a 2x3 PNG in memory
    img := image.NewRGBA(image.Rect(0,0,2,3))
    img.Set(0,0, color.RGBA{255,0,0,255})
    var buf bytes.Buffer
    if err := png.Encode(&buf, img); err != nil {
        t.Fatalf("png encode error: %v", err)
    }
    info, err := AnalyzeImage(bytes.NewReader(buf.Bytes()))
    if err != nil { t.Fatalf("AnalyzeImage error: %v", err) }
    if info.Width != 2 || info.Height != 3 { t.Fatalf("unexpected size: %+v", info) }
    if info.Format != "png" { t.Fatalf("unexpected format: %s", info.Format) }
}

