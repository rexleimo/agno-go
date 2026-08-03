package security

import (
	"fmt"
	"strings"
)

// CommandValidator validates MCP server commands for security
// CommandValidator 验证 MCP 服务器命令的安全性
type CommandValidator struct {
	// AllowedCommands is the whitelist of allowed command executables
	// AllowedCommands 是允许的命令可执行文件白名单
	AllowedCommands []string

	// BlockedChars are shell metacharacters that should be blocked
	// BlockedChars 是应被阻止的 shell 元字符
	BlockedChars []string
}

// DefaultAllowedCommands returns the default whitelist of allowed commands
// DefaultAllowedCommands 返回默认允许的命令白名单
func DefaultAllowedCommands() []string {
	return []string{
		"python",
		"python3",
		"node",
		"npm",
		"npx",
		"uvx",
		"docker",
	}
}

// DefaultBlockedChars returns the default list of blocked shell metacharacters
// DefaultBlockedChars 返回默认阻止的 shell 元字符列表
func DefaultBlockedChars() []string {
	return []string{
		"&",  // Background execution
		"|",  // Pipe
		";",  // Command separator
		"`",  // Command substitution
		"$",  // Variable expansion
		"<",  // Input redirection
		">",  // Output redirection
		"\\", // Escape character
		"(",  // Subshell
		")",  // Subshell
		"{",  // Brace expansion
		"}",  // Brace expansion
		"*",  // Glob
		"?",  // Glob
		"[",  // Glob
		"]",  // Glob
		"'",  // Single quote (can be used for injection)
		"\"", // Double quote (can be used for injection)
		"\n", // Newline (command separator)
		"\r", // Carriage return
	}
}

// NewCommandValidator creates a new command validator with default settings.
// NewCommandValidator 使用默认设置创建新的命令验证器。
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		AllowedCommands: DefaultAllowedCommands(),
		BlockedChars:    DefaultBlockedChars(),
	}
}

// NewCustomCommandValidator creates a new command validator with custom settings.
// NewCustomCommandValidator 使用自定义设置创建新的命令验证器。
func NewCustomCommandValidator(allowedCommands, blockedChars []string) *CommandValidator {
	return &CommandValidator{
		AllowedCommands: allowedCommands,
		BlockedChars:    blockedChars,
	}
}

// Validate validates a command and its arguments for security.
// Returns an error if the command is not allowed or contains dangerous characters.
//
// Validate 验证命令及其参数的安全性。
// 如果命令不被允许或包含危险字符，则返回错误。
func (v *CommandValidator) Validate(command string, args []string) error {
	// Check if command is in whitelist
	// 检查命令是否在白名单中
	if !v.isCommandAllowed(command) {
		return fmt.Errorf("command '%s' is not in the allowed list", command)
	}

	// Check command for dangerous characters
	// 检查命令中的危险字符
	if err := v.checkBlockedChars(command); err != nil {
		return fmt.Errorf("command contains dangerous characters: %w", err)
	}

	// Check all arguments for dangerous characters
	// 检查所有参数中的危险字符
	for i, arg := range args {
		if err := v.checkBlockedChars(arg); err != nil {
			return fmt.Errorf("argument %d contains dangerous characters: %w", i, err)
		}
	}

	return nil
}

// isCommandAllowed checks if the command is in the whitelist
// isCommandAllowed 检查命令是否在白名单中
func (v *CommandValidator) isCommandAllowed(command string) bool {
	// Extract base command name (remove path)
	// 提取基本命令名称（删除路径）
	baseName := command
	if idx := strings.LastIndex(command, "/"); idx != -1 {
		baseName = command[idx+1:]
	}
	if idx := strings.LastIndex(baseName, "\\"); idx != -1 {
		baseName = baseName[idx+1:]
	}

	// Check against whitelist
	// 对照白名单检查
	for _, allowed := range v.AllowedCommands {
		if baseName == allowed {
			return true
		}
	}

	return false
}

// checkBlockedChars checks if the string contains any blocked characters
// checkBlockedChars 检查字符串是否包含任何阻止的字符
func (v *CommandValidator) checkBlockedChars(s string) error {
	for _, char := range v.BlockedChars {
		if strings.Contains(s, char) {
			return fmt.Errorf("contains blocked character: %s", char)
		}
	}
	return nil
}

// SanitizeArgs sanitizes command arguments by removing or escaping dangerous characters.
// This is a best-effort approach and should be used with caution.
//
// SanitizeArgs 通过删除或转义危险字符来清理命令参数。
// 这是一种尽力而为的方法，应谨慎使用。
func (v *CommandValidator) SanitizeArgs(args []string) []string {
	sanitized := make([]string, len(args))
	for i, arg := range args {
		sanitized[i] = v.sanitizeString(arg)
	}
	return sanitized
}

// sanitizeString removes blocked characters from a string
// sanitizeString 从字符串中删除阻止的字符
func (v *CommandValidator) sanitizeString(s string) string {
	result := s
	for _, char := range v.BlockedChars {
		result = strings.ReplaceAll(result, char, "")
	}
	return result
}

// AddAllowedCommand adds a command to the whitelist
// AddAllowedCommand 将命令添加到白名单
func (v *CommandValidator) AddAllowedCommand(command string) {
	// Check if already exists
	// 检查是否已存在
	for _, c := range v.AllowedCommands {
		if c == command {
			return
		}
	}
	v.AllowedCommands = append(v.AllowedCommands, command)
}

// RemoveAllowedCommand removes a command from the whitelist
// RemoveAllowedCommand 从白名单中删除命令
func (v *CommandValidator) RemoveAllowedCommand(command string) {
	for i, c := range v.AllowedCommands {
		if c == command {
			v.AllowedCommands = append(v.AllowedCommands[:i], v.AllowedCommands[i+1:]...)
			return
		}
	}
}
