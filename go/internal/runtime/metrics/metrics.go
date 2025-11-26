package metrics

import "time"

// Recorder captures runtime metrics for latency, tokens, and resource usage.
// Implementations may forward to Prometheus/OTel; default is no-op.
type Recorder interface {
	ObserveModelLatency(provider string, model string, duration time.Duration)
	ObserveToolLatency(tool string, duration time.Duration)
	ObserveTokens(provider string, prompt int, completion int)
	ObserveMemory(bytes uint64)
}

// NoopRecorder is the default when no metrics backend is wired.
type NoopRecorder struct{}

func (NoopRecorder) ObserveModelLatency(string, string, time.Duration) {}
func (NoopRecorder) ObserveToolLatency(string, time.Duration)          {}
func (NoopRecorder) ObserveTokens(string, int, int)                    {}
func (NoopRecorder) ObserveMemory(uint64)                              {}
