// Package backup implements backup orchestration and storage for Kubernetes resources.
//
// The storage layer provides a pluggable backend interface for persisting
// backup archives. The default implementation uses the local filesystem,
// with an S3 stub provided for future cloud storage integration.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// StorageBackend defines the interface for backup storage operations.
// Implementations must be safe for concurrent use.
type StorageBackend interface {
	// Write stores data at the given path, creating parent directories as needed.
	Write(ctx context.Context, path string, data []byte) error

	// Read retrieves data from the given path.
	Read(ctx context.Context, path string) ([]byte, error)

	// Delete removes the data at the given path.
	Delete(ctx context.Context, path string) error

	// List returns all paths under the given prefix, sorted alphabetically.
	List(ctx context.Context, prefix string) ([]string, error)

	// Exists checks whether data exists at the given path.
	Exists(ctx context.Context, path string) (bool, error)
}

// LocalStorage implements StorageBackend using the local filesystem.
// All paths are resolved relative to the configured root directory.
type LocalStorage struct {
	rootDir string
	mu      sync.RWMutex
}

// NewLocalStorage creates a new LocalStorage backend rooted at the given directory.
// The root directory is created if it does not exist.
func NewLocalStorage(rootDir string) (*LocalStorage, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to resolve root directory %q: %w", rootDir, err)
	}

	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, fmt.Errorf("storage: failed to create root directory %q: %w", absRoot, err)
	}

	return &LocalStorage{
		rootDir: absRoot,
	}, nil
}

// resolvePath joins the root directory with the given path and validates
// that the result does not escape the root directory.
func (s *LocalStorage) resolvePath(path string) (string, error) {
	// Clean the path to prevent directory traversal
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("storage: invalid path %q: must be relative and not escape root", path)
	}

	fullPath := filepath.Join(s.rootDir, cleaned)

	// Verify the resolved path is still under rootDir
	if !strings.HasPrefix(fullPath, s.rootDir) {
		return "", fmt.Errorf("storage: path %q resolves outside root directory", path)
	}

	return fullPath, nil
}

// Write stores data at the given path on the local filesystem.
func (s *LocalStorage) Write(ctx context.Context, path string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("storage: failed to create directory %q: %w", dir, err)
	}

	// Write to a temporary file first, then rename for atomicity
	tmpFile, err := os.CreateTemp(dir, ".aegis-tmp-*")
	if err != nil {
		return fmt.Errorf("storage: failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	_, writeErr := io.Copy(tmpFile, bytes.NewReader(data))
	closeErr := tmpFile.Close()
	if writeErr != nil {
		err = fmt.Errorf("storage: failed to write data: %w", writeErr)
		return err
	}
	if closeErr != nil {
		err = fmt.Errorf("storage: failed to close temp file: %w", closeErr)
		return err
	}

	// Atomic rename
	if err = os.Rename(tmpPath, fullPath); err != nil {
		return fmt.Errorf("storage: failed to rename temp file: %w", err)
	}

	return nil
}

// Read retrieves data from the given path on the local filesystem.
func (s *LocalStorage) Read(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: file not found at %q", path)
		}
		return nil, fmt.Errorf("storage: failed to read %q: %w", path, err)
	}

	return data, nil
}

// Delete removes the data at the given path on the local filesystem.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("storage: failed to delete %q: %w", path, err)
	}

	return nil
}

// List returns all file paths under the given prefix, relative to the root directory.
func (s *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPrefix, err := s.resolvePath(prefix)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var paths []string

	err = filepath.Walk(fullPrefix, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// If the prefix directory doesn't exist, return an empty list
			if os.IsNotExist(walkErr) {
				return filepath.SkipAll
			}
			return walkErr
		}
		if !info.IsDir() {
			// Convert to relative path
			relPath, relErr := filepath.Rel(s.rootDir, path)
			if relErr != nil {
				return relErr
			}
			paths = append(paths, relPath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("storage: failed to list prefix %q: %w", prefix, err)
	}

	sort.Strings(paths)
	return paths, nil
}

// Exists checks whether a file exists at the given path.
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("storage: failed to stat %q: %w", path, err)
	}
	return true, nil
}

// S3Storage is a stub implementation of StorageBackend for Amazon S3.
// This is provided for future integration and currently returns
// not-implemented errors for all operations.
type S3Storage struct {
	Bucket    string
	Region    string
	Prefix    string
	Endpoint  string // Custom endpoint for S3-compatible stores
}

// NewS3Storage creates a new S3Storage backend.
// This is a stub that will be implemented when S3 support is added.
func NewS3Storage(bucket, region, prefix string) *S3Storage {
	return &S3Storage{
		Bucket: bucket,
		Region: region,
		Prefix: prefix,
	}
}

func (s *S3Storage) Write(ctx context.Context, path string, data []byte) error {
	return fmt.Errorf("storage: S3 backend not yet implemented")
}

func (s *S3Storage) Read(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("storage: S3 backend not yet implemented")
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("storage: S3 backend not yet implemented")
}

func (s *S3Storage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, fmt.Errorf("storage: S3 backend not yet implemented")
}

func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("storage: S3 backend not yet implemented")
}
