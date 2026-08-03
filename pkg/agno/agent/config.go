package agent

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/cache"
	"github.com/rexleimo/agno-go/pkg/agno/hooks"
	"github.com/rexleimo/agno-go/pkg/agno/memory"
	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

type Config struct {
	ID            string
	Name          string
	Model         models.Model
	Toolkits      []toolkit.Toolkit
	Memory        memory.Memory
	Instructions  string
	MaxLoops      int
	UserID        string       // User ID for multi-tenant scenarios / 多租户场景的用户ID
	PreHooks      []hooks.Hook // Hooks to execute before processing input
	PostHooks     []hooks.Hook // Hooks to execute after generating output
	Logger        *slog.Logger
	EnableCache   bool
	CacheProvider cache.Provider
	CacheTTL      time.Duration

	// Storage control flags (nil means use default: true) / 存储控制标志 (nil 表示使用默认值: true)
	// StoreToolMessages controls whether tool-related messages are included in RunOutput.
	// When false, tool messages and tool-related fields are filtered from output.
	// StoreToolMessages 控制是否在 RunOutput 中包含工具相关消息
	// 当为 false 时，工具消息和工具相关字段会从输出中过滤
	StoreToolMessages *bool

	// StoreHistoryMessages controls whether historical messages (from Memory) are included in RunOutput.
	// When false, only messages generated during the current Run are included.
	// StoreHistoryMessages 控制是否在 RunOutput 中包含历史消息(来自 Memory)
	// 当为 false 时，仅包含当前 Run 生成的消息
	StoreHistoryMessages *bool
}

// New creates a new agent
func New(config Config) (*Agent, error) {
	if config.Model == nil {
		return nil, types.NewInvalidConfigError("model is required", nil)
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("agent-%s", config.Model.GetID())
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	if config.Memory == nil {
		config.Memory = memory.NewInMemory(100)
	}

	if config.MaxLoops <= 0 {
		config.MaxLoops = 10
	}

	if config.Logger == nil {
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	var cacheProvider cache.Provider
	if config.EnableCache {
		cacheProvider = config.CacheProvider
		if cacheProvider == nil {
			provider, err := cache.NewMemoryProvider(0, config.CacheTTL)
			if err != nil {
				return nil, err
			}
			cacheProvider = provider
		}
	}

	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}

	// Helper function to get bool value or default / 辅助函数获取布尔值或默认值
	boolOrDefault := func(ptr *bool, defaultVal bool) bool {
		if ptr == nil {
			return defaultVal
		}
		return *ptr
	}

	agent := &Agent{
		ID:           config.ID,
		Name:         config.Name,
		Model:        config.Model,
		Toolkits:     config.Toolkits,
		Memory:       config.Memory,
		Instructions: config.Instructions,
		MaxLoops:     config.MaxLoops,
		UserID:       config.UserID,
		PreHooks:     config.PreHooks,
		PostHooks:    config.PostHooks,
		logger:       config.Logger,
		cache:        cacheProvider,
		cacheTTL:     cacheTTL,
		cacheEnabled: config.EnableCache && cacheProvider != nil,

		// Storage control (default to true for backward compatibility) / 存储控制 (默认为 true 以保持向后兼容)
		storeToolMessages:    boolOrDefault(config.StoreToolMessages, true),
		storeHistoryMessages: boolOrDefault(config.StoreHistoryMessages, true),
	}

	// Add system message if instructions provided
	// 如果提供了指令则添加系统消息
	if config.Instructions != "" {
		agent.Memory.Add(types.NewSystemMessage(config.Instructions), config.UserID)
	}

	return agent, nil
}
