package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootPolicyAllowsInsideAndRejectsOutside(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a.txt")
	if err := os.WriteFile(inside, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := NewRootPolicy([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.ResolveExisting(inside); err != nil {
		t.Fatalf("inside denied: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "no.txt")
	if err := os.WriteFile(outside, []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ResolveExisting(outside); err == nil {
		t.Fatal("outside path accepted")
	}
	if _, err := p.ResolveForCreate(filepath.Join(root, "new", "child.txt")); err != nil {
		t.Fatalf("inside create denied: %v", err)
	}
}

func TestRootPolicyRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	p, err := NewRootPolicy([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.ResolveForCreate(filepath.Join(link, "x.txt")); err == nil {
		t.Fatal("symlink escape accepted")
	}
}
