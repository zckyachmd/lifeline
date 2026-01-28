package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"zckyachmd/lifeline/pkg/jailer"
)

// FileService handles sandboxed file operations.
type FileService struct {
	jail     *jailer.Resolver
	maxBytes int64
}

// NewFileService builds service with sandbox root and max size (MB).
func NewFileService(j *jailer.Resolver, maxMB int) *FileService {
	return &FileService{jail: j, maxBytes: int64(maxMB) * 1024 * 1024}
}

// MaxBytes exposes maximum allowed file size in bytes.
func (f *FileService) MaxBytes() int64 {
	return f.maxBytes
}

// List lists directory contents.
func (f *FileService) List(path string) ([]string, error) {
	abs, err := f.jail.Resolve(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// Read opens file for sending.
func (f *FileService) Read(path string) (*os.File, int64, error) {
	abs, err := f.jail.Resolve(path)
	if err != nil {
		return nil, 0, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, 0, err
	}
	if info.IsDir() {
		return nil, 0, fmt.Errorf("path is directory")
	}
	if info.Size() > f.maxBytes {
		return nil, 0, fmt.Errorf("file too large")
	}
	file, err := os.Open(abs)
	if err != nil {
		return nil, 0, err
	}
	return file, info.Size(), nil
}

// Save stores upload into inbox with time-based name if empty.
func (f *FileService) Save(filename string, r io.Reader) (string, error) {
	if filename == "" {
		filename = fmt.Sprintf("upload-%d", time.Now().Unix())
	}
	safe := filepath.Join("inbox", filepath.Base(filename))
	return f.jail.WriteFile(safe, r, f.maxBytes)
}
