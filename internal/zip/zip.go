package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Default exclusions when zipping a directory.
var defaultExclusions = []string{
	".git",
	".gitignore",
	".github",
	".DS_Store",
	"Thumbs.db",
	"__MACOSX",
	".vscode",
	".idea",
	"node_modules",
	"package.json",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"tsconfig.json",
	".npmrc",
	"cws.toml",
}

// Default file extension exclusions.
var defaultExtExclusions = []string{
	".map",
}

// ZipDirectory creates a zip archive from a directory, excluding default patterns.
func ZipDirectory(dir string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Check exclusions
		if ShouldExclude(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Use forward slashes in zip entries
		zipPath := filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for %s: %w", relPath, err)
		}
		header.Name = zipPath
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", relPath, err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", relPath, err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to zip directory: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	return buf.Bytes(), nil
}

// ContainsManifest checks if a directory contains a manifest.json file.
func ContainsManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "manifest.json"))
	return err == nil
}

// ContainsManifestInZip checks if a zip archive contains a manifest.json file.
func ContainsManifestInZip(data []byte) (bool, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false, fmt.Errorf("failed to read zip file: %w", err)
	}

	for _, f := range reader.File {
		if f.Name == "manifest.json" || strings.HasSuffix(f.Name, "/manifest.json") {
			return true, nil
		}
	}
	return false, nil
}

// ShouldExclude checks if a file or directory should be excluded from the zip.
func ShouldExclude(relPath string, isDir bool) bool {
	base := filepath.Base(relPath)
	parts := strings.Split(filepath.ToSlash(relPath), "/")

	for _, excl := range defaultExclusions {
		// Check each path component
		for _, part := range parts {
			if part == excl {
				return true
			}
		}
		if base == excl {
			return true
		}
	}

	if !isDir {
		ext := filepath.Ext(base)
		for _, exclExt := range defaultExtExclusions {
			if ext == exclExt {
				return true
			}
		}
	}

	return false
}
