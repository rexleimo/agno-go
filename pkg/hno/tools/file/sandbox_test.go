package file

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSandbox_FailClosedByDefault(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if _, err := sandbox.ReadFile("a.txt"); err == nil {
		t.Fatal("expected read to be denied without a read root")
	}
	if _, err := sandbox.ReadDir("."); err == nil {
		t.Fatal("expected list to be denied without a read root")
	}
	if _, err := sandbox.Stat("a.txt"); err == nil {
		t.Fatal("expected stat to be denied without a read root")
	}
	if err := sandbox.WriteFile("a.txt", []byte("content"), 0644); err == nil {
		t.Fatal("expected write to be denied without a write root")
	}
	if err := sandbox.DeleteFile("a.txt"); err == nil {
		t.Fatal("expected delete to be denied without a write root")
	}
	if err := sandbox.MkdirAll("nested", 0755); err == nil {
		t.Fatal("expected mkdir to be denied without a write root")
	}
}

func TestSandbox_SeparatesReadAndWriteRoots(t *testing.T) {
	readRoot := t.TempDir()
	writeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(readRoot, ".env"), []byte("safe=true"), 0644); err != nil {
		t.Fatal(err)
	}

	readOnly, err := NewSandbox(WithReadRoots(readRoot))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readOnly.Close() }()
	content, err := readOnly.ReadFile(".env")
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if string(content) != "safe=true" {
		t.Errorf("content = %q, want %q", content, "safe=true")
	}
	if err := readOnly.WriteFile("created.txt", []byte("no"), 0644); err == nil {
		t.Fatal("read root must not grant write access")
	}

	writeOnly, err := NewSandbox(WithWriteRoot(writeRoot))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writeOnly.Close() }()
	if err := writeOnly.WriteFile("created.txt", []byte("yes"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := writeOnly.ReadFile("created.txt"); err == nil {
		t.Fatal("write root must not implicitly grant read access")
	}
	if _, err := writeOnly.Stat("created.txt"); err == nil {
		t.Fatal("write root must not implicitly grant metadata access")
	}
}

func TestSandbox_RejectsTraversalAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(WithReadRoots(root), WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	for _, path := range []string{"../secret.txt", filepath.Join("a", "..", "..", "secret.txt"), ""} {
		if _, err := sandbox.ReadFile(path); err == nil {
			t.Errorf("expected read rejection for %q", path)
		}
		if err := sandbox.WriteFile(path, []byte("blocked"), 0644); err == nil {
			t.Errorf("expected write rejection for %q", path)
		}
	}
	if err := sandbox.WriteFile(filepath.Join(root, "inside.txt"), []byte("blocked"), 0644); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestSandbox_ReadFile_SymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("s3cr3t"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	sandbox, err := NewSandbox(WithReadRoots(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	content, err := sandbox.ReadFile("link.txt")
	if err == nil {
		t.Fatalf("expected symlink escape to fail, got %q", content)
	}
}

func TestSandbox_WriteFile_DirectorySymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	sandbox, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile(filepath.Join("out", "new.txt"), []byte("blocked"), 0644); err == nil {
		t.Fatal("expected directory symlink escape to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("outside file was created or could not be checked: %v", err)
	}
}

func TestSandbox_WriteFile_OverwriteRejectedByDefault(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	sandbox, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile("existing.txt", []byte("new"), 0644); err == nil {
		t.Fatal("expected overwrite to be rejected by default")
	}
}

func TestSandbox_WriteFile_OverwriteAllowed(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	sandbox, err := NewSandbox(WithWriteRoot(root), WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile("existing.txt", []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile with allow overwrite: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
}

func TestSandbox_CreateFile_OverwriteRequiresCallerAndPolicy(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	withoutPolicy, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutPolicy.CreateFile("existing.txt", []byte("new"), 0644, true); err == nil {
		t.Fatal("expected overwrite to require sandbox policy")
	}
	if err := withoutPolicy.Close(); err != nil {
		t.Fatal(err)
	}

	withPolicy, err := NewSandbox(WithWriteRoot(root), WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = withPolicy.Close() }()
	if err := withPolicy.CreateFile("existing.txt", []byte("new"), 0644, false); err == nil {
		t.Fatal("expected overwrite to require caller opt-in")
	}
	if err := withPolicy.CreateFile("existing.txt", []byte("new"), 0644, true); err != nil {
		t.Fatalf("overwrite with both opt-ins: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want new", got)
	}
}

func TestSandbox_CreateDirectory_CreatesParentsAndRejectsExistingTarget(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	path := filepath.Join("generated", "nested")
	if err := sandbox.CreateDirectory(path, 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("created target is not a directory")
	}
	if err := sandbox.CreateDirectory(path, 0755); err == nil {
		t.Fatal("expected existing directory target to fail")
	}
}

func TestSandbox_WriteFile_PreservesBackslashesOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backslash is a path separator on Windows")
	}

	root := t.TempDir()
	sandbox, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	path := "reports\\drafts/summary.txt"
	if err := sandbox.WriteFile(path, []byte("sandboxed"), 0644); err != nil {
		t.Fatalf("write file with Unix backslash name: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "reports\\drafts", "summary.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "sandboxed" {
		t.Fatalf("content = %q, want sandboxed", content)
	}
}

func TestSandbox_WriteFile_SizeLimit(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(WithWriteRoot(root), WithMaxWriteBytes(4))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile("big.txt", []byte("12345"), 0644); err == nil {
		t.Fatal("expected write over max write bytes to be rejected")
	}
}

func TestSandbox_ReadFile_SizeLimit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "big.bin"), make([]byte, 1024), 0644); err != nil {
		t.Fatal(err)
	}

	sandbox, err := NewSandbox(WithReadRoots(root), WithMaxReadBytes(512))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if _, err := sandbox.ReadFile("big.bin"); err == nil {
		t.Fatal("expected read over max read bytes to be rejected")
	}
}

func TestSandbox_WriteFile_CreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile(filepath.Join("nested", "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "nested", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "content" {
		t.Errorf("content = %q, want %q", got, "content")
	}
}

func TestSandbox_WriteFile_FinalSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	sandbox, err := NewSandbox(WithWriteRoot(root), WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile("link.txt", []byte("new"), 0644); err == nil {
		t.Fatal("expected final symlink to be rejected")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Errorf("target was modified through symlink: %q", got)
	}
}

func TestSandbox_AuditEntries(t *testing.T) {
	root := t.TempDir()
	var entries []AuditEntry
	sandbox, err := NewSandbox(
		WithWriteRoot(root),
		WithAudit(func(entry AuditEntry) { entries = append(entries, entry) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile("a.txt", []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(entries))
	}
	if entries[0].Operation != "write_file" || entries[0].Path != "a.txt" || entries[0].Size != 2 || entries[0].At.IsZero() {
		t.Fatalf("unexpected audit entry: %+v", entries[0])
	}
}

func TestSandbox_RejectsWindowsSpecialPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific path validation")
	}

	root := t.TempDir()
	sandbox, err := NewSandbox(WithReadRoots(root), WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sandbox.Close() }()

	for _, path := range []string{
		`C:foo`,
		`C:\Windows\win.ini`,
		`\Windows\win.ini`,
		`\\server\share\file.txt`,
		`\\?\C:\Windows\file.txt`,
		`file.txt:stream`,
		"NUL",
		"CON.txt",
		"COM1",
	} {
		if err := sandbox.WriteFile(path, []byte("blocked"), 0644); err == nil {
			t.Errorf("expected Windows special path %q to be rejected", path)
		}
	}
}
