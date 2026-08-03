package security

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PathValidator validates executable paths with multiple strategies
// PathValidator 使用多种策略验证可执行文件路径
type PathValidator struct {
	validator *CommandValidator
}

// NewPathValidator creates a new path validator
// NewPathValidator 创建新的路径验证器
func NewPathValidator(validator *CommandValidator) *PathValidator {
	if validator == nil {
		validator = NewCommandValidator()
	}
	return &PathValidator{
		validator: validator,
	}
}

// ValidateExecutable validates an executable using 8-layer validation strategy:
// 1. Shell metacharacter check
// 2. Command splitting
// 3. Relative path validation (./script, ../script)
// 4. Absolute path validation
// 5. Current directory check
// 6. PATH lookup
// 7. Whitelist check
// 8. Argument validation
//
// ValidateExecutable 使用 8 层验证策略验证可执行文件:
// 1. Shell 元字符检查
// 2. 命令分割
// 3. 相对路径验证 (./script, ../script)
// 4. 绝对路径验证
// 5. 当前目录检查
// 6. PATH 查找
// 7. 白名单检查
// 8. 参数验证
func (pv *PathValidator) ValidateExecutable(executable string, args []string) error {
	// Layer 1: Check for shell metacharacters in executable
	// 第 1 层: 检查可执行文件中的 shell 元字符
	if err := pv.validator.checkBlockedChars(executable); err != nil {
		return fmt.Errorf("executable contains dangerous characters: %w", err)
	}

	// Layer 2: Split command if needed (handle spaces)
	// 第 2 层: 如果需要则分割命令（处理空格）
	parts := pv.splitCommand(executable)
	if len(parts) == 0 {
		return fmt.Errorf("empty executable after parsing")
	}

	// Use first part as the actual executable
	// 使用第一部分作为实际的可执行文件
	firstPart := parts[0]

	// Layer 3: Relative path validation (./ or ../)
	// 第 3 层: 相对路径验证 (./ 或 ../)
	if pv.isRelativePath(firstPart) {
		return pv.validateRelativePath(firstPart, args)
	}

	// Layer 4: Absolute path validation
	// 第 4 层: 绝对路径验证
	if filepath.IsAbs(firstPart) {
		return pv.validateAbsolutePath(firstPart, args)
	}

	// Layer 5: Check current directory
	// 第 5 层: 检查当前目录
	if pv.existsInCurrentDir(firstPart) {
		// File exists in current directory, validate as relative path
		// 文件存在于当前目录，作为相对路径验证
		fullPath := filepath.Join(".", firstPart)
		return pv.validateRelativePath(fullPath, args)
	}

	// Layer 6: PATH lookup
	// 第 6 层: PATH 查找
	pathResolved, err := exec.LookPath(firstPart)
	if err == nil {
		// Found in PATH, now check whitelist
		// 在 PATH 中找到，现在检查白名单
		return pv.validateWithWhitelist(firstPart, pathResolved, args)
	}

	// Layer 7: Whitelist check (if not found anywhere else)
	// 第 7 层: 白名单检查（如果在其他地方找不到）
	if !pv.validator.isCommandAllowed(firstPart) {
		return fmt.Errorf("command '%s' is not in the allowed list and not found in PATH", firstPart)
	}

	// Layer 8: Validate all arguments
	// 第 8 层: 验证所有参数
	return pv.validateArguments(args)
}

// isRelativePath checks if the path is a relative path starting with ./ or ../
// isRelativePath 检查路径是否为以 ./ 或 ../ 开头的相对路径
func (pv *PathValidator) isRelativePath(path string) bool {
	// Unix-style relative paths
	// Unix 风格的相对路径
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return true
	}

	// Windows-style relative paths
	// Windows 风格的相对路径
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(path, ".\\") || strings.HasPrefix(path, "..\\") {
			return true
		}
	}

	return false
}

// validateRelativePath validates a relative path executable
// validateRelativePath 验证相对路径可执行文件
func (pv *PathValidator) validateRelativePath(path string, args []string) error {
	// Clean the path to prevent traversal attacks
	// 清理路径以防止遍历攻击
	cleanPath := filepath.Clean(path)

	// Check if file exists and is executable
	// 检查文件是否存在且可执行
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("relative path '%s' does not exist: %w", path, err)
	}

	// Check if it's a regular file
	// 检查是否为常规文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("relative path '%s' is not a regular file", path)
	}

	// On Unix, check if file has execute permission
	// 在 Unix 上，检查文件是否有执行权限
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("relative path '%s' is not executable", path)
		}
	}

	// Validate arguments
	// 验证参数
	return pv.validateArguments(args)
}

// validateAbsolutePath validates an absolute path executable
// validateAbsolutePath 验证绝对路径可执行文件
func (pv *PathValidator) validateAbsolutePath(path string, args []string) error {
	// Check if file exists
	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("absolute path '%s' does not exist: %w", path, err)
	}

	// Check if it's a regular file
	// 检查是否为常规文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("absolute path '%s' is not a regular file", path)
	}

	// On Unix, check if file has execute permission
	// 在 Unix 上，检查文件是否有执行权限
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("absolute path '%s' is not executable", path)
		}
	}

	// For absolute paths, also check whitelist with base name
	// 对于绝对路径，还要使用基本名称检查白名单
	baseName := filepath.Base(path)
	if !pv.validator.isCommandAllowed(baseName) {
		return fmt.Errorf("absolute path command '%s' is not in the allowed list", baseName)
	}

	// Validate arguments
	// 验证参数
	return pv.validateArguments(args)
}

// existsInCurrentDir checks if a file exists in the current directory
// existsInCurrentDir 检查文件是否存在于当前目录
func (pv *PathValidator) existsInCurrentDir(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// validateWithWhitelist validates using whitelist after PATH resolution
// validateWithWhitelist 在 PATH 解析后使用白名单验证
func (pv *PathValidator) validateWithWhitelist(original string, resolved string, args []string) error {
	// Check if the command is in whitelist
	// 检查命令是否在白名单中
	if !pv.validator.isCommandAllowed(original) {
		return fmt.Errorf("command '%s' is not in the allowed list", original)
	}

	// Validate the resolved path exists
	// 验证解析的路径存在
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("resolved path '%s' does not exist: %w", resolved, err)
	}

	// Check if it's a regular file
	// 检查是否为常规文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("resolved path '%s' is not a regular file", resolved)
	}

	// Validate arguments
	// 验证参数
	return pv.validateArguments(args)
}

// validateArguments validates all command arguments
// validateArguments 验证所有命令参数
func (pv *PathValidator) validateArguments(args []string) error {
	// Check all arguments for dangerous characters
	// 检查所有参数中的危险字符
	for i, arg := range args {
		if err := pv.validator.checkBlockedChars(arg); err != nil {
			return fmt.Errorf("argument %d contains dangerous characters: %w", i, err)
		}
	}
	return nil
}

// splitCommand splits a command string into parts
// This is a simple implementation - for production use, consider using a proper shell lexer
// splitCommand 将命令字符串分割为部分
// 这是一个简单的实现 - 对于生产使用，考虑使用适当的 shell 词法分析器
func (pv *PathValidator) splitCommand(cmd string) []string {
	// Trim whitespace
	// 修剪空白
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return []string{}
	}

	// Simple split by spaces (doesn't handle quotes)
	// For more complex scenarios, use a proper shell lexer like github.com/google/shlex
	// 简单的按空格分割（不处理引号）
	// 对于更复杂的场景，使用适当的 shell 词法分析器，如 github.com/google/shlex
	parts := strings.Fields(cmd)
	return parts
}

// GetValidator returns the underlying command validator
// GetValidator 返回底层的命令验证器
func (pv *PathValidator) GetValidator() *CommandValidator {
	return pv.validator
}
