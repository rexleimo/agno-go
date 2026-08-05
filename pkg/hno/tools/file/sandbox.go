package file

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxReadBytes  int64 = 10 << 20 // 10 MiB
	defaultMaxWriteBytes int64 = 5 << 20  // 5 MiB
)

// AuditEntry describes a completed sandboxed file operation.
type AuditEntry struct {
	Operation string
	Path      string
	Size      int64
	At        time.Time
}

// Sandbox limits file operations to explicitly configured roots. A sandbox
// without read or write roots denies that capability by default.
type Sandbox struct {
	readRoots      []*sandboxRoot
	writeRoot      *sandboxRoot
	maxReadBytes   int64
	maxWriteBytes  int64
	allowOverwrite bool
	audit          func(AuditEntry)

	closeOnce sync.Once
	closeErr  error
}

type sandboxRoot struct {
	path string
	root *os.Root
}

type sandboxConfig struct {
	readRootPaths  []string
	writeRootPath  string
	maxReadBytes   int64
	maxWriteBytes  int64
	allowOverwrite bool
	audit          func(AuditEntry)
}

// SandboxOption configures a Sandbox created by NewSandbox.
type SandboxOption func(*sandboxConfig)

// WithReadRoots grants read access beneath the given directories.
func WithReadRoots(roots ...string) SandboxOption {
	paths := append([]string(nil), roots...)
	return func(config *sandboxConfig) {
		config.readRootPaths = append(config.readRootPaths, paths...)
	}
}

// WithWriteRoot grants write access beneath one directory.
func WithWriteRoot(root string) SandboxOption {
	return func(config *sandboxConfig) {
		config.writeRootPath = root
	}
}

// WithMaxReadBytes sets the maximum number of bytes a single read may return.
func WithMaxReadBytes(size int64) SandboxOption {
	return func(config *sandboxConfig) {
		config.maxReadBytes = size
	}
}

// WithMaxWriteBytes sets the maximum number of bytes a single write may accept.
func WithMaxWriteBytes(size int64) SandboxOption {
	return func(config *sandboxConfig) {
		config.maxWriteBytes = size
	}
}

// WithAllowOverwrite permits WriteFile to replace an existing regular file.
func WithAllowOverwrite(allow bool) SandboxOption {
	return func(config *sandboxConfig) {
		config.allowOverwrite = allow
	}
}

// WithAudit receives an entry after a successful sandboxed operation.
func WithAudit(audit func(AuditEntry)) SandboxOption {
	return func(config *sandboxConfig) {
		config.audit = audit
	}
}

// NewSandbox creates a fail-closed sandbox. Calling it without roots is valid,
// but all read and write path resolution attempts will be denied.
func NewSandbox(options ...SandboxOption) (*Sandbox, error) {
	config := sandboxConfig{
		maxReadBytes:  defaultMaxReadBytes,
		maxWriteBytes: defaultMaxWriteBytes,
	}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("sandbox option cannot be nil")
		}
		option(&config)
	}
	if config.maxReadBytes <= 0 {
		return nil, fmt.Errorf("max read bytes must be positive")
	}
	if config.maxWriteBytes <= 0 {
		return nil, fmt.Errorf("max write bytes must be positive")
	}

	sandbox := &Sandbox{
		maxReadBytes:   config.maxReadBytes,
		maxWriteBytes:  config.maxWriteBytes,
		allowOverwrite: config.allowOverwrite,
		audit:          config.audit,
	}
	for _, path := range config.readRootPaths {
		root, err := openSandboxRoot(path)
		if err != nil {
			_ = sandbox.Close()
			return nil, fmt.Errorf("open read root %q: %w", path, err)
		}
		sandbox.readRoots = append(sandbox.readRoots, root)
	}
	if config.writeRootPath != "" {
		root, err := openSandboxRoot(config.writeRootPath)
		if err != nil {
			_ = sandbox.Close()
			return nil, fmt.Errorf("open write root %q: %w", config.writeRootPath, err)
		}
		sandbox.writeRoot = root
	}

	return sandbox, nil
}

// Close releases the directory handles that enforce the sandbox boundary.
// It is safe to call more than once after all file operations have completed.
func (sandbox *Sandbox) Close() error {
	sandbox.closeOnce.Do(func() {
		var errs []error
		for _, root := range sandbox.readRoots {
			if err := root.root.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if sandbox.writeRoot != nil {
			if err := sandbox.writeRoot.root.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		sandbox.closeErr = errors.Join(errs...)
	})
	return sandbox.closeErr
}

func openSandboxRoot(path string) (*sandboxRoot, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve root path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve root symlinks: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory")
	}

	root, err := os.OpenRoot(resolved)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}
	return &sandboxRoot{path: filepath.Clean(resolved), root: root}, nil
}

func (root *sandboxRoot) relativePath(path string) (string, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", fmt.Errorf("invalid path")
	}

	cleaned := filepath.Clean(path)
	var relative string
	if filepath.IsAbs(cleaned) {
		absolute, err := filepath.Abs(cleaned)
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		canonical, err := canonicalPath(absolute)
		if err != nil {
			return "", err
		}
		relativePath, err := filepath.Rel(root.path, canonical)
		if err != nil {
			return "", fmt.Errorf("make path relative to sandbox root: %w", err)
		}
		relative = relativePath
	} else {
		if filepath.VolumeName(cleaned) != "" || isWindowsRootedRelative(cleaned) {
			return "", fmt.Errorf("path must not be volume-relative")
		}
		relative = cleaned
	}

	if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside sandbox root", path)
	}
	return relative, nil
}

func isWindowsRootedRelative(path string) bool {
	return runtime.GOOS == "windows" && (strings.HasPrefix(path, "\\") || strings.HasPrefix(path, "/"))
}

// canonicalPath resolves the deepest existing ancestor so Windows aliases and
// existing symlinks are normalized even when the final path does not exist.
func canonicalPath(path string) (string, error) {
	current := filepath.Clean(path)
	var suffix []string

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("resolve path symlinks: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("resolve path symlinks: %w", err)
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func (root *sandboxRoot) absolutePath(relative string) string {
	return filepath.Join(root.path, relative)
}

func (sandbox *Sandbox) selectReadRoot(path string) (*sandboxRoot, string, error) {
	if len(sandbox.readRoots) == 0 {
		return nil, "", fmt.Errorf("sandbox has no read roots")
	}

	var firstRoot *sandboxRoot
	var firstRelative string
	for _, root := range sandbox.readRoots {
		relative, err := root.relativePath(path)
		if err != nil {
			continue
		}
		if firstRoot == nil {
			firstRoot = root
			firstRelative = relative
		}

		if _, err := root.root.Stat(relative); err == nil {
			return root, relative, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", fmt.Errorf("validate read path: %w", err)
		}
	}
	if firstRoot != nil {
		return firstRoot, firstRelative, nil
	}
	return nil, "", fmt.Errorf("path %q is outside all read roots", path)
}

func (sandbox *Sandbox) writePath(path string) (*sandboxRoot, string, error) {
	if sandbox.writeRoot == nil {
		return nil, "", fmt.Errorf("sandbox has no write root")
	}
	relative, err := sandbox.writeRoot.relativePath(path)
	if err != nil {
		return nil, "", err
	}
	return sandbox.writeRoot, relative, nil
}

// ResolveRead validates path against the configured read roots and returns a
// root-relative absolute path for display or logging. Actual I/O must use the
// Sandbox methods so os.Root continues to enforce the boundary.
func (sandbox *Sandbox) ResolveRead(path string) (string, error) {
	root, relative, err := sandbox.selectReadRoot(path)
	if err != nil {
		return "", err
	}
	return root.absolutePath(relative), nil
}

// ResolveWrite validates path against the configured write root and returns a
// root-relative absolute path for display or logging. Actual I/O must use the
// Sandbox methods so os.Root continues to enforce the boundary.
func (sandbox *Sandbox) ResolveWrite(path string) (string, error) {
	root, relative, err := sandbox.writePath(path)
	if err != nil {
		return "", err
	}
	if relative == "." {
		return "", fmt.Errorf("path must name a file or directory below the write root")
	}

	parent := filepath.Dir(relative)
	if parent != "." {
		parentRoot, err := root.root.OpenRoot(parent)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("validate write path: %w", err)
		}
		if err == nil {
			if err := parentRoot.Close(); err != nil {
				return "", fmt.Errorf("close write parent: %w", err)
			}
		}
	}
	return root.absolutePath(relative), nil
}
