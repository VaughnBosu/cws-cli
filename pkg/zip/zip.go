package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
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
	".zip",
	".crx",
}

// Options controls zip packaging behavior.
type Options struct {
	// Exclude adds patterns (path components like "docs" or extensions like ".log")
	// to the default exclusion list.
	Exclude []string
	// Include keeps files a default exclusion would otherwise drop (matched against
	// path components and extensions, e.g. "package.json" or ".map").
	Include []string
}

// ZipDirectory creates a zip archive from a directory, excluding default patterns.
// Symlinks inside the tree are skipped to prevent directory traversal.
func ZipDirectory(dir string) ([]byte, error) {
	return ZipDirectoryWithOptions(dir, Options{})
}

// ZipDirectoryWithOptions is ZipDirectory with configurable exclusions.
func ZipDirectoryWithOptions(dir string, opts Options) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Resolve a symlinked source directory (e.g. dist -> build/output) so the
	// walk descends into the real tree instead of skipping the root symlink.
	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	fileCount := 0
	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip symlinks to prevent directory traversal
		if d.Type()&os.ModeSymlink != 0 {
			return nil
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
		if ShouldExcludeWithOptions(relPath, d.IsDir(), opts) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}
		fileCount++

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", relPath, err)
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
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to read %s: %w", relPath, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close %s: %w", relPath, closeErr)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to zip directory: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	if fileCount == 0 {
		return nil, fmt.Errorf("no files to package in %s (everything was excluded or the directory is empty)", dir)
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
	return ContainsFileInZip(data, "manifest.json")
}

// ContainsFileInZip checks if a zip archive contains the named file (root-relative,
// forward-slash path).
func ContainsFileInZip(data []byte, name string) (bool, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false, fmt.Errorf("failed to read zip file: %w", err)
	}

	for _, f := range reader.File {
		if f.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// ShouldExclude checks if a file or directory should be excluded from the zip.
func ShouldExclude(relPath string, isDir bool) bool {
	return ShouldExcludeWithOptions(relPath, isDir, Options{})
}

// ShouldExcludeWithOptions is ShouldExclude with configurable exclusions.
// Include entries win over both default and configured exclusions.
func ShouldExcludeWithOptions(relPath string, isDir bool, opts Options) bool {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	base := filepath.Base(relPath)
	ext := filepath.Ext(base)

	matches := func(pattern string) bool {
		if !isDir && strings.HasPrefix(pattern, ".") && pattern == ext {
			return true
		}
		for _, part := range parts {
			if part == pattern {
				return true
			}
		}
		return false
	}

	for _, inc := range opts.Include {
		if matches(inc) {
			return false
		}
	}

	for _, excl := range append(defaultExclusions, opts.Exclude...) {
		for _, part := range parts {
			if part == excl {
				return true
			}
		}
	}

	if !isDir {
		for _, exclExt := range append(defaultExtExclusions, opts.Exclude...) {
			if ext == exclExt {
				return true
			}
		}
	}

	return false
}
