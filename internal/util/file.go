package util

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hungdv136/rio/internal/log"
)

// Unzip decompress a file into an output directory
func Unzip(ctx context.Context, inputPath string, outputDir string) error {
	archive, err := zip.OpenReader(inputPath)
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	defer CloseSilently(ctx, archive.Close)

	log.Info(ctx, "nb files", len(archive.File))

	for _, f := range archive.File {
		filePath, err := SanitizePath(ctx, outputDir, f.Name)
		if err != nil {
			log.Error(ctx, err)
			return err
		}

		if f.FileInfo().IsDir() {
			continue
		}

		log.Info(ctx, "unzipping file", filePath)

		fileInArchive, err := f.Open()
		if err != nil {
			log.Error(ctx, err)
			return err
		}

		if err := WriteToFile(ctx, fileInArchive, filePath); err != nil {
			return err
		}

		CloseSilently(ctx, fileInArchive.Close)
	}

	return nil
}

// WriteToFile writes to file
func WriteToFile(ctx context.Context, reader io.Reader, path string) error {
	dir := filepath.Dir(path)
	if err := EnsureDirExist(ctx, dir); err != nil {
		return err
	}

	outputFile, err := os.Create(path)
	if err != nil {
		log.Error(ctx, "cannot create file", path)
		return err
	}
	defer CloseSilently(ctx, outputFile.Close)

	if _, err := io.Copy(outputFile, reader); err != nil {
		log.Error(ctx, "cannot write file", path)
		return err
	}

	return nil
}

// EnsureDirExist creates a directory if it is not existed
func EnsureDirExist(ctx context.Context, dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Error(ctx, "cannot create directory", err)
			return err
		}

		log.Info(ctx, "created dir", dir)
		return nil
	}

	return err
}

// SanitizePath is to void "G305: Zip Slip vulnerability"
func SanitizePath(ctx context.Context, d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	err = fmt.Errorf("%s: %s", "content filepath is tainted", t)
	log.Error(ctx, err)
	return "", err
}
