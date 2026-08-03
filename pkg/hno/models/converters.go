package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// MessageConverter converts framework messages (types.Message) into a
// provider-specific wire representation. Providers implement this for their
// own API shape; the runner/agent layer never sees provider wire types.
//
// MessageConverter 将框架消息（types.Message）转换为 provider 特有的
// 线上表示。各 provider 为自己的 API 形态实现此接口；runner/agent 层
// 永远看不到 provider 的线上类型。
type MessageConverter interface {
	// ConvertMessages converts framework messages into the provider wire
	// representation (e.g. []ClaudeMessage, []Content for Gemini, or
	// []openai.ChatCompletionMessage). The result is serialized directly
	// into the request body.
	// ConvertMessages 将框架消息转换为 provider 线上表示（如 []ClaudeMessage、
	// Gemini 的 []Content、或 []openai.ChatCompletionMessage）。结果直接
	// 序列化进请求体。
	ConvertMessages(msgs []*types.Message) (interface{}, error)
}

// UsageExtractor extracts a normalized Usage from a provider response.
//
// UsageExtractor 从 provider 响应中提取归一化的 Usage。
type UsageExtractor interface {
	// ExtractUsage returns token usage from a provider response object.
	// ExtractUsage 从 provider 响应对象返回 token 用量。
	ExtractUsage(resp interface{}) types.Usage
}

// FromToolCall converts a types.ToolCall into a provider-agnostic
// ToolDefinition.
//
// FromToolCall 将 types.ToolCall 转换为与 provider 无关的 ToolDefinition。
func FromToolCall(tc types.ToolCall) ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionSchema{
			Name:        tc.Function.Name,
			Description: "",
			Parameters:  nil,
		},
	}
}

// NewToolCall builds a framework ToolCall from provider wire fields.
//
// NewToolCall 从 provider 线上字段构建框架 ToolCall。
func NewToolCall(id, name, arguments string) types.ToolCall {
	return types.ToolCall{
		ID:   id,
		Type: "function",
		Function: types.ToolCallFunction{
			Name:      name,
			Arguments: arguments,
		},
	}
}

// MarshalMap serializes a map into a JSON string, returning "{}" for nil.
//
// MarshalMap 将 map 序列化为 JSON 字符串，nil 时返回 "{}"。
func MarshalMap(m map[string]interface{}) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ParseArguments parses a JSON string of tool arguments into a map.
//
// ParseArguments 将工具参数的 JSON 字符串解析为 map。
func ParseArguments(args string) (map[string]interface{}, error) {
	if args == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(args), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ToInt converts a JSON-decoded value into an int. It handles all numeric
// kinds, strings holding an integer, and json.Number. The second return
// value reports whether the conversion succeeded.
//
// ToInt 将 JSON 解码后的值转换为 int。它处理所有数值类型、包含整数的
// 字符串以及 json.Number。第二个返回值报告转换是否成功。
func ToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if v == "" {
			return 0, false
		}
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

// ToBool converts a JSON-decoded value into a bool. It handles bool,
// strings parseable by strconv.ParseBool, and numeric kinds (nonzero=true).
// The second return value reports whether the conversion succeeded.
//
// ToBool 将 JSON 解码后的值转换为 bool。它处理 bool、可被
// strconv.ParseBool 解析的字符串、以及数值类型（非零为 true）。
// 第二个返回值报告转换是否成功。
func ToBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if v == "" {
			return false, false
		}
		if b, err := strconv.ParseBool(v); err == nil {
			return b, true
		}
	case int:
		return v != 0, true
	case int8:
		return v != 0, true
	case int16:
		return v != 0, true
	case int32:
		return v != 0, true
	case int64:
		return v != 0, true
	case uint:
		return v != 0, true
	case uint8:
		return v != 0, true
	case uint16:
		return v != 0, true
	case uint32:
		return v != 0, true
	case uint64:
		return v != 0, true
	case float32:
		return v != 0, true
	case float64:
		return v != 0, true
	}
	return false, false
}

// NewToolCallID generates a unique tool call ID.
//
// NewToolCallID 生成唯一的工具调用 ID。
func NewToolCallID() string {
	n := toolCallCounter.Add(1)
	return fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), n)
}

var toolCallCounter = &atomic.Int64{}
