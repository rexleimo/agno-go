package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    together "github.com/rexleimo/agno-go/pkg/hno/models/together"
)

// Minimal example showing Together AI model instantiation.
// Note: Requires TOGETHER_API_KEY at runtime to actually call the API.
func main() {
    model, err := together.New("meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", together.Config{APIKey: "YOUR_TOGETHER_API_KEY"})
    if err != nil { panic(err) }

    ag, err := agent.New(agent.Config{ Name: "Together Agent", Model: model })
    if err != nil { panic(err) }

    // This would call the API if a valid key is provided
    out, err := ag.Run(context.Background(), "Say hello in one short sentence.")
    if err != nil { fmt.Println("run error:", err); return }
    fmt.Println(out.Content)
}

