package groq

// Common Groq model identifiers
// 常见 Groq 模型标识符
const (
	// LLaMA models - 超快速推理
	ModelLlama38B    = "llama-3.1-8b-instant"    // LLaMA 3.1 8B (fastest)
	ModelLlama370B   = "llama-3.1-70b-versatile" // LLaMA 3.1 70B (most capable)
	ModelLlama3405B  = "llama-3.3-70b-versatile" // LLaMA 3.3 70B
	ModelLlama31405B = "llama-3.3-70b-specdec"   // LLaMA 3.3 70B Speculative Decoding

	// Mixtral models
	ModelMixtral8x7B = "mixtral-8x7b-32768" // Mixtral 8x7B

	// Gemma models
	ModelGemma7B   = "gemma-7b-it"  // Gemma 7B
	ModelGemma2_9B = "gemma2-9b-it" // Gemma 2 9B

	// Whisper models - 语音识别
	ModelWhisperLarge = "whisper-large-v3"       // Whisper Large V3
	ModelWhisperTurbo = "whisper-large-v3-turbo" // Whisper Large V3 Turbo

	// LLaMA Guard - 内容审核
	ModelLlamaGuard3 = "llama-guard-3-8b" // LLaMA Guard 3 8B
)

// ModelInfo contains information about a Groq model
// ModelInfo 包含 Groq 模型的信息
type ModelInfo struct {
	ID            string // Model identifier / 模型标识符
	Name          string // Display name / 显示名称
	ContextWindow int    // Maximum context window size / 最大上下文窗口大小
	Developer     string // Model developer / 模型开发者
	Description   string // Model description / 模型描述
	SupportsTools bool   // Whether the model supports function calling / 是否支持函数调用
}

// AvailableModels returns information about available Groq models
// AvailableModels 返回可用 Groq 模型的信息
var AvailableModels = map[string]ModelInfo{
	ModelLlama38B: {
		ID:            ModelLlama38B,
		Name:          "LLaMA 3.1 8B Instant",
		ContextWindow: 128000,
		Developer:     "Meta",
		Description:   "Ultra-fast inference optimized for speed",
		SupportsTools: true,
	},
	ModelLlama370B: {
		ID:            ModelLlama370B,
		Name:          "LLaMA 3.1 70B Versatile",
		ContextWindow: 128000,
		Developer:     "Meta",
		Description:   "Most capable LLaMA model with balanced speed and quality",
		SupportsTools: true,
	},
	ModelLlama3405B: {
		ID:            ModelLlama3405B,
		Name:          "LLaMA 3.3 70B Versatile",
		ContextWindow: 128000,
		Developer:     "Meta",
		Description:   "Latest LLaMA model with improved capabilities",
		SupportsTools: true,
	},
	ModelMixtral8x7B: {
		ID:            ModelMixtral8x7B,
		Name:          "Mixtral 8x7B",
		ContextWindow: 32768,
		Developer:     "Mistral AI",
		Description:   "Mixture of Experts model for efficient inference",
		SupportsTools: true,
	},
	ModelGemma2_9B: {
		ID:            ModelGemma2_9B,
		Name:          "Gemma 2 9B",
		ContextWindow: 8192,
		Developer:     "Google",
		Description:   "Google's compact but powerful language model",
		SupportsTools: true,
	},
	ModelWhisperLarge: {
		ID:            ModelWhisperLarge,
		Name:          "Whisper Large V3",
		ContextWindow: 0, // Audio model
		Developer:     "OpenAI",
		Description:   "High-quality speech recognition model",
		SupportsTools: false,
	},
	ModelLlamaGuard3: {
		ID:            ModelLlamaGuard3,
		Name:          "LLaMA Guard 3 8B",
		ContextWindow: 8192,
		Developer:     "Meta",
		Description:   "Content moderation and safety model",
		SupportsTools: false,
	},
}

// GetModelInfo returns information about a specific model
// GetModelInfo 返回特定模型的信息
func GetModelInfo(modelID string) (ModelInfo, bool) {
	info, ok := AvailableModels[modelID]
	return info, ok
}
