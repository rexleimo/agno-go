package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
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

// WithAllowOverwrite permits WriteFile to replace an existing regular file and
// enables CreateFile's explicit overwrite opt-in.
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
	if config.maxReadBytes <= 0 || config.maxReadBytes == math.MaxInt64 {
		return nil, fmt.Errorf("max read bytes must be positive and below max int64")
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
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory")
	}

	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}
	return &sandboxRoot{root: root}, nil
}

func (root *sandboxRoot) relativePath(path string) (string, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", fmt.Errorf("invalid path")
	}

	relative := filepath.Clean(path)
	if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || isWindowsRootedRelative(relative) {
		return "", fmt.Errorf("path must be relative to the sandbox root")
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside sandbox root", path)
	}
	if runtime.GOOS == "windows" && strings.Contains(relative, ":") {
		return "", fmt.Errorf("path must not contain a Windows alternate data stream")
	}
	if runtime.GOOS == "windows" && hasWindowsReservedName(relative) {
		return "", fmt.Errorf("path contains a Windows reserved device name")
	}
	return relative, nil
}

func hasWindowsReservedName(path string) bool {
	for _, component := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		base := strings.TrimRight(component, " .")
		if dot := strings.IndexByte(base, '.'); dot >= 0 {
			base = base[:dot]
		}
		base = strings.ToUpper(base)
		switch base {
		case "CON", "PRN", "AUX", "NUL", "CLOCK$":
			return true
		}
		if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
			return true
		}
	}
	return false
}

func isWindowsRootedRelative(path string) bool {
	return runtime.GOOS == "windows" && (strings.HasPrefix(path, "\\") || strings.HasPrefix(path, "/"))
}

func (sandbox *Sandbox) selectReadRoot(path string) (*sandboxRoot, string, error) {
	if len(sandbox.readRoots) == 0 {
		return nil, "", fmt.Errorf("sandbox has no read roots")
	}

	relative, err := sandbox.readRoots[0].relativePath(path)
	if err != nil {
		return nil, "", err
	}
	for _, root := range sandbox.readRoots {
		if _, err := root.root.Stat(relative); err == nil {
			return root, relative, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", fmt.Errorf("validate read path: %w", err)
		}
	}
	return sandbox.readRoots[0], relative, nil
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

// ReadFile reads one regular file within a configured read root. It limits the
// bytes read even if the file grows after its initial size check.
func (sandbox *Sandbox) ReadFile(path string) ([]byte, error) {
	file, relative, _, err := sandbox.openReadFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, sandbox.maxReadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if int64(len(data)) > sandbox.maxReadBytes {
		return nil, fmt.Errorf("file %q exceeds maximum read size of %d bytes", path, sandbox.maxReadBytes)
	}
	sandbox.record("read_file", relative, int64(len(data)))
	return data, nil
}

func (sandbox *Sandbox) openReadFile(path string) (*os.File, string, fs.FileInfo, error) {
	root, relative, err := sandbox.selectReadRoot(path)
	if err != nil {
		return nil, "", nil, err
	}

	file, err := root.root.Open(relative)
	if err != nil {
		return nil, "", nil, fmt.Errorf("open file for reading: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, "", nil, fmt.Errorf("stat file for reading: %w", err)
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, "", nil, fmt.Errorf("file %q is not a regular file", path)
	}
	if info.Size() > sandbox.maxReadBytes {
		file.Close()
		return nil, "", nil, fmt.Errorf("file %q exceeds maximum read size of %d bytes", path, sandbox.maxReadBytes)
	}
	return file, relative, info, nil
}

// ReadDir lists one directory within a configured read root.
func (sandbox *Sandbox) ReadDir(path string) ([]fs.DirEntry, error) {
	root, relative, err := sandbox.selectReadRoot(path)
	if err != nil {
		return nil, err
	}

	directory, err := root.root.Open(relative)
	if err != nil {
		return nil, fmt.Errorf("open directory: %w", err)
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", path)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	sandbox.record("list_files", relative, int64(len(entries)))
	return entries, nil
}

// Stat returns metadata for one path within a configured read root.
func (sandbox *Sandbox) Stat(path string) (fs.FileInfo, error) {
	root, relative, err := sandbox.selectReadRoot(path)
	if err != nil {
		return nil, err
	}
	info, err := root.root.Stat(relative)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	sandbox.record("stat_file", relative, info.Size())
	return info, nil
}

// WriteFile writes content within the configured write root according to the
// Sandbox overwrite policy.
func (sandbox *Sandbox) WriteFile(path string, content []byte, mode os.FileMode) error {
	return sandbox.writeFile(path, content, mode, sandbox.allowOverwrite, "write_file")
}

// CreateFile creates a file within the configured write root. Replacing an
// existing file requires both overwrite=true and WithAllowOverwrite(true).
func (sandbox *Sandbox) CreateFile(path string, content []byte, mode os.FileMode, overwrite bool) error {
	return sandbox.writeFile(path, content, mode, overwrite && sandbox.allowOverwrite, "create_file")
}

// writeFile writes content within the configured write root. New targets are
// created with O_EXCL; replacing an existing regular file requires an allowed
// overwrite. Go 1.24 does not expose a root-bound rename, so an allowed
// overwrite has normal truncate-and-write semantics rather than an atomic
// replacement.
func (sandbox *Sandbox) writeFile(path string, content []byte, mode os.FileMode, allowOverwrite bool, operation string) error {
	if int64(len(content)) > sandbox.maxWriteBytes {
		return fmt.Errorf("write to %q exceeds maximum write size of %d bytes", path, sandbox.maxWriteBytes)
	}

	root, relative, err := sandbox.writePath(path)
	if err != nil {
		return err
	}
	if relative == "." {
		return fmt.Errorf("path must name a file below the write root")
	}

	parent := filepath.Dir(relative)
	if err := mkdirAllInRoot(root.root, parent, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	existing := false
	if info, err := root.root.Lstat(relative); err == nil {
		existing = true
		if info.IsDir() {
			return fmt.Errorf("write target %q is a directory", path)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("write target %q is a symbolic link", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("write target %q is not a regular file", path)
		}
		if !allowOverwrite {
			return fmt.Errorf("file %q already exists and overwrite is disabled", path)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect write target: %w", err)
	}

	permissions := mode.Perm()
	if permissions == 0 {
		permissions = 0644
	}
	flags := os.O_WRONLY | os.O_CREATE
	if existing {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := root.root.OpenFile(relative, flags, permissions)
	if err != nil {
		return fmt.Errorf("open write target: %w", err)
	}
	cleanupCreatedFile := !existing
	defer func() {
		_ = file.Close()
		if cleanupCreatedFile {
			_ = root.root.Remove(relative)
		}
	}()

	written, err := file.Write(content)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if written != len(content) {
		return fmt.Errorf("write file: %w", io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	cleanupCreatedFile = false
	sandbox.record(operation, relative, int64(len(content)))
	return nil
}

func mkdirAllInRoot(root *os.Root, path string, mode os.FileMode) error {
	if path == "." {
		return nil
	}

	current := root
	var opened []*os.Root
	defer func() {
		for index := len(opened) - 1; index >= 0; index-- {
			_ = opened[index].Close()
		}
	}()

	separator := rune(filepath.Separator)
	for _, component := range strings.FieldsFunc(path, func(r rune) bool {
		if r == separator {
			return true
		}
		return runtime.GOOS == "windows" && r == '/'
	}) {
		if component == "" || component == "." {
			continue
		}
		if component == ".." {
			return fmt.Errorf("directory path escapes sandbox root")
		}
		if err := current.Mkdir(component, mode); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			return err
		}
		opened = append(opened, next)
		current = next
	}
	return nil
}

// DeleteFile removes a file or empty directory beneath the write root.
func (sandbox *Sandbox) DeleteFile(path string) error {
	root, relative, err := sandbox.writePath(path)
	if err != nil {
		return err
	}
	if relative == "." {
		return fmt.Errorf("path must name a file or directory below the write root")
	}
	if err := root.root.Remove(relative); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	sandbox.record("delete_file", relative, 0)
	return nil
}

// CreateDirectory creates one directory beneath the write root. Its parent
// hierarchy is created as needed, but an existing target remains an error.
func (sandbox *Sandbox) CreateDirectory(path string, mode os.FileMode) error {
	root, relative, err := sandbox.writePath(path)
	if err != nil {
		return err
	}
	if relative == "." {
		return fmt.Errorf("path must name a directory below the write root")
	}

	parent := filepath.Dir(relative)
	if err := mkdirAllInRoot(root.root, parent, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	permissions := mode.Perm()
	if permissions == 0 {
		permissions = 0755
	}
	if err := root.root.Mkdir(relative, permissions); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	sandbox.record("create_directory", relative, 0)
	return nil
}

// MkdirAll creates a directory hierarchy beneath the write root.
func (sandbox *Sandbox) MkdirAll(path string, mode os.FileMode) error {
	root, relative, err := sandbox.writePath(path)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	permissions := mode.Perm()
	if permissions == 0 {
		permissions = 0755
	}
	if err := mkdirAllInRoot(root.root, relative, permissions); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	sandbox.record("create_directory", relative, 0)
	return nil
}

func (sandbox *Sandbox) record(operation, relative string, size int64) {
	if sandbox.audit == nil {
		return
	}
	sandbox.audit(AuditEntry{
		Operation: operation,
		Path:      relative,
		Size:      size,
		At:        time.Now().UTC(),
	})
}
