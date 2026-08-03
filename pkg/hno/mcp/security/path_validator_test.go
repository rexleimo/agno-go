package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewPathValidator(t *testing.T) {
	// Test with nil validator
	pv := NewPathValidator(nil)
	if pv == nil {
		t.Fatal("Expected non-nil path validator")
	}
	if pv.validator == nil {
		t.Fatal("Expected validator to be created")
	}

	// Test with custom validator
	customValidator := NewCommandValidator()
	pv2 := NewPathValidator(customValidator)
	if pv2.validator != customValidator {
		t.Error("Expected custom validator to be used")
	}
}

func TestPathValidator_ValidateExecutable_ShellMetacharacters(t *testing.T) {
	pv := NewPathValidator(nil)

	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
	}{
		{
			name:       "clean executable",
			executable: "python",
			args:       []string{"-m", "server"},
			wantErr:    false,
		},
		{
			name:       "executable with semicolon",
			executable: "python; rm -rf /",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "executable with pipe",
			executable: "python | cat",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "executable with backtick",
			executable: "python`whoami`",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "args with dangerous chars",
			executable: "python",
			args:       []string{"-m", "server; whoami"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_ValidateExecutable_RelativePath(t *testing.T) {
	pv := NewPathValidator(nil)

	// Create a temporary script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	// Create the script
	content := "#!/bin/bash\necho 'test'\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
	}{
		{
			name:       "valid relative path with ./",
			executable: "./test_script.sh",
			args:       []string{},
			wantErr:    false,
		},
		{
			name:       "invalid relative path",
			executable: "./nonexistent.sh",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "relative path with args",
			executable: "./test_script.sh",
			args:       []string{"arg1", "arg2"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_ValidateExecutable_AbsolutePath(t *testing.T) {
	pv := NewPathValidator(nil)

	// Add python to allowed commands for this test
	pv.validator.AddAllowedCommand("test_script.sh")

	// Create a temporary script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	// Create the script
	content := "#!/bin/bash\necho 'test'\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
	}{
		{
			name:       "valid absolute path",
			executable: scriptPath,
			args:       []string{},
			wantErr:    false,
		},
		{
			name:       "invalid absolute path",
			executable: "/nonexistent/script.sh",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "absolute path with args",
			executable: scriptPath,
			args:       []string{"arg1", "arg2"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_ValidateExecutable_PATHLookup(t *testing.T) {
	pv := NewPathValidator(nil)

	// Test with commands that should be in PATH
	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
		skipOnOS   string // Skip test on specific OS
	}{
		{
			name:       "python in PATH",
			executable: "python",
			args:       []string{"-m", "test"},
			wantErr:    false,
		},
		{
			name:       "python3 in PATH",
			executable: "python3",
			args:       []string{"--version"},
			wantErr:    false,
		},
		{
			name:       "nonexistent command",
			executable: "nonexistentcommand12345",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "node in PATH",
			executable: "node",
			args:       []string{"--version"},
			wantErr:    false,
			skipOnOS:   "", // May not exist on all systems
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnOS != "" && runtime.GOOS == tt.skipOnOS {
				t.Skipf("Skipping on %s", runtime.GOOS)
			}

			// Check if command exists in PATH before testing
			if tt.executable != "nonexistentcommand12345" {
				if _, err := exec.LookPath(tt.executable); err != nil {
					t.Skipf("Command %s not found in PATH, skipping", tt.executable)
				}
			}

			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_ValidateExecutable_CurrentDirectory(t *testing.T) {
	pv := NewPathValidator(nil)

	// Create a temporary script in current directory
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "current_dir_script.sh")

	// Create the script
	content := "#!/bin/bash\necho 'test'\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
	}{
		{
			name:       "script in current directory",
			executable: "current_dir_script.sh",
			args:       []string{},
			wantErr:    false,
		},
		{
			name:       "nonexistent in current directory",
			executable: "nonexistent_script.sh",
			args:       []string{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_isRelativePath(t *testing.T) {
	pv := NewPathValidator(nil)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"unix relative current", "./script", true},
		{"unix relative parent", "../script", true},
		{"absolute unix", "/usr/bin/python", false},
		{"plain name", "python", false},
		{"windows relative current", ".\\script", runtime.GOOS == "windows"},
		{"windows relative parent", "..\\script", runtime.GOOS == "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.isRelativePath(tt.path)
			if got != tt.want {
				t.Errorf("isRelativePath(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestPathValidator_splitCommand(t *testing.T) {
	pv := NewPathValidator(nil)

	tests := []struct {
		name string
		cmd  string
		want []string
	}{
		{
			name: "single command",
			cmd:  "python",
			want: []string{"python"},
		},
		{
			name: "command with spaces",
			cmd:  "python -m server",
			want: []string{"python", "-m", "server"},
		},
		{
			name: "empty command",
			cmd:  "",
			want: []string{},
		},
		{
			name: "command with multiple spaces",
			cmd:  "python  -m   server",
			want: []string{"python", "-m", "server"},
		},
		{
			name: "command with leading/trailing spaces",
			cmd:  "  python -m server  ",
			want: []string{"python", "-m", "server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.splitCommand(tt.cmd)
			if len(got) != len(tt.want) {
				t.Errorf("splitCommand() returned %d parts, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommand()[%d] = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPathValidator_validateArguments(t *testing.T) {
	pv := NewPathValidator(nil)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "clean args",
			args:    []string{"-m", "server", "--port", "8000"},
			wantErr: false,
		},
		{
			name:    "args with semicolon",
			args:    []string{"-m", "server; rm -rf /"},
			wantErr: true,
		},
		{
			name:    "args with pipe",
			args:    []string{"--config", "file | cat"},
			wantErr: true,
		},
		{
			name:    "args with backtick",
			args:    []string{"--name", "`whoami`"},
			wantErr: true,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.validateArguments(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArguments() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	pv := NewPathValidator(nil)

	tests := []struct {
		name       string
		executable string
		wantErr    bool
	}{
		{
			name:       "windows relative path with backslash",
			executable: ".\\script.bat",
			wantErr:    true, // Backslash is blocked
		},
		{
			name:       "windows relative path with forward slash",
			executable: "./script.bat",
			wantErr:    false, // Forward slash should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_EdgeCases(t *testing.T) {
	pv := NewPathValidator(nil)

	tests := []struct {
		name       string
		executable string
		args       []string
		wantErr    bool
	}{
		{
			name:       "empty executable",
			executable: "",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "whitespace only executable",
			executable: "   ",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "path traversal attempt",
			executable: "../../etc/passwd",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "null byte attempt",
			executable: "python\x00malicious",
			args:       []string{},
			wantErr:    true, // Command name with null byte should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pv.ValidateExecutable(tt.executable, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator_GetValidator(t *testing.T) {
	customValidator := NewCommandValidator()
	pv := NewPathValidator(customValidator)

	got := pv.GetValidator()
	if got != customValidator {
		t.Error("GetValidator() did not return the correct validator")
	}
}

// Benchmark tests
func BenchmarkPathValidator_ValidateExecutable_Simple(b *testing.B) {
	pv := NewPathValidator(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = pv.ValidateExecutable("python", []string{"-m", "server"})
	}
}

func BenchmarkPathValidator_ValidateExecutable_WithPath(b *testing.B) {
	pv := NewPathValidator(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = pv.ValidateExecutable("/usr/bin/python3", []string{"--version"})
	}
}
