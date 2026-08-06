package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaughnbosu/cws-cli/pkg/config"
	"github.com/vaughnbosu/cws-cli/pkg/manifest"
	cwszip "github.com/vaughnbosu/cws-cli/pkg/zip"
)

// PackResult is the output of packing an extension directory.
type PackResult struct {
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Pack zips an extension directory and writes it to disk.
func Pack(source, output string, pkg config.PackageConfig) (*PackResult, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return nil, fmt.Errorf("source not found: %s", source)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("pack requires a directory source, got a file: %s", source)
	}

	manifestPath := filepath.Join(absSource, "manifest.json")
	m, err := manifest.Parse(manifestPath)
	if err != nil {
		return nil, err
	}

	data, err := cwszip.ZipDirectoryWithOptions(absSource, zipOptions(pkg))
	if err != nil {
		return nil, err
	}

	outPath := output
	if outPath == "" {
		name := sanitizeFilename(m.Name)
		if name == "" {
			name = "extension"
		}
		outPath = fmt.Sprintf("%s-%s.zip", name, m.Version)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return &PackResult{
		Path:    outPath,
		Bytes:   len(data),
		Name:    m.Name,
		Version: m.Version,
	}, nil
}

// sanitizeFilename converts an extension name to a safe file name fragment.
func sanitizeFilename(name string) string {
	var b []rune
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b = append(b, r)
		case r == ' ':
			b = append(b, '-')
		}
	}
	return string(b)
}
