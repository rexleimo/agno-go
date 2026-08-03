// protocolsim 协议模拟器：暴露 Anthropic Messages API / Gemini 端点，
// 转发给 OpenAI 兼容后端（llama.cpp）。
// 用途：本地真实协议验证（无需云端 key）
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rexleimo/agno-go/internal/protocolsim"
)

func main() {
	var (
		listenAddr   = flag.String("listen", "127.0.0.1:16000", "listen address for the simulator")
		backendURL   = flag.String("backend", "http://127.0.0.1:18080/v1", "OpenAI-compatible backend base URL")
		backendKey   = flag.String("api-key", "local-test", "backend API key (if required)")
		backendModel = flag.String("model", "qwen3-4b", "model name sent to the backend")
	)
	flag.Parse()

	backend := protocolsim.Backend{
		BaseURL: *backendURL,
		APIKey:  *backendKey,
		Model:   *backendModel,
	}

	mux := http.NewServeMux()
	// Anthropic Messages API at /v1/messages
	// Anthropic Messages API 位于 /v1/messages
	mux.Handle("/v1/messages", protocolsim.NewAnthropicServer(backend))
	// Gemini generateContent API at /v1beta/models/{model}:generateContent
	// Gemini generateContent API 位于 /v1beta/models/{model}:generateContent
	mux.Handle("/v1beta/models/", protocolsim.NewGeminiServer(backend))
	// OpenAI Responses API at /v1/responses
	// OpenAI Responses API 位于 /v1/responses
	mux.Handle("/v1/responses", protocolsim.NewResponsesServer(backend))

	srv := &http.Server{Addr: *listenAddr, Handler: mux}

	go func() {
		log.Printf("协议模拟器已启动: http://%s/v1/messages (Anthropic 格式)", *listenAddr)
		log.Printf("后端: %s (model=%s)", *backendURL, *backendModel)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown / 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("\n正在关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
