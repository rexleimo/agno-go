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
