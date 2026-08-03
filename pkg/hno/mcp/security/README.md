# MCP Security - Custom Binary Support

## Overview / 概述

This package provides security validation for MCP (Model Context Protocol) server commands, with support for custom binary executables.

该包为 MCP (Model Context Protocol) 服务器命令提供安全验证，支持自定义二进制可执行文件。

## Features / 特性

### 8-Layer Validation Strategy / 8 层验证策略

The `PathValidator` implements a comprehensive 8-layer validation strategy:

`PathValidator` 实现了全面的 8 层验证策略:

1. **Shell Metacharacter Check** / **Shell 元字符检查**
   - Blocks dangerous characters like `;`, `|`, `&`, `` ` ``, `$`, etc.
   - 阻止危险字符，如 `;`, `|`, `&`, `` ` ``, `$` 等

2. **Command Splitting** / **命令分割**
   - Parses command strings to extract the executable
   - 解析命令字符串以提取可执行文件

3. **Relative Path Validation** / **相对路径验证**
   - Validates scripts starting with `./` or `../`
   - Windows support: `.\\` and `..\\`
   - 验证以 `./` 或 `../` 开头的脚本
   - Windows 支持: `.\\` 和 `..\\`

4. **Absolute Path Validation** / **绝对路径验证**
   - Validates full file system paths
   - Checks file existence and permissions
   - 验证完整的文件系统路径
   - 检查文件存在性和权限

5. **Current Directory Check** / **当前目录检查**
   - Looks for executables in the current working directory
   - 在当前工作目录中查找可执行文件

6. **PATH Lookup** / **PATH 查找**
   - Uses `exec.LookPath` to find commands in system PATH
   - 使用 `exec.LookPath` 在系统 PATH 中查找命令

7. **Whitelist Validation** / **白名单验证**
   - Checks against allowed commands list
   - 对照允许的命令列表检查

8. **Argument Validation** / **参数验证**
   - Validates all command arguments for dangerous characters
   - 验证所有命令参数中的危险字符

## Usage / 使用方法

### Basic Usage / 基本用法

```go
import "github.com/rexleimo/agno-go/pkg/hno/mcp/security"

// Create a path validator
validator := security.NewPathValidator(nil) // nil uses default whitelist

// Validate a command
err := validator.ValidateExecutable("python", []string{"-m", "server"})
if err != nil {
    log.Fatal(err)
}
```

### With Custom Whitelist / 使用自定义白名单

```go
// Create custom command validator
customValidator := security.NewCustomCommandValidator(
    []string{"python", "node", "my-custom-tool"},
    security.DefaultBlockedChars(),
)

// Create path validator with custom whitelist
pathValidator := security.NewPathValidator(customValidator)

// Validate
err := pathValidator.ValidateExecutable("my-custom-tool", []string{"arg1"})
```

### Relative Path Scripts / 相对路径脚本

```go
// Validate relative path script
err := validator.ValidateExecutable("./my_script.sh", []string{})
```

### Absolute Path Executables / 绝对路径可执行文件

```go
// Add script name to whitelist first
validator.GetValidator().AddAllowedCommand("my_script.sh")

// Validate absolute path
err := validator.ValidateExecutable("/path/to/my_script.sh", []string{})
```

## Integration with MCP Client / 与 MCP 客户端集成

The validation is automatically integrated into `StdioTransport`:

验证已自动集成到 `StdioTransport`:

### Default Validation (Enabled) / 默认验证（已启用）

```go
import "github.com/rexleimo/agno-go/pkg/hno/mcp/client"

// Validation enabled by default when AllowedCommands is set
// 设置 AllowedCommands 时默认启用验证
config := client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server"},
    ValidateCommand: true, // Explicit enable
}

transport, err := client.NewStdioTransport(config)
```

### Custom Whitelist / 自定义白名单

```go
config := client.StdioConfig{
    Command: "./my_custom_server.sh",
    Args:    []string{"--port", "8000"},
    AllowedCommands: []string{"my_custom_server.sh"},
}

transport, err := client.NewStdioTransport(config)
```

### Disable Validation (Not Recommended) / 禁用验证（不推荐）

```go
config := client.StdioConfig{
    Command: "bash",
    Args:    []string{"-c", "some command"},
    ValidateCommand: false, // Explicitly disable
}

transport, err := client.NewStdioTransport(config)
```

## Security Considerations / 安全考虑

### Default Whitelist / 默认白名单

The default whitelist includes commonly used MCP server commands:

默认白名单包括常用的 MCP 服务器命令:

- `python`, `python3`
- `node`, `npm`, `npx`
- `uvx`
- `docker`

### Blocked Characters / 阻止的字符

The following shell metacharacters are blocked:

以下 shell 元字符被阻止:

- Command separators: `;`, `&`, `|`, `\n`, `\r`
- Command substitution: `` ` ``, `$`
- Redirection: `<`, `>`
- Glob patterns: `*`, `?`, `[`, `]`
- Brace expansion: `{`, `}`
- Escape character: `\`
- Quotes: `'`, `"`

### Cross-Platform Support / 跨平台支持

The validator supports both Unix and Windows path formats:

验证器支持 Unix 和 Windows 路径格式:

- Unix: `./script`, `../script`, `/usr/bin/python`
- Windows: `.\\script`, `..\\script`, `C:\\Python\\python.exe`

**Note**: Backslashes (`\`) are blocked by default. Use forward slashes (`/`) for cross-platform compatibility.

**注意**: 默认情况下反斜杠 (`\`) 被阻止。使用正斜杠 (`/`) 以实现跨平台兼容性。

## Performance / 性能

Benchmark results on Apple M3:

Apple M3 上的基准测试结果:

```
BenchmarkPathValidator_ValidateExecutable_Simple-8     59895    19898 ns/op    8104 B/op    94 allocs/op
BenchmarkPathValidator_ValidateExecutable_WithPath-8  312634     3857 ns/op     248 B/op     3 allocs/op
```

- Simple validation (PATH lookup): ~20 μs / 简单验证 (PATH 查找): ~20 μs
- Absolute path validation: ~4 μs / 绝对路径验证: ~4 μs

## Testing / 测试

The package includes comprehensive tests:

该包包含全面的测试:

- **Test Coverage**: 90.1% / **测试覆盖率**: 90.1%
- **Unit Tests**: All validation layers / **单元测试**: 所有验证层
- **Edge Cases**: Empty input, path traversal, null bytes / **边界情况**: 空输入、路径遍历、空字节
- **Cross-Platform**: Unix and Windows paths / **跨平台**: Unix 和 Windows 路径

Run tests:

运行测试:

```bash
go test -v ./pkg/hno/mcp/security/...
```

## Examples / 示例

See `/cmd/examples/mcp_custom_binary/main.go` for complete examples:

查看 `/cmd/examples/mcp_custom_binary/main.go` 获取完整示例:

1. Whitelisted commands / 白名单命令
2. Relative path scripts / 相对路径脚本
3. Absolute path executables / 绝对路径可执行文件
4. Custom whitelist / 自定义白名单
5. Validation disabled (for testing) / 禁用验证（用于测试）

## Migration from Python / 从 Python 迁移

This Go implementation provides equivalent functionality to Python's MCP validation with improvements:

此 Go 实现提供了与 Python 的 MCP 验证等效的功能，并进行了改进:

| Feature | Python | Go |
|---------|--------|-----|
| Shell metacharacter check | ✓ | ✓ |
| Command splitting | ✓ | ✓ |
| Relative path validation | ✓ | ✓ |
| Absolute path validation | ✓ | ✓ |
| PATH lookup | ✓ | ✓ |
| Whitelist validation | ✓ | ✓ |
| Windows support | Partial | ✓ |
| Performance | ~baseline | ~10x faster |

## API Reference / API 参考

### Types / 类型

```go
type PathValidator struct {
    validator *CommandValidator
}
```

### Functions / 函数

```go
// NewPathValidator creates a new path validator
func NewPathValidator(validator *CommandValidator) *PathValidator

// ValidateExecutable validates an executable using 8-layer strategy
func (pv *PathValidator) ValidateExecutable(executable string, args []string) error

// GetValidator returns the underlying command validator
func (pv *PathValidator) GetValidator() *CommandValidator
```

## License / 许可证

See repository LICENSE file.

查看仓库的 LICENSE 文件。
