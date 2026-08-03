package media

import "errors"

// VideoInfo contains basic video metadata
type VideoInfo struct {
    Format      string  `json:"format"`
    DurationSec float64 `json:"duration_sec"`
    Width       int     `json:"width"`
    Height      int     `json:"height"`
}

var ErrVideoProbeUnsupported = errors.New("video probing not implemented")

// ProbeVideo is a placeholder for video metadata extraction.
func ProbeVideo([]byte) (VideoInfo, error) {
    return VideoInfo{}, ErrVideoProbeUnsupported
}

