package glm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// TestParseAPIKey tests the API key parsing function
// TestParseAPIKey 测试 API key 解析函数
func TestParseAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantKeyID string
		wantErr   bool
	}{
		{
			name:      "valid API key",
			apiKey:    "test_key_id.test_key_secret",
			wantKeyID: "test_key_id",
			wantErr:   false,
		},
		{
			name:    "missing separator",
			apiKey:  "invalid_key_format",
			wantErr: true,
		},
		{
			name:    "empty key ID",
			apiKey:  ".test_secret",
			wantErr: true,
		},
		{
			name:    "empty key secret",
			apiKey:  "test_id.",
			wantErr: true,
		},
		{
			name:    "too many parts",
			apiKey:  "part1.part2.part3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyID, keySecret, err := parseAPIKey(tt.apiKey)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAPIKey() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("parseAPIKey() unexpected error = %v", err)
				}
				if keyID != tt.wantKeyID {
					t.Errorf("parseAPIKey() keyID = %v, want %v", keyID, tt.wantKeyID)
				}
				if keySecret == "" {
					t.Errorf("parseAPIKey() keySecret is empty")
				}
			}
		})
	}
}

// TestGenerateJWT tests the JWT generation function
// TestGenerateJWT 测试 JWT 生成函数
func TestGenerateJWT(t *testing.T) {
	keyID := "test_key_id"
	keySecret := "test_key_secret"

	token, err := generateJWT(keyID, keySecret)
	if err != nil {
		t.Fatalf("generateJWT() error = %v", err)
	}

	if token == "" {
		t.Fatalf("generateJWT() returned empty token")
	}

	// Verify token can be parsed
	// 验证令牌可以被解析
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(keySecret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse generated JWT: %v", err)
	}

	if !parsed.Valid {
		t.Errorf("Generated JWT is not valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("Failed to extract claims from JWT")
	}

	// Verify claims
	// 验证声明
	if claims["api_key"] != keyID {
		t.Errorf("JWT api_key = %v, want %v", claims["api_key"], keyID)
	}

	if claims["timestamp"] == nil {
		t.Errorf("JWT timestamp is missing")
	}

	if claims["exp"] == nil {
		t.Errorf("JWT exp is missing")
	}

	// Verify header
	// 验证头
	if parsed.Header["alg"] != "HS256" {
		t.Errorf("JWT alg = %v, want HS256", parsed.Header["alg"])
	}

	if parsed.Header["sign_type"] != "SIGN" {
		t.Errorf("JWT sign_type = %v, want SIGN", parsed.Header["sign_type"])
	}
}

// TestNew tests the GLM constructor
// TestNew 测试 GLM 构造函数
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modelID: "glm-4",
			config:  Config{APIKey: "test_id.test_secret"},
			wantErr: false,
		},
		{
			name:    "missing API key",
			modelID: "glm-4",
			config:  Config{},
			wantErr: true,
			errMsg:  "API key is required",
		},
		{
			name:    "malformed API key",
			modelID: "glm-4",
			config:  Config{APIKey: "invalid-key-format"},
			wantErr: true,
			errMsg:  "invalid GLM API key",
		},
		{
			name:    "with custom base URL",
			modelID: "glm-4",
			config: Config{
				APIKey:  "test_id.test_secret",
				BaseURL: "https://custom.example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.modelID, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("New() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("New() unexpected error = %v", err)
				}
				if got == nil {
					t.Errorf("New() returned nil")
				}
				if got != nil {
					if got.GetID() != tt.modelID {
						t.Errorf("New() model ID = %v, want %v", got.GetID(), tt.modelID)
					}
					if got.GetProvider() != "glm" {
						t.Errorf("New() provider = %v, want glm", got.GetProvider())
					}
					if tt.config.BaseURL != "" && got.config.BaseURL != tt.config.BaseURL {
						t.Errorf("New() baseURL = %v, want %v", got.config.BaseURL, tt.config.BaseURL)
					}
					if tt.config.BaseURL == "" && got.config.BaseURL != DefaultBaseURL {
						t.Errorf("New() baseURL = %v, want default %v", got.config.BaseURL, DefaultBaseURL)
					}
				}
			}
		})
	}
}

// TestBuildGLMRequest tests the request building function
// TestBuildGLMRequest 测试请求构建函数
func TestBuildGLMRequest(t *testing.T) {
	glm, err := New("glm-4", Config{
		APIKey:      "test_id.test_secret",
		Temperature: 0.7,
		MaxTokens:   1024,
		TopP:        0.9,
		DoSample:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create GLM: %v", err)
	}

	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleSystem, Content: "You are a helpful assistant."},
			{Role: types.RoleUser, Content: "Hello!"},
		},
		Tools: []models.ToolDefinition{
			{
				Type: "function",
				Function: models.FunctionSchema{
					Name:        "get_weather",
					Description: "Get weather information",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
					},
				},
			},
		},
		Temperature: 0.8,
		MaxTokens:   2048,
	}

	glmReq, err := glm.buildGLMRequest(req, false)
	if err != nil {
		t.Fatalf("buildGLMRequest() error = %v", err)
	}

	// Verify basic fields
	// 验证基本字段
	if glmReq.Model != "glm-4" {
		t.Errorf("buildGLMRequest() model = %v, want glm-4", glmReq.Model)
	}

	if len(glmReq.Messages) != 2 {
		t.Errorf("buildGLMRequest() messages count = %v, want 2", len(glmReq.Messages))
	}

	if glmReq.Stream != false {
		t.Errorf("buildGLMRequest() stream = %v, want false", glmReq.Stream)
	}

	// Verify temperature (should use req.Temperature, not config)
	// 验证温度（应使用 req.Temperature，而非 config）
	if glmReq.Temperature == nil || *glmReq.Temperature != 0.8 {
		t.Errorf("buildGLMRequest() temperature = %v, want 0.8", glmReq.Temperature)
	}

	// Verify max tokens (should use req.MaxTokens, not config)
	// 验证最大 token 数（应使用 req.MaxTokens，而非 config）
	if glmReq.MaxTokens == nil || *glmReq.MaxTokens != 2048 {
		t.Errorf("buildGLMRequest() max_tokens = %v, want 2048", glmReq.MaxTokens)
	}

	// Verify top_p from config
	// 验证来自 config 的 top_p
	if glmReq.TopP == nil || *glmReq.TopP != 0.9 {
		t.Errorf("buildGLMRequest() top_p = %v, want 0.9", glmReq.TopP)
	}

	// Verify do_sample from config
	// 验证来自 config 的 do_sample
	if glmReq.DoSample == nil || *glmReq.DoSample != true {
		t.Errorf("buildGLMRequest() do_sample = %v, want true", glmReq.DoSample)
	}

	// Verify tools
	// 验证工具
	if len(glmReq.Tools) != 1 {
		t.Errorf("buildGLMRequest() tools count = %v, want 1", len(glmReq.Tools))
	}

	if glmReq.ToolChoice != "auto" {
		t.Errorf("buildGLMRequest() tool_choice = %v, want auto", glmReq.ToolChoice)
	}
}

// TestInvoke tests the synchronous API call
// TestInvoke 测试同步 API 调用
func TestInvoke(t *testing.T) {
	// Create mock server
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		// 验证方法
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify path
		// 验证路径
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
		}

		// Verify headers
		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json")
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		// Return mock response
		// 返回模拟响应
		resp := glmResponse{
			ID:      "test-id",
			Created: time.Now().Unix(),
			Model:   "glm-4",
			Choices: []glmChoice{
				{
					Index: 0,
					Message: glmMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: glmUsage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create GLM client with mock server
	// 使用模拟服务器创建 GLM 客户端
	glm, err := New("glm-4", Config{
		APIKey:  "test_id.test_secret",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create GLM: %v", err)
	}

	// Make request
	// 发起请求
	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Hello!"},
		},
	}

	resp, err := glm.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// Verify response
	// 验证响应
	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("Invoke() content = %v, want 'Hello! How can I help you today?'", resp.Content)
	}

	if resp.Usage.TotalTokens != 25 {
		t.Errorf("Invoke() total_tokens = %v, want 25", resp.Usage.TotalTokens)
	}

	if resp.Model != "glm-4" {
		t.Errorf("Invoke() model = %v, want glm-4", resp.Model)
	}
}

// TestInvokeError tests error handling in API calls
// TestInvokeError 测试 API 调用中的错误处理
func TestInvokeError(t *testing.T) {
	// Create mock server that returns error
	// 创建返回错误的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := glmErrorResponse{
			Error: glmError{
				Code:    "invalid_request",
				Message: "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	glm, err := New("glm-4", Config{
		APIKey:  "test_id.test_secret",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create GLM: %v", err)
	}

	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Hello!"},
		},
	}

	_, err = glm.Invoke(context.Background(), req)
	if err == nil {
		t.Errorf("Invoke() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Invoke() error = %v, want error containing 'Invalid API key'", err)
	}
}

// TestConvertToModelResponse tests response conversion
// TestConvertToModelResponse 测试响应转换
func TestConvertToModelResponse(t *testing.T) {
	glm, _ := New("glm-4", Config{APIKey: "test_id.test_secret"})

	glmResp := &glmResponse{
		ID:      "test-id",
		Created: time.Now().Unix(),
		Model:   "glm-4",
		Choices: []glmChoice{
			{
				Index: 0,
				Message: glmMessage{
					Role:    "assistant",
					Content: "Test response",
					ToolCalls: []glmToolCall{
						{
							ID:   "call-123",
							Type: "function",
							Function: glmToolCallFunction{
								Name:      "get_weather",
								Arguments: `{"location":"Beijing"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: glmUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	modelResp := glm.convertToModelResponse(glmResp)

	// Verify basic fields
	// 验证基本字段
	if modelResp.ID != "test-id" {
		t.Errorf("convertToModelResponse() id = %v, want test-id", modelResp.ID)
	}

	if modelResp.Content != "Test response" {
		t.Errorf("convertToModelResponse() content = %v, want 'Test response'", modelResp.Content)
	}

	if modelResp.Model != "glm-4" {
		t.Errorf("convertToModelResponse() model = %v, want glm-4", modelResp.Model)
	}

	// Verify usage
	// 验证使用情况
	if modelResp.Usage.TotalTokens != 30 {
		t.Errorf("convertToModelResponse() total_tokens = %v, want 30", modelResp.Usage.TotalTokens)
	}

	// Verify tool calls
	// 验证工具调用
	if len(modelResp.ToolCalls) != 1 {
		t.Errorf("convertToModelResponse() tool_calls count = %v, want 1", len(modelResp.ToolCalls))
	}

	if modelResp.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("convertToModelResponse() function name = %v, want get_weather", modelResp.ToolCalls[0].Function.Name)
	}

	// Verify metadata
	// 验证元数据
	if modelResp.Metadata.FinishReason != "tool_calls" {
		t.Errorf("convertToModelResponse() finish_reason = %v, want tool_calls", modelResp.Metadata.FinishReason)
	}
}
