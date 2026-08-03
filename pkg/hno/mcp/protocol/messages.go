package protocol

// MCP Protocol Methods
// MCP 协议方法
const (
	MethodInitialize      = "initialize"
	MethodToolsList       = "tools/list"
	MethodToolsCall       = "tools/call"
	MethodResourcesList   = "resources/list"
	MethodResourcesRead   = "resources/read"
	MethodPromptsList     = "prompts/list"
	MethodPromptsGet      = "prompts/get"
	MethodLoggingSetLevel = "logging/setLevel"
)

// InitializeParams represents the parameters for the initialize method
// InitializeParams 表示 initialize 方法的参数
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
}

// ClientInfo contains information about the MCP client
// ClientInfo 包含 MCP 客户端的信息
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents the result of the initialize method
// InitializeResult 表示 initialize 方法的结果
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
}

// ServerInfo contains information about the MCP server
// ServerInfo 包含 MCP 服务器的信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListParams represents the parameters for the tools/list method
// ToolsListParams 表示 tools/list 方法的参数
type ToolsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResult represents the result of the tools/list method
// ToolsListResult 表示 tools/list 方法的结果
type ToolsListResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// Tool represents an MCP tool definition
// Tool 表示 MCP 工具定义
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents the JSON schema for tool input parameters
// InputSchema 表示工具输入参数的 JSON schema
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ToolsCallParams represents the parameters for the tools/call method
// ToolsCallParams 表示 tools/call 方法的参数
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolsCallResult represents the result of the tools/call method
// ToolsCallResult 表示 tools/call 方法的结果
type ToolsCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents different types of content in MCP responses
// Content 表示 MCP 响应中的不同类型内容
type Content struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	Data     string      `json:"data,omitempty"`     // Base64 encoded for images
	MimeType string      `json:"mimeType,omitempty"` // For images/resources
	URI      string      `json:"uri,omitempty"`      // For resources
	Resource interface{} `json:"resource,omitempty"` // For embedded resources
}

// Content type constants
// 内容类型常量
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeResource = "resource"
)

// ResourcesListParams represents the parameters for the resources/list method
// ResourcesListParams 表示 resources/list 方法的参数
type ResourcesListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ResourcesListResult represents the result of the resources/list method
// ResourcesListResult 表示 resources/list 方法的结果
type ResourcesListResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

// Resource represents an MCP resource
// Resource 表示 MCP 资源
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourcesReadParams represents the parameters for the resources/read method
// ResourcesReadParams 表示 resources/read 方法的参数
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// ResourcesReadResult represents the result of the resources/read method
// ResourcesReadResult 表示 resources/read 方法的结果
type ResourcesReadResult struct {
	Contents []Content `json:"contents"`
}

// PromptsListParams represents the parameters for the prompts/list method
// PromptsListParams 表示 prompts/list 方法的参数
type PromptsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// PromptsListResult represents the result of the prompts/list method
// PromptsListResult 表示 prompts/list 方法的结果
type PromptsListResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

// Prompt represents an MCP prompt template
// Prompt 表示 MCP 提示模板
type Prompt struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Arguments   []Argument `json:"arguments,omitempty"`
}

// Argument represents a prompt argument
// Argument 表示提示参数
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsGetParams represents the parameters for the prompts/get method
// PromptsGetParams 表示 prompts/get 方法的参数
type PromptsGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptsGetResult represents the result of the prompts/get method
// PromptsGetResult 表示 prompts/get 方法的结果
type PromptsGetResult struct {
	Description string    `json:"description,omitempty"`
	Messages    []Message `json:"messages"`
}

// Message represents a prompt message
// Message 表示提示消息
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// LoggingSetLevelParams represents the parameters for the logging/setLevel method
// LoggingSetLevelParams 表示 logging/setLevel 方法的参数
type LoggingSetLevelParams struct {
	Level string `json:"level"` // debug, info, warn, error
}
