package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	openaimodel "github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

type sample struct {
	DurationNS int64  `json:"duration_ns"`
	Content    string `json:"content,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

var remoteCachePrefix = strings.Repeat(
	"You are a deterministic benchmark assistant. Follow the user instruction exactly. "+
		"Keep the answer concise and do not add explanations. ",
	80,
)

type report struct {
	Framework         string   `json:"framework"`
	Scenario          string   `json:"scenario"`
	Endpoint          string   `json:"endpoint"`
	Model             string   `json:"model"`
	Warmup            int      `json:"warmup"`
	Runs              int      `json:"runs"`
	Concurrency       int      `json:"concurrency"`
	Lifecycle         string   `json:"lifecycle"`
	MeasuredElapsedNS int64    `json:"measured_elapsed_ns"`
	Samples           []sample `json:"samples"`
}

func normalizeBaseURL(endpoint string) string {
	base := strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	return base + "/"
}

func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if key != "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func promptFor(scenario string) string {
	prefix := "/no_think\n"
	if os.Getenv("AGNES_API_KEY") != "" {
		prefix = remoteCachePrefix + "\n"
	}
	switch scenario {
	case "simple":
		expected := "LOCAL_MODEL_OK"
		if os.Getenv("AGNES_API_KEY") != "" {
			expected = "REMOTE_MODEL_OK"
		}
		return prefix + "Reply with exactly: " + expected
	case "tool":
		return prefix + "Use the add tool to calculate 25 + 17. After the tool returns, reply with exactly: RESULT_42"
	case "memory":
		return prefix + "Remember this code exactly: BLUE-42. Reply with ACK only."
	default:
		return ""
	}
}

func addOnlyToolkit() toolkit.Toolkit {
	t := toolkit.NewBaseToolkit("calculator")
	t.RegisterFunction(&toolkit.Function{
		Name:        "add",
		Description: "Add two numbers together",
		Parameters: map[string]toolkit.Parameter{
			"a": {Type: "number", Description: "First number", Required: true},
			"b": {Type: "number", Description: "Second number", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			a, okA := args["a"].(float64)
			b, okB := args["b"].(float64)
			if !okA || !okB {
				return nil, fmt.Errorf("add expects numeric arguments")
			}
			return a + b, nil
		},
	})
	return t
}

func newModel(endpoint, modelID string) (*openaimodel.OpenAI, error) {
	seed := 42
	apiKey := os.Getenv("AGNES_API_KEY")
	if apiKey == "" {
		apiKey = "local"
	}
	return openaimodel.New(modelID, openaimodel.Config{
		APIKey:         apiKey,
		BaseURL:        normalizeBaseURL(endpoint),
		Temperature:    0,
		TemperatureSet: true,
		Seed:           &seed,
		MaxTokens:      128,
		Timeout:        120 * time.Second,
	})
}

func runOnce(ctx context.Context, model *openaimodel.OpenAI, scenario string) sample {
	config := agent.Config{
		Name:     "local-benchmark-agent",
		Model:    model,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxLoops: 4,
	}
	if scenario == "tool" {
		config.Toolkits = []toolkit.Toolkit{addOnlyToolkit()}
	}

	started := time.Now()
	ag, err := agent.New(config)
	if err != nil {
		return sample{DurationNS: time.Since(started).Nanoseconds(), Error: err.Error()}
	}
	output, err := agentOutput(ctx, ag, scenario)
	result := sample{DurationNS: time.Since(started).Nanoseconds()}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Content = output
	result.Success = successFor(scenario, output)
	return result
}

func runOnceFresh(ctx context.Context, endpoint, modelID, scenario string) sample {
	started := time.Now()
	model, err := newModel(endpoint, modelID)
	if err != nil {
		return sample{DurationNS: time.Since(started).Nanoseconds(), Error: err.Error()}
	}
	result := runOnce(ctx, model, scenario)
	result.DurationNS = time.Since(started).Nanoseconds()
	return result
}

func runConcurrent(ctx context.Context, model *openaimodel.OpenAI, scenario string, runs, concurrency int) []sample {
	return runConcurrentFunc(ctx, runs, concurrency, func() sample {
		return runOnce(ctx, model, scenario)
	})
}

func runConcurrentFresh(ctx context.Context, endpoint, modelID, scenario string, runs, concurrency int) []sample {
	return runConcurrentFunc(ctx, runs, concurrency, func() sample {
		return runOnceFresh(ctx, endpoint, modelID, scenario)
	})
}

func runConcurrentFunc(ctx context.Context, runs, concurrency int, run func() sample) []sample {
	if runs <= 0 {
		return []sample{}
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > runs {
		concurrency = runs
	}

	samples := make([]sample, runs)
	jobs := make(chan int)
	var workers sync.WaitGroup
	workers.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				samples[index] = run()
			}
		}()
	}
	for i := 0; i < runs; i++ {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	return samples
}

func agentOutput(ctx context.Context, ag *agent.Agent, scenario string) (string, error) {
	if scenario == "memory" {
		if _, err := ag.Run(ctx, promptFor("memory")); err != nil {
			return "", err
		}
		prefix := "/no_think\n"
		if os.Getenv("AGNES_API_KEY") != "" {
			prefix = remoteCachePrefix + "\n"
		}
		output, err := ag.Run(ctx, prefix+"What code did I ask you to remember? Reply with the code only.")
		if err != nil {
			return "", err
		}
		return output.Content, nil
	}
	output, err := ag.Run(ctx, promptFor(scenario))
	if err != nil {
		return "", err
	}
	return output.Content, nil
}

func successFor(scenario, content string) bool {
	upper := strings.ToUpper(content)
	switch scenario {
	case "simple":
		expected := "LOCAL_MODEL_OK"
		if os.Getenv("AGNES_API_KEY") != "" {
			expected = "REMOTE_MODEL_OK"
		}
		return strings.Contains(upper, expected)
	case "tool":
		return strings.Contains(upper, "RESULT_42") || strings.Contains(upper, "42")
	case "memory":
		return strings.Contains(upper, "BLUE-42")
	default:
		return false
	}
}

func main() {
	configPath := flag.String("config", "", "optional local env file; secrets are read locally and never printed")
	endpoint := flag.String("endpoint", "", "OpenAI-compatible endpoint")
	modelID := flag.String("model", "", "model id from /v1/models")
	scenario := flag.String("scenario", "simple", "simple, tool, or memory")
	warmup := flag.Int("warmup", 3, "warmup runs")
	runs := flag.Int("runs", 100, "measured runs")
	concurrency := flag.Int("concurrency", 8, "maximum concurrent measured runs")
	lifecycle := flag.String("lifecycle", "shared", "model lifecycle: shared or fresh per operation")
	flag.Parse()
	if *scenario != "simple" && *scenario != "tool" && *scenario != "memory" {
		fmt.Fprintln(os.Stderr, "scenario must be simple, tool, or memory")
		os.Exit(2)
	}
	if *configPath != "" {
		if err := loadEnvFile(*configPath); err != nil {
			fmt.Fprintln(os.Stderr, "failed to load config:", err)
			os.Exit(2)
		}
	}
	if *endpoint == "" {
		*endpoint = os.Getenv("AGNES_BASE_URL")
	}
	if *endpoint == "" {
		*endpoint = "http://127.0.0.1:8081"
	}
	if *modelID == "" {
		*modelID = os.Getenv("AGNES_MODEL")
	}
	if *modelID == "" {
		*modelID = `models\Qwen3-4B-Q8_0.gguf`
	}
	if *lifecycle != "shared" && *lifecycle != "fresh" {
		fmt.Fprintln(os.Stderr, "lifecycle must be shared or fresh")
		os.Exit(2)
	}
	if *warmup < 0 || *runs < 1 || *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "warmup must be non-negative, runs positive, and concurrency positive")
		os.Exit(2)
	}

	var model *openaimodel.OpenAI
	var err error
	if *lifecycle == "shared" {
		model, err = newModel(*endpoint, *modelID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	ctx := context.Background()
	for i := 0; i < *warmup; i++ {
		var result sample
		if *lifecycle == "fresh" {
			result = runOnceFresh(ctx, *endpoint, *modelID, *scenario)
		} else {
			result = runOnce(ctx, model, *scenario)
		}
		if result.Error != "" {
			fmt.Fprintln(os.Stderr, result.Error)
			os.Exit(1)
		}
	}

	result := report{
		Framework:   "hno",
		Scenario:    *scenario,
		Endpoint:    *endpoint,
		Model:       *modelID,
		Warmup:      *warmup,
		Runs:        *runs,
		Concurrency: *concurrency,
		Lifecycle:   *lifecycle,
		Samples:     make([]sample, 0, *runs),
	}
	measuredStarted := time.Now()
	if *lifecycle == "fresh" {
		result.Samples = runConcurrentFresh(ctx, *endpoint, *modelID, *scenario, *runs, *concurrency)
	} else {
		result.Samples = runConcurrent(ctx, model, *scenario, *runs, *concurrency)
	}
	result.MeasuredElapsedNS = time.Since(measuredStarted).Nanoseconds()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
