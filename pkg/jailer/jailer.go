package jailer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Resolver enforces sandboxed path access.
type Resolver struct {
	root string
}

// New creates a new Resolver for given root.
func New(root string) (*Resolver, error) {
	if root == "" {
		return nil, errors.New("root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Resolver{root: abs}, nil
}

// Resolve returns an absolute path inside the sandbox root.
func (r *Resolver) Resolve(rel string) (string, error) {
	if strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	cleaned := filepath.Clean(rel)
	full := filepath.Join(r.root, cleaned)
	if !strings.HasPrefix(full, r.root) {
		return "", fmt.Errorf("path escapes sandbox")
	}
	return full, nil
}

// Within checks whether target is inside sandbox root.
func (r *Resolver) Within(target string) bool {
	abs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, r.root)
}

// EnsureDir ensures a directory exists within sandbox.
func (r *Resolver) EnsureDir(rel string) (string, error) {
	p, err := r.Resolve(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", err
	}
	return p, nil
}

// WriteFile writes data into a sandboxed path with size cap.
func (r *Resolver) WriteFile(rel string, data io.Reader, maxBytes int64) (string, error) {
	path, err := r.Resolve(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := data.Read(buf)
		if n > 0 {
			written += int64(n)
			if written > maxBytes {
				return "", fmt.Errorf("file exceeds size limit")
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return "", err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return path, nil
}

// OpenSafe opens a file inside sandbox for reading ensuring it exists.
func (r *Resolver) OpenSafe(rel string) (*os.File, error) {
	path, err := r.Resolve(rel)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

// Exists checks if a relative path exists inside sandbox.
func (r *Resolver) Exists(rel string) bool {
	p, err := r.Resolve(rel)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
