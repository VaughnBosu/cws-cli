package tests

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cwszip "github.com/vaughnbosu/cws-cli/pkg/zip"
)

// Regression test: a symlinked source directory must be resolved and zipped,
// not silently skipped (which produced a 22-byte empty archive).
func TestZipDirectory_SymlinkedRoot(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0755); err != nil {
		t.Fatal(err)
	}
	createFile(t, filepath.Join(real, "manifest.json"), `{"name":"x","version":"1.0","manifest_version":3}`)

	link := filepath.Join(base, "dist")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	data, err := cwszip.ZipDirectory(link)
	if err != nil {
		t.Fatalf("ZipDirectory error: %v", err)
	}

	names := zipEntryNames(t, data)
	if len(names) != 1 || names[0] != "manifest.json" {
		t.Errorf("zip entries = %v, want [manifest.json]", names)
	}
}

// Regression test: an all-excluded (or empty) directory must error rather than
// produce an empty archive that uploads and fails store-side.
func TestZipDirectory_EmptyResultErrors(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, ".DS_Store"), "junk")

	_, err := cwszip.ZipDirectory(dir)
	if err == nil {
		t.Fatal("expected error for zip with no files, got nil")
	}
	if !strings.Contains(err.Error(), "no files") {
		t.Errorf("error = %q, want mention of no files", err.Error())
	}
}

func TestZipDirectoryWithOptions_IncludeOverridesDefaultExclusion(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "manifest.json"), "{}")
	createFile(t, filepath.Join(dir, "package.json"), "{}")
	createFile(t, filepath.Join(dir, "fixture.zip"), "archive")

	data, err := cwszip.ZipDirectoryWithOptions(dir, cwszip.Options{Include: []string{"package.json", ".zip"}})
	if err != nil {
		t.Fatalf("ZipDirectoryWithOptions error: %v", err)
	}

	names := zipEntryNames(t, data)
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"package.json", "fixture.zip"} {
		if !found[want] {
			t.Errorf("zip entries = %v, want %s kept via Include", names, want)
		}
	}
}

func TestZipDirectory_ExcludesPackageArtifacts(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "manifest.json"), "{}")
	createFile(t, filepath.Join(dir, "previous.zip"), "archive")
	createFile(t, filepath.Join(dir, "previous.crx"), "archive")

	data, err := cwszip.ZipDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if names := zipEntryNames(t, data); len(names) != 1 || names[0] != "manifest.json" {
		t.Fatalf("zip entries = %v, want only manifest.json", names)
	}
}

func TestZipDirectoryWithOptions_ExtraExclusions(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "manifest.json"), "{}")
	createFile(t, filepath.Join(dir, "notes.txt"), "internal")
	createFile(t, filepath.Join(dir, "debug.log"), "log")

	data, err := cwszip.ZipDirectoryWithOptions(dir, cwszip.Options{Exclude: []string{"notes.txt", ".log"}})
	if err != nil {
		t.Fatalf("ZipDirectoryWithOptions error: %v", err)
	}

	names := zipEntryNames(t, data)
	if len(names) != 1 || names[0] != "manifest.json" {
		t.Errorf("zip entries = %v, want only manifest.json", names)
	}
}

// --- CRX extraction tests ---

// buildCRX3 wraps zip bytes in a minimal CRX3 container.
func buildCRX3(zipData []byte, headerLen int) []byte {
	var buf bytes.Buffer
	buf.WriteString("Cr24")
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	binary.Write(&buf, binary.LittleEndian, uint32(headerLen))
	buf.Write(make([]byte, headerLen))
	buf.Write(zipData)
	return buf.Bytes()
}

func TestExtractZipFromCRX_CRX3(t *testing.T) {
	zipData := buildZipWithManifest(t)
	crx := buildCRX3(zipData, 64)

	got, err := cwszip.ExtractZipFromCRX(crx)
	if err != nil {
		t.Fatalf("ExtractZipFromCRX error: %v", err)
	}
	if !bytes.Equal(got, zipData) {
		t.Error("extracted zip does not match original")
	}
	has, err := cwszip.ContainsManifestInZip(got)
	if err != nil || !has {
		t.Errorf("extracted zip should contain manifest.json (has=%v, err=%v)", has, err)
	}
}

func TestExtractZipFromCRX_CRX2(t *testing.T) {
	zipData := buildZipWithManifest(t)
	var buf bytes.Buffer
	buf.WriteString("Cr24")
	binary.Write(&buf, binary.LittleEndian, uint32(2))
	binary.Write(&buf, binary.LittleEndian, uint32(8))  // pubkey len
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // sig len
	buf.Write(make([]byte, 24))
	buf.Write(zipData)

	got, err := cwszip.ExtractZipFromCRX(buf.Bytes())
	if err != nil {
		t.Fatalf("ExtractZipFromCRX error: %v", err)
	}
	if !bytes.Equal(got, zipData) {
		t.Error("extracted zip does not match original")
	}
}

func TestExtractZipFromCRX_NotCRX(t *testing.T) {
	if _, err := cwszip.ExtractZipFromCRX([]byte("PK\x03\x04zipdata")); err == nil {
		t.Fatal("expected error for non-CRX input")
	}
}

func TestExtractZipFromCRX_TruncatedHeader(t *testing.T) {
	crx := buildCRX3(nil, 0)
	// Claim a header larger than the file
	binary.LittleEndian.PutUint32(crx[8:12], 9999)
	if _, err := cwszip.ExtractZipFromCRX(crx); err == nil {
		t.Fatal("expected error for truncated CRX header")
	}
}

func TestIsCRX(t *testing.T) {
	if !cwszip.IsCRX([]byte("Cr24rest")) {
		t.Error("IsCRX should detect Cr24 magic")
	}
	if cwszip.IsCRX([]byte("PK\x03\x04")) {
		t.Error("IsCRX should reject zip magic")
	}
}

// --- helpers ---

func buildZipWithManifest(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte(`{"name":"x","version":"1.0","manifest_version":3}`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
