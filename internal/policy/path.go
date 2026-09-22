package policy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type rootEntry struct {
	paths []string
	root  *os.Root
}

type RootPolicy struct {
	roots []rootEntry
}

func NewRootPolicy(roots []string) (*RootPolicy, error) {
	p := &RootPolicy{}
	for _, raw := range roots {
		if !filepath.IsAbs(raw) {
			p.Close()
			return nil, fmt.Errorf("root must be absolute: %q", raw)
		}
		configured, err := filepath.Abs(filepath.Clean(raw))
		if err != nil {
			p.Close()
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(configured)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("resolve root %q: %w", raw, err)
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			p.Close()
			return nil, err
		}
		resolved = filepath.Clean(resolved)
		info, err := os.Stat(resolved)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("stat root %q: %w", resolved, err)
		}
		if !info.IsDir() {
			p.Close()
			return nil, fmt.Errorf("root is not a directory: %q", resolved)
		}
		r, err := os.OpenRoot(resolved)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("open root %q: %w", resolved, err)
		}
		aliases := []string{configured}
		if resolved != configured {
			aliases = append(aliases, resolved)
		}
		p.roots = append(p.roots, rootEntry{paths: aliases, root: r})
	}
	sort.Slice(p.roots, func(i, j int) bool {
		return longestPath(p.roots[i].paths) > longestPath(p.roots[j].paths)
	})
	return p, nil
}

func longestPath(paths []string) int {
	longest := 0
	for _, path := range paths {
		if len(path) > longest {
			longest = len(path)
		}
	}
	return longest
}

func (p *RootPolicy) Close() error {
	if p == nil {
		return nil
	}
	var first error
	for i := range p.roots {
		if p.roots[i].root == nil {
			continue
		}
		if err := p.roots[i].root.Close(); err != nil && first == nil {
			first = err
		}
		p.roots[i].root = nil
	}
	return first
}

func (p *RootPolicy) Empty() bool { return p == nil || len(p.roots) == 0 }

func (p *RootPolicy) ResolveExisting(path string) (string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return "", err
	}
	if _, err := entry.root.Stat(rel); err != nil {
		return "", err
	}
	return abs, nil
}

func (p *RootPolicy) Open(path string) (*os.File, string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return nil, "", err
	}
	f, err := entry.root.Open(rel)
	if err != nil {
		return nil, "", err
	}
	return f, abs, nil
}

func (p *RootPolicy) OpenFile(path string, flag int, perm fs.FileMode) (*os.File, string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return nil, "", err
	}
	f, err := entry.root.OpenFile(rel, flag, perm)
	if err != nil {
		return nil, "", err
	}
	return f, abs, nil
}

func (p *RootPolicy) Stat(path string) (fs.FileInfo, string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return nil, "", err
	}
	info, err := entry.root.Stat(rel)
	if err != nil {
		return nil, "", err
	}
	return info, abs, nil
}

func (p *RootPolicy) Lstat(path string) (fs.FileInfo, string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return nil, "", err
	}
	info, err := entry.root.Lstat(rel)
	if err != nil {
		return nil, "", err
	}
	return info, abs, nil
}

func (p *RootPolicy) Mkdir(path string, perm fs.FileMode) (string, error) {
	entry, rel, abs, err := p.match(path)
	if err != nil {
		return "", err
	}
	if err := entry.root.Mkdir(rel, perm); err != nil {
		return "", err
	}
	return abs, nil
}

func (p *RootPolicy) match(path string) (*rootEntry, string, string, error) {
	if p.Empty() {
		return nil, "", "", errors.New("no roots configured")
	}
	if !filepath.IsAbs(path) {
		return nil, "", "", errors.New("path must be absolute")
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, "", "", err
	}
	abs = filepath.Clean(abs)
	for i := range p.roots {
		for _, alias := range p.roots[i].paths {
			rel, err := filepath.Rel(alias, abs)
			if err != nil || !filepath.IsLocal(rel) {
				continue
			}
			return &p.roots[i], rel, abs, nil
		}
	}
	return nil, "", "", errors.New("path is outside configured roots")
}
