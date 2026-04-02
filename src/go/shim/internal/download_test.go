package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchive(t *testing.T) {
	// Build a tar.gz archive in memory with the expected structure
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add .kit/bin/fab-go (binary)
	addTarFile(t, tw, ".kit/bin/fab-go", "#!/bin/sh\necho fab-go", 0755)

	// Add .kit/bin/wt (should be skipped)
	addTarFile(t, tw, ".kit/bin/wt", "#!/bin/sh\necho wt", 0755)

	// Add .kit/VERSION
	addTarFile(t, tw, ".kit/VERSION", "0.43.0\n", 0644)

	// Add .kit/skills/test.md
	addTarDir(t, tw, ".kit/skills/")
	addTarFile(t, tw, ".kit/skills/test.md", "# Test Skill\n", 0644)

	tw.Close()
	gw.Close()

	// Extract to temp dir
	cacheDir := t.TempDir()
	if err := extractArchive(&buf, cacheDir); err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify fab-go was extracted to cacheDir/fab-go
	fabGo := filepath.Join(cacheDir, "fab-go")
	info, err := os.Stat(fabGo)
	if err != nil {
		t.Fatalf("fab-go not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("fab-go should be executable")
	}

	// Verify kit/VERSION was extracted
	versionFile := filepath.Join(cacheDir, "kit", "VERSION")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("kit/VERSION not found: %v", err)
	}
	if string(data) != "0.43.0\n" {
		t.Errorf("expected VERSION content '0.43.0\\n', got '%s'", string(data))
	}

	// Verify kit/skills/test.md was extracted
	skillFile := filepath.Join(cacheDir, "kit", "skills", "test.md")
	data, err = os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("kit/skills/test.md not found: %v", err)
	}
	if string(data) != "# Test Skill\n" {
		t.Errorf("unexpected skill content: '%s'", string(data))
	}

	// Verify wt was NOT extracted (system-only binary)
	wtPath := filepath.Join(cacheDir, "kit", "bin", "wt")
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("wt should not be extracted (system-only)")
	}
	wtPath2 := filepath.Join(cacheDir, "wt")
	if _, err := os.Stat(wtPath2); !os.IsNotExist(err) {
		t.Error("wt should not be extracted to cache root")
	}
}

func TestLatestVersionParsing(t *testing.T) {
	// Verify the JSON struct unmarshals tag_name correctly
	// (LatestVersion itself hits the network, so we test the parsing shape)
	body := []byte(`{"tag_name": "v0.43.0", "id": 123, "name": "test"}`)
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if release.TagName != "v0.43.0" {
		t.Errorf("expected tag_name 'v0.43.0', got '%s'", release.TagName)
	}
}

// addTarFile adds a regular file to a tar writer.
func addTarFile(t *testing.T, tw *tar.Writer, name, content string, mode int64) {
	t.Helper()
	hdr := &tar.Header{
		Name: name,
		Mode: mode,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}

// addTarDir adds a directory entry to a tar writer.
func addTarDir(t *testing.T, tw *tar.Writer, name string) {
	t.Helper()
	hdr := &tar.Header{
		Name:     name,
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
}
