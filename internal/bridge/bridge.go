package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/policy"
)

type Bridge struct {
	cfg           *config.Config
	readPolicy    *policy.RootPolicy
	writePolicy   *policy.RootPolicy
	workingPolicy *policy.RootPolicy
	allowedExec   map[string]string
}

func New(cfg *config.Config) (*Bridge, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	var readPolicy, writePolicy, workingPolicy *policy.RootPolicy
	var err error
	if len(cfg.ReadRoots) != 0 {
		readPolicy, err = policy.NewRootPolicy(cfg.ReadRoots)
		if err != nil {
			return nil, err
		}
	}
	if len(cfg.WriteRoots) != 0 {
		writePolicy, err = policy.NewRootPolicy(cfg.WriteRoots)
		if err != nil {
			if readPolicy != nil {
				_ = readPolicy.Close()
			}
			return nil, err
		}
	}
	if cfg.Process.Enabled {
		workingPolicy, err = policy.NewRootPolicy(cfg.Process.WorkingRoots)
		if err != nil {
			if readPolicy != nil {
				_ = readPolicy.Close()
			}
			if writePolicy != nil {
				_ = writePolicy.Close()
			}
			return nil, err
		}
	}
	b := &Bridge{
		cfg:           cfg,
		readPolicy:    readPolicy,
		writePolicy:   writePolicy,
		workingPolicy: workingPolicy,
		allowedExec:   map[string]string{},
	}
	if cfg.Process.Enabled {
		for _, executable := range cfg.Process.AllowedExecutables {
			canonical, err := canonicalExecutable(executable)
			if err != nil {
				_ = b.Close()
				return nil, fmt.Errorf("allowed executable %q: %w", executable, err)
			}
			b.allowedExec[pathKey(canonical)] = canonical
		}
	}
	return b, nil
}

func (b *Bridge) HasRead() bool    { return b.readPolicy != nil && !b.readPolicy.Empty() }
func (b *Bridge) HasWrite() bool   { return b.writePolicy != nil && !b.writePolicy.Empty() }
func (b *Bridge) HasProcess() bool { return b.cfg.Process.Enabled }

func (b *Bridge) Close() error {
	var first error
	for _, p := range []*policy.RootPolicy{b.readPolicy, b.writePolicy, b.workingPolicy} {
		if p == nil {
			continue
		}
		if err := p.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (b *Bridge) Info() map[string]any {
	host, _ := os.Hostname()
	return map[string]any{
		"hostname": host,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"capabilities": map[string]bool{
			"read":    b.HasRead(),
			"write":   b.HasWrite(),
			"process": b.HasProcess(),
		},
		"policy": map[string]any{
			"read_root_count":          len(b.cfg.ReadRoots),
			"write_root_count":         len(b.cfg.WriteRoots),
			"process_executable_count": len(b.cfg.Process.AllowedExecutables),
		},
	}
}

func canonicalExecutable(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
