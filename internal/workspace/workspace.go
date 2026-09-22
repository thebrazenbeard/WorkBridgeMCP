package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
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
		read.Close()
		return nil, err
	}
	return &Service{
		read: read, write: write,
		maxRead: cfg.Limits.MaxReadBytes,
		maxWrite: cfg.Limits.MaxWriteBytes,
		maxEntries: cfg.Limits.MaxDirectoryEntries,
	}, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	var first error
	if s.read != nil {
		first = s.read.Close()
	}
	if s.write != nil {
		if err := s.write.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Service) CanWrite() bool { return !s.write.Empty() }

func (s *Service) List(path string) ([]Entry, error) {
	f, _, err := s.read.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}
	items, err := f.ReadDir(s.maxEntries + 1)
	if err != nil {
		return nil, err
	}
	if len(items) > s.maxEntries {
		return nil, fmt.Errorf("directory exceeds entry limit %d", s.maxEntries)
	}
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		itemInfo, err := item.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{
			Name: item.Name(), IsDir: item.IsDir(), Size: itemInfo.Size(),
			Modified: itemInfo.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) Stat(path string) (*Stat, error) {
	info, resolved, err := s.read.Stat(path)
	if err != nil {
		return nil, err
	}
	return &Stat{
		Path: resolved, IsDir: info.IsDir(), Size: info.Size(), Mode: info.Mode().String(),
		Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}, nil
}

func (s *Service) ReadText(path string) (string, error) {
	f, _, err := s.read.Open(path)
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
	if info, _, err := s.write.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return 0, errors.New("refusing to overwrite a symbolic link")
		}
		if info.IsDir() {
			return 0, errors.New("refusing to overwrite a directory")
		}
		if !overwrite {
			return 0, errors.New("target already exists; set overwrite=true to replace it")
		}
	} else if !os.IsNotExist(err) {
		return 0, err
	}

	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, _, err := s.write.OpenFile(path, flags, 0o600)
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
	_, err := s.write.Mkdir(path, 0o700)
	return err
}
