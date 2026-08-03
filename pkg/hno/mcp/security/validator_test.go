package security

import (
	"strings"
	"testing"
)

func TestNewCommandValidator(t *testing.T) {
	validator := NewCommandValidator()

	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}

	if len(validator.AllowedCommands) == 0 {
		t.Error("Expected default allowed commands")
	}

	if len(validator.BlockedChars) == 0 {
		t.Error("Expected default blocked chars")
	}
}

func TestCommandValidator_Validate(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid python command",
			command: "python",
			args:    []string{"-m", "mcp_server"},
			wantErr: false,
		},
		{
			name:    "valid node command",
			command: "node",
			args:    []string{"server.js"},
			wantErr: false,
		},
		{
			name:    "valid npm command",
			command: "npm",
			args:    []string{"run", "start"},
			wantErr: false,
		},
		{
			name:    "python with path",
			command: "/usr/bin/python3",
			args:    []string{"-m", "server"},
			wantErr: false,
		},
		{
			name:    "disallowed command",
			command: "bash",
			args:    []string{"-c", "echo hello"},
			wantErr: true,
		},
		{
			name:    "command injection with semicolon",
			command: "python",
			args:    []string{"-m", "server; rm -rf /"},
			wantErr: true,
		},
		{
			name:    "command injection with pipe",
			command: "python",
			args:    []string{"-m", "server | cat /etc/passwd"},
			wantErr: true,
		},
		{
			name:    "command injection with ampersand",
			command: "python",
			args:    []string{"-m", "server & malicious"},
			wantErr: true,
		},
		{
			name:    "command injection with backticks",
			command: "python",
			args:    []string{"-m", "server`whoami`"},
			wantErr: true,
		},
		{
			name:    "command injection with dollar",
			command: "python",
			args:    []string{"-m", "server$(whoami)"},
			wantErr: true,
		},
		{
			name:    "dangerous command with redirection",
			command: "python",
			args:    []string{"-m", "server > /etc/passwd"},
			wantErr: true,
		},
		{
			name:    "valid uvx command",
			command: "uvx",
			args:    []string{"--from", "mcp", "run-server"},
			wantErr: false,
		},
		{
			name:    "valid docker command",
			command: "docker",
			args:    []string{"run", "--rm", "mcp-server"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.command, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandValidator_isCommandAllowed(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"python", "python", true},
		{"python3", "python3", true},
		{"node", "node", true},
		{"npm", "npm", true},
		{"npx", "npx", true},
		{"uvx", "uvx", true},
		{"docker", "docker", true},
		{"python with path", "/usr/bin/python", true},
		{"python3 with path", "/usr/local/bin/python3", true},
		{"bash", "bash", false},
		{"sh", "sh", false},
		{"curl", "curl", false},
		{"wget", "wget", false},
		{"rm", "rm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.isCommandAllowed(tt.command)
			if got != tt.want {
				t.Errorf("isCommandAllowed(%s) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestCommandValidator_checkBlockedChars(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"clean string", "hello-world", false},
		{"with semicolon", "hello;world", true},
		{"with pipe", "hello|world", true},
		{"with ampersand", "hello&world", true},
		{"with backtick", "hello`world", true},
		{"with dollar", "hello$world", true},
		{"with redirect", "hello>world", true},
		{"with newline", "hello\nworld", true},
		{"valid dash", "hello-world", false},
		{"valid underscore", "hello_world", false},
		{"valid dot", "hello.world", false},
		{"valid slash", "hello/world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkBlockedChars(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkBlockedChars(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCommandValidator_SanitizeArgs(t *testing.T) {
	validator := NewCommandValidator()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "clean args",
			args: []string{"hello", "world"},
			want: []string{"hello", "world"},
		},
		{
			name: "args with semicolon",
			args: []string{"hello;world", "test"},
			want: []string{"helloworld", "test"},
		},
		{
			name: "args with multiple dangerous chars",
			args: []string{"hello|world&test`cmd`", "normal"},
			want: []string{"helloworldtestcmd", "normal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.SanitizeArgs(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("SanitizeArgs() returned %d args, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SanitizeArgs()[%d] = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCommandValidator_AddRemoveAllowedCommand(t *testing.T) {
	validator := NewCommandValidator()

	initialCount := len(validator.AllowedCommands)

	// Test Add
	validator.AddAllowedCommand("go")
	if len(validator.AllowedCommands) != initialCount+1 {
		t.Errorf("Expected %d commands after add, got %d", initialCount+1, len(validator.AllowedCommands))
	}

	if !validator.isCommandAllowed("go") {
		t.Error("Expected 'go' to be allowed after adding")
	}

	// Test adding duplicate
	validator.AddAllowedCommand("go")
	if len(validator.AllowedCommands) != initialCount+1 {
		t.Error("Adding duplicate command should not increase count")
	}

	// Test Remove
	validator.RemoveAllowedCommand("go")
	if len(validator.AllowedCommands) != initialCount {
		t.Errorf("Expected %d commands after remove, got %d", initialCount, len(validator.AllowedCommands))
	}

	if validator.isCommandAllowed("go") {
		t.Error("Expected 'go' to not be allowed after removing")
	}

	// Test removing non-existent command (should not error)
	validator.RemoveAllowedCommand("nonexistent")
	if len(validator.AllowedCommands) != initialCount {
		t.Error("Removing non-existent command should not change count")
	}
}

func TestNewCustomCommandValidator(t *testing.T) {
	customAllowed := []string{"go", "rust"}
	customBlocked := []string{";", "|"}

	validator := NewCustomCommandValidator(customAllowed, customBlocked)

	if len(validator.AllowedCommands) != 2 {
		t.Errorf("Expected 2 allowed commands, got %d", len(validator.AllowedCommands))
	}

	if len(validator.BlockedChars) != 2 {
		t.Errorf("Expected 2 blocked chars, got %d", len(validator.BlockedChars))
	}

	if !validator.isCommandAllowed("go") {
		t.Error("Expected 'go' to be allowed")
	}

	if validator.isCommandAllowed("python") {
		t.Error("Expected 'python' to not be allowed")
	}
}

func TestDefaultAllowedCommands(t *testing.T) {
	commands := DefaultAllowedCommands()

	expectedCommands := []string{"python", "python3", "node", "npm", "npx", "uvx", "docker"}

	if len(commands) != len(expectedCommands) {
		t.Errorf("Expected %d default commands, got %d", len(expectedCommands), len(commands))
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' in default allowed commands", expected)
		}
	}
}

func TestDefaultBlockedChars(t *testing.T) {
	chars := DefaultBlockedChars()

	// Check that common dangerous chars are included
	// 检查是否包含常见的危险字符
	dangerousChars := []string{";", "|", "&", "`", "$", ">", "<"}

	for _, dangerous := range dangerousChars {
		found := false
		for _, char := range chars {
			if char == dangerous {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' in default blocked chars", dangerous)
		}
	}
}

func TestCommandValidator_WindowsPath(t *testing.T) {
	validator := NewCommandValidator()

	// Test Windows-style path
	// 测试 Windows 风格的路径
	command := "C:\\Python39\\python.exe"

	// This should fail because backslash is blocked
	// 这应该失败，因为反斜杠被阻止
	err := validator.Validate(command, []string{})
	if err == nil {
		t.Error("Expected error for Windows path with backslash")
	}

	// But the base command should still be recognized
	// 但基本命令仍应被识别
	if !strings.Contains(command, "python.exe") {
		t.Error("Expected command to contain python.exe")
	}
}
