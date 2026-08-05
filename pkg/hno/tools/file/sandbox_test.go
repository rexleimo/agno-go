package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandbox_FailClosedByDefault(t *testing.T) {
	sb, err := NewSandbox()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()
	if _, err := sb.ResolveRead(filepath.Join(t.TempDir(), "a.txt")); err == nil {
		t.Fatal("expected error for sandbox without read roots")
	}
	if _, err := sb.ResolveWrite(filepath.Join(t.TempDir(), "a.txt")); err == nil {
		t.Fatal("expected error for sandbox without write root")
	}
}

func TestSandbox_ResolveRead_InsideRoot(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(WithReadRoots(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()
	got, err := sb.ResolveRead(filepath.Join(root, "sub", "a.txt"))
	if err != nil {
		t.Fatalf("ResolveRead inside root: %v", err)
	}
	want, err := canonicalPath(filepath.Join(root, "sub", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSandbox_ResolveRead_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(WithReadRoots(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()
	cases := []string{
		filepath.Join(root, "..", "secret.txt"),
		filepath.Join(root, "..", "..", "etc", "passwd"),
		filepath.Join(root, "a", "..", "..", "x"),
	}
	for _, p := range cases {
		if _, err := sb.ResolveRead(p); err == nil {
			t.Errorf("expected rejection for %q", p)
		}
	}
}

func TestSandbox_ResolveRead_SymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("s3cr3t"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	sb, err := NewSandbox(WithReadRoots(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()
	if _, err := sb.ResolveRead(link); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestSandbox_ResolveWrite_DirSymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	sb, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()
	if _, err := sb.ResolveWrite(filepath.Join(root, "out", "new.txt")); err == nil {
		t.Fatal("expected dir symlink escape to be rejected")
	}
}

func TestSandbox_WriteFile_OverwriteRejectedByDefault(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	sb, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if err := sb.WriteFile(target, []byte("new"), 0644); err == nil {
		t.Fatal("expected overwrite to be rejected by default")
	}
}

func TestSandbox_WriteFile_OverwriteAllowed(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	sb, err := NewSandbox(WithWriteRoot(root), WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if err := sb.WriteFile(target, []byte("new"), 0644); err != nil {
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

func TestSandbox_WriteFile_SizeLimit(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(WithWriteRoot(root), WithMaxWriteBytes(4))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if err := sb.WriteFile(filepath.Join(root, "big.txt"), []byte("12345"), 0644); err == nil {
		t.Fatal("expected write over max write bytes to be rejected")
	}
}

func TestSandbox_ReadFile_SizeLimit(t *testing.T) {
	root := t.TempDir()
	big := filepath.Join(root, "big.bin")
	if err := os.WriteFile(big, make([]byte, 1024), 0644); err != nil {
		t.Fatal(err)
	}

	sb, err := NewSandbox(WithReadRoots(root), WithMaxReadBytes(512))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if _, err := sb.ReadFile(big); err == nil {
		t.Fatal("expected read over max read bytes to be rejected")
	}
}

func TestSandbox_WriteFile_CreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if err := sb.WriteFile(filepath.Join("nested", "file.txt"), []byte("content"), 0644); err != nil {
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
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	sb, err := NewSandbox(WithWriteRoot(root), WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sb.Close() }()

	if err := sb.WriteFile(link, []byte("new"), 0644); err == nil {
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
