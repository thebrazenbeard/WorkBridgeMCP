package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/policy"
)

type Service struct {
	read       *policy.RootPolicy
	write      *policy.RootPolicy
	maxRead    int64
	maxWrite   int64
	maxEntries int
}

type Entry struct {
	Name     string `json:"name"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type Stat struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Modified string `json:"modified"`
}

func New(cfg *config.Config) (*Service, error) {
	read, err := policy.NewRootPolicy(cfg.ReadRoots)
	if err != nil {
		return nil, err
	}
	write, err := policy.NewRootPolicy(cfg.WriteRoots)
	if err != nil {
		return nil, err
	}
	return &Service{
		read: read, write: write,
		maxRead: cfg.Limits.MaxReadBytes,
		maxWrite: cfg.Limits.MaxWriteBytes,
		maxEntries: cfg.Limits.MaxDirectoryEntries,
	}, nil
}

func (s *Service) CanWrite() bool { return !s.write.Empty() }

func (s *Service) List(path string) ([]Entry, error) {
	resolved, err := s.read.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}
	items, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}
	if len(items) > s.maxEntries {
		return nil, fmt.Errorf("directory has %d entries; limit is %d", len(items), s.maxEntries)
	}
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		info, err := item.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{
			Name: item.Name(), IsDir: item.IsDir(), Size: info.Size(),
			Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) Stat(path string) (*Stat, error) {
	resolved, err := s.read.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	return &Stat{
		Path: resolved, IsDir: info.IsDir(), Size: info.Size(), Mode: info.Mode().String(),
		Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}, nil
}

func (s *Service) ReadText(path string) (string, error) {
	resolved, err := s.read.ResolveExisting(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	if info.Size() > s.maxRead {
		return "", fmt.Errorf("file size %d exceeds read limit %d", info.Size(), s.maxRead)
	}
	data, err := io.ReadAll(io.LimitReader(f, s.maxRead+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > s.maxRead {
		return "", fmt.Errorf("read exceeds limit %d", s.maxRead)
	}
	if !utf8.Valid(data) {
		return "", errors.New("file is not valid UTF-8 text")
	}
	return string(data), nil
}

func (s *Service) WriteText(path, content string, overwrite bool) (int, error) {
	if s.write.Empty() {
		return 0, errors.New("write capability is disabled")
	}
	if int64(len(content)) > s.maxWrite {
		return 0, fmt.Errorf("write size %d exceeds limit %d", len(content), s.maxWrite)
	}
	clean := filepath.Clean(path)
	if info, err := os.Lstat(clean); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return 0, errors.New("refusing to overwrite a symbolic link")
	} else if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	candidate, err := s.write.ResolveForCreate(path)
	if err != nil {
		return 0, err
	}
	parent, err := s.write.ResolveExisting(filepath.Dir(candidate))
	if err != nil {
		return 0, err
	}
	target := filepath.Join(parent, filepath.Base(candidate))
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return 0, errors.New("refusing to overwrite a symbolic link")
		}
		if info.IsDir() {
			return 0, errors.New("refusing to overwrite a directory")
		}
		if !overwrite {
			return 0, errors.New("target already exists; set overwrite=true to replace it")
		}
		resolved, err := s.write.ResolveExisting(target)
		if err != nil {
			return 0, err
		}
		f, err := os.OpenFile(resolved, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		n, err := io.WriteString(f, content)
		if err == nil {
			err = f.Sync()
		}
		return n, err
	} else if !os.IsNotExist(err) {
		return 0, err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := io.WriteString(f, content)
	if err == nil {
		err = f.Sync()
	}
	return n, err
}

func (s *Service) Mkdir(path string) error {
	if s.write.Empty() {
		return errors.New("write capability is disabled")
	}
	candidate, err := s.write.ResolveForCreate(path)
	if err != nil {
		return err
	}
	parent, err := s.write.ResolveExisting(filepath.Dir(candidate))
	if err != nil {
		return err
	}
	target := filepath.Join(parent, filepath.Base(candidate))
	return os.Mkdir(target, 0o700)
}
