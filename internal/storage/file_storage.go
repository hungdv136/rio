package storage

//go:generate mockgen -source file-storage.go -destination ../test/mock/file-storage.go -package mock

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/hungdv136/rio/internal/log"
)

var _ FileStorage = (*LocalStorage)(nil)

// FileStorage defines interface for file store
type FileStorage interface {
	UploadFile(ctx context.Context, objectKey string, reader io.Reader) (string, error)
	DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, objectKey string) error
	ListFiles(ctx context.Context) ([]string, error)
	Reset(ctx context.Context) error
}

// NewFileStorage creates a new file storages
// TODO: Support other types (GCS, S3) to deploy mock server as a cluster
func NewFileStorage(cfg interface{}) FileStorage {
	return NewLocalStorage(cfg.(LocalStorageConfig))
}

// LocalStorageConfig defines config for local storage
type LocalStorageConfig struct {
	StoragePath string
	UseTempDir  bool
}

// LocalStorage define methods to access local storage
type LocalStorage struct {
	config         LocalStorageConfig
	tempDirOnce    sync.Once
	tempDirCreated int32
}

// NewLocalStorage creates a new instance of LocalStorage
func NewLocalStorage(config LocalStorageConfig) *LocalStorage {
	return &LocalStorage{
		config: config,
	}
}

// UploadFile reads from reader and saves to local storage
func (s *LocalStorage) UploadFile(ctx context.Context, objectKey string, reader io.Reader) (string, error) {
	filePath := path.Join(s.getStoragePath(), objectKey)
	if err := os.MkdirAll(path.Dir(filePath), os.ModePerm); err != nil {
		log.Error(ctx, err)
		return "", err
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}

	if err = os.WriteFile(filePath, content, 0o600); err != nil {
		log.Error(ctx, err)
		return "", err
	}

	return filePath, nil
}

// DownloadFile writes file saved in local storage to writer
func (s *LocalStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	filePath := path.Join(s.getStoragePath(), objectKey)
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

// DeleteFile deletes file from local storage
func (s *LocalStorage) DeleteFile(ctx context.Context, objectKey string) error {
	filePath := path.Join(s.getStoragePath(), objectKey)
	if err := os.Remove(filePath); err != nil {
		log.Error(ctx, err)
		return err
	}

	return nil
}

// ListFiles list all files
func (s *LocalStorage) ListFiles(ctx context.Context) ([]string, error) {
	files, err := os.ReadDir(s.getStoragePath())
	if err != nil {
		log.Error(ctx, err)
		return []string{}, err
	}

	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if file.Type().IsRegular() {
			fileNames = append(fileNames, file.Name())
		}
	}

	return fileNames, nil
}

// Reset removes all files
func (s *LocalStorage) Reset(ctx context.Context) error {
	// If temp dir is not created, then don't need to remove
	if s.config.UseTempDir && atomic.LoadInt32(&s.tempDirCreated) == 0 {
		return nil
	}

	storagePath := s.getStoragePath()
	if len(storagePath) == 0 {
		return nil
	}

	if err := os.RemoveAll(storagePath); err != nil {
		log.Error(ctx, "cannot remove files", storagePath, err)
		return err
	}

	return nil
}

func (s *LocalStorage) getStoragePath() string {
	if !s.config.UseTempDir {
		return s.config.StoragePath
	}

	// Lazy create temp directory to avoid creating unnecessary folders
	s.tempDirOnce.Do(func() {
		uploadDir, err := os.MkdirTemp("", s.config.StoragePath)
		if err != nil {
			panic(err)
		}

		s.config.StoragePath = uploadDir
		atomic.AddInt32(&s.tempDirCreated, 1)
	})

	return s.config.StoragePath
}
