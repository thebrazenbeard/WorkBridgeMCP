package policy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type RootPolicy struct {
	roots []string
}

func NewRootPolicy(roots []string) (*RootPolicy, error) {
	p := &RootPolicy{}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			return nil, fmt.Errorf("root must be absolute: %q", root)
		}
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("stat root %q: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("root is not a directory: %q", root)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Clean(root))
		if err != nil {
			return nil, fmt.Errorf("resolve root %q: %w", root, err)
		}
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return nil, err
		}
		p.roots = append(p.roots, filepath.Clean(abs))
	}
	return p, nil
}

func (p *RootPolicy) Empty() bool { return p == nil || len(p.roots) == 0 }

func (p *RootPolicy) ResolveExisting(path string) (string, error) {
	if p.Empty() {
		return "", errors.New("no roots configured")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("path must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if !p.contains(abs) {
		return "", errors.New("path is outside configured roots")
	}
	return abs, nil
}

func (p *RootPolicy) ResolveForCreate(path string) (string, error) {
	if p.Empty() {
		return "", errors.New("no roots configured")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("path must be absolute")
	}
	clean := filepath.Clean(path)
	if _, err := os.Lstat(clean); err == nil {
		return p.ResolveExisting(clean)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	cur := clean
	var missing []string
	for {
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", errors.New("no existing ancestor found")
		}
		missing = append(missing, filepath.Base(cur))
		cur = parent
		if _, err := os.Lstat(cur); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	resolvedAncestor, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(resolvedAncestor)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(abs)
	for i := len(missing) - 1; i >= 0; i-- {
		candidate = filepath.Join(candidate, missing[i])
	}
	if !p.contains(candidate) {
		return "", errors.New("path is outside configured roots")
	}
	return candidate, nil
}

func (p *RootPolicy) contains(target string) bool {
	for _, root := range p.roots {
		rel, err := filepath.Rel(root, target)
		if err != nil {
			continue
		}
		if rel == "." {
			return true
		}
		if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if runtime.GOOS == "windows" {
			rootFold := strings.ToLower(filepath.Clean(root))
			targetFold := strings.ToLower(filepath.Clean(target))
			prefix := rootFold + string(filepath.Separator)
			if targetFold == rootFold || strings.HasPrefix(targetFold, prefix) {
				return true
			}
			continue
		}
		return true
	}
	return false
}
