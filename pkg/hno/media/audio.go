package media

import "errors"

// AudioInfo contains basic audio metadata
type AudioInfo struct {
    Format      string  `json:"format"`
    DurationSec float64 `json:"duration_sec"`
    SampleRate  int     `json:"sample_rate"`
    Channels    int     `json:"channels"`
}

var ErrAudioProbeUnsupported = errors.New("audio probing not implemented")

// ProbeAudio is a placeholder for audio metadata extraction.
// Minimal stub to satisfy media processing requirement; extend with formats as needed.
func ProbeAudio([]byte) (AudioInfo, error) {
    return AudioInfo{}, ErrAudioProbeUnsupported
}

