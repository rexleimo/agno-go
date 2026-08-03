package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Assertion validates a model response against an expectation. Implementations
// are stateless; call Check with the response content and tool calls.
//
// Assertion 校验模型响应是否符合预期。实现无状态；用响应内容与工具调用
// 调用 Check。
type Assertion interface {
	// Check returns nil when the response satisfies the assertion.
	// Check 在响应满足断言时返回 nil。
	Check(ctx context.Context, content string, toolCalls []types.ToolCall) error
}

// ---------- Contains ----------

// ContainsAssertion passes when content contains the expected substring
// (case-insensitive).
// ContainsAssertion 在内容包含预期子串（不区分大小写）时通过。
type ContainsAssertion struct {
	Expected string
}

// Check implements Assertion.
// Check 实现 Assertion。
func (a ContainsAssertion) Check(_ context.Context, content string, _ []types.ToolCall) error {
	if a.Expected == "" || strings.Contains(strings.ToLower(content), strings.ToLower(a.Expected)) {
		return nil
	}
	return fmt.Errorf("content does not contain %q", a.Expected)
}

// ---------- Regex ----------

// RegexAssertion passes when content matches the pattern.
// RegexAssertion 在内容匹配正则时通过。
type RegexAssertion struct {
	Pattern string
	re      *regexp.Regexp
}

// NewRegexAssertion compiles the pattern eagerly.
// NewRegexAssertion 预编译正则。
func NewRegexAssertion(pattern string) (*RegexAssertion, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexAssertion{Pattern: pattern, re: re}, nil
}

// Check implements Assertion.
// Check 实现 Assertion。
func (a *RegexAssertion) Check(_ context.Context, content string, _ []types.ToolCall) error {
	if a.re.MatchString(content) {
		return nil
	}
	return fmt.Errorf("content does not match %q", a.Pattern)
}

// ---------- JSONSchema ----------

// JSONSchemaAssertion passes when the content is valid JSON and matches the
// schema (a subset: required keys present with matching types).
//
// JSONSchemaAssertion 在内容是合法 JSON 且匹配 schema 时通过
// （子集：必需键存在且类型匹配）。
type JSONSchemaAssertion struct {
	// Required maps a JSON path (e.g. "result" or "data.items") to a type
	// name ("string", "number", "bool", "object", "array").
	// Required 将 JSON 路径（如 "result" 或 "data.items"）映射到类型名。
	Required map[string]string
}

// Check implements Assertion.
// Check 实现 Assertion。
func (a JSONSchemaAssertion) Check(_ context.Context, content string, _ []types.ToolCall) error {
	var doc interface{}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return fmt.Errorf("content is not valid JSON: %w", err)
	}
	for path, wantType := range a.Required {
		val, ok := lookupJSONPath(doc, path)
		if !ok {
			return fmt.Errorf("JSON path %q missing", path)
		}
		if got := jsonTypeName(val); got != wantType {
			return fmt.Errorf("JSON path %q type = %s, want %s", path, got, wantType)
		}
	}
	return nil
}

func lookupJSONPath(doc interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var cur interface{} = doc
	for _, p := range parts {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func jsonTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int64:
		return "number"
	case bool:
		return "bool"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

// ---------- ToolTrace ----------

// ToolTraceAssertion passes when the model issued a tool call matching the
// expected name during the run.
// ToolTraceAssertion 在模型运行期间发起过匹配预期名称的工具调用时通过。
type ToolTraceAssertion struct {
	ToolName string
}

// Check implements Assertion.
// Check 实现 Assertion。
func (a ToolTraceAssertion) Check(_ context.Context, _ string, toolCalls []types.ToolCall) error {
	for _, tc := range toolCalls {
		if tc.Function.Name == a.ToolName {
			return nil
		}
	}
	return fmt.Errorf("expected tool call %q not found", a.ToolName)
}

// ---------- LLMJudge ----------

// LLMJudgeAssertion uses another model to judge whether the response meets
// the criterion (LLM-as-a-judge).
// LLMJudgeAssertion 使用另一个模型评判响应是否满足标准（LLM 作评委）。
type LLMJudgeAssertion struct {
	Model     models.Model
	Criterion string
}

// Check implements Assertion.
// Check 实现 Assertion。
func (a LLMJudgeAssertion) Check(ctx context.Context, content string, _ []types.ToolCall) error {
	prompt := fmt.Sprintf(
		"Judge whether the following response meets the criterion.\nCriterion: %s\nResponse: %s\nReply with exactly YES or NO.",
		a.Criterion, content)
	resp, err := a.Model.Invoke(ctx, &models.InvokeRequest{
		Messages:  []*types.Message{types.NewUserMessage(prompt)},
		MaxTokens: 10,
	})
	if err != nil {
		return fmt.Errorf("judge model failed: %w", err)
	}
	if strings.Contains(strings.ToUpper(resp.Content), "YES") {
		return nil
	}
	return fmt.Errorf("judge rejected: %q", resp.Content)
}

// ---------- Runner ----------

// RunOne executes a scenario with an assertion against a model and reports
// whether the assertion passed.
// RunOne 用断言对模型执行一个场景，并报告断言是否通过。
func RunOne(ctx context.Context, m models.Model, input string, assertions ...Assertion) (bool, error) {
	resp, err := m.Invoke(ctx, &models.InvokeRequest{
		Messages: []*types.Message{types.NewUserMessage(input)},
	})
	if err != nil {
		return false, err
	}
	for _, a := range assertions {
		if err := a.Check(ctx, resp.Content, resp.ToolCalls); err != nil {
			return false, err
		}
	}
	return true, nil
}
