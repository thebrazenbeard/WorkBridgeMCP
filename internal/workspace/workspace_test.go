package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func testConfig(root string, writable bool) *config.Config {
	cfg := &config.Config{
		Schema: config.Schema,
		ReadRoots: []string{root},
		Limits: config.Limits{MaxReadBytes: 1024, MaxWriteBytes: 1024, MaxDirectoryEntries: 10},
		Process: config.ProcessConfig{MaxRuntimeSeconds: 1, MaxOutputBytes: 1024, MaxArgs: 8},
		HTTP: config.HTTPConfig{Listen: "127.0.0.1:8765", Path: "/mcp"},
	}
	if writable {
		cfg.WriteRoots = []string{root}
	}
	return cfg
}

func TestReadListStatAndWriteBoundaries(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := New(testConfig(root, true))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	text, err := s.ReadText(file)
	if err != nil || text != "hello" {
		t.Fatalf("read=%q err=%v", text, err)
	}
	entries, err := s.List(root)
	if err != nil || len(entries) != 1 || entries[0].Name != "hello.txt" {
		t.Fatalf("list=%v err=%v", entries, err)
	}
	if entries[0].Path != file || entries[0].Type != "file" || entries[0].IsSymlink || entries[0].SizeBytes != 5 {
		t.Fatalf("unexpected normalized entry metadata: %#v", entries[0])
	}
	stat, err := s.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name != "hello.txt" || stat.Path != file || stat.Type != "file" || stat.IsSymlink || stat.SizeBytes != 5 || stat.MtimeNS == 0 {
		t.Fatalf("unexpected normalized stat metadata: %#v", stat)
	}
	newFile := filepath.Join(root, "new.txt")
	if _, err := s.WriteText(newFile, "new", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WriteText(newFile, "changed", false); err == nil {
		t.Fatal("overwrite occurred without overwrite=true")
	}
	if _, err := s.WriteText(newFile, "changed", true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(newFile)
	if string(got) != "changed" {
		t.Fatalf("unexpected write: %q", got)
	}
	nestedDir := filepath.Join(root, "nested", "deep")
	if err := s.Mkdir(nestedDir, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Mkdir(nestedDir, false); err != nil {
		t.Fatalf("idempotent mkdir failed: %v", err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(newFile, link); err == nil {
		if _, err := s.WriteText(link, "through-link", true); err == nil {
			t.Fatal("symbolic-link overwrite accepted")
		}
	}
}

func TestWriteDisabledAndOutsideRootDenied(t *testing.T) {
	root := t.TempDir()
	s, err := New(testConfig(root, false))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.WriteText(filepath.Join(root, "x.txt"), "x", false); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled write accepted: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "x.txt")
	if _, err := s.ReadText(outside); err == nil {
		t.Fatal("outside read accepted")
	}
}

func TestStatReportsSymlinkWithoutBreakingCompatibilityFields(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	s, err := New(testConfig(root, false))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	stat, err := s.Stat(link)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name != "link.txt" || stat.Type != "symlink" || !stat.IsSymlink || stat.IsDir {
		t.Fatalf("symlink metadata lost: %#v", stat)
	}
}
