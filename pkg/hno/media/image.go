package media

import (
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "io"
)

// ImageInfo contains basic image metadata
type ImageInfo struct {
    Format string `json:"format"`
    Width  int    `json:"width"`
    Height int    `json:"height"`
}

// AnalyzeImage reads image metadata (format, dimensions) using image.DecodeConfig
func AnalyzeImage(r io.Reader) (ImageInfo, error) {
    cfg, format, err := image.DecodeConfig(r)
    if err != nil {
        return ImageInfo{}, err
    }
    return ImageInfo{Format: format, Width: cfg.Width, Height: cfg.Height}, nil
}

