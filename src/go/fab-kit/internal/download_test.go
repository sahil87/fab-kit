package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

// --- Download (lock + checksum + atomic install) tests ---

// buildKitArchive returns a valid kit release tar.gz for the given version.
func buildKitArchive(t *testing.T, version string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	addTarFile(t, tw, ".kit/bin/fab-go", "#!/bin/sh\necho fab-go", 0755)
	addTarFile(t, tw, ".kit/VERSION", version+"\n", 0644)
	addTarDir(t, tw, ".kit/skills/")
	addTarFile(t, tw, ".kit/skills/test.md", "# Test Skill\n", 0644)
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// testArchiveName is the platform archive name Download will request.
func testArchiveName() string {
	return fmt.Sprintf("kit-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// serveRelease stands up an httptest release server for one version and
// points githubDownloadURL at it. sums == "" means no SHA256SUMS asset (404).
// Cleanup restores the URL and closes the server.
func serveRelease(t *testing.T, version string, archive []byte, sums string, archiveHits *atomic.Int32) {
	t.Helper()
	archivePath := fmt.Sprintf("/%s/releases/download/v%s/%s", githubRepo, version, testArchiveName())
	sumsPath := fmt.Sprintf("/%s/releases/download/v%s/SHA256SUMS", githubRepo, version)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case archivePath:
			if archiveHits != nil {
				archiveHits.Add(1)
			}
			w.Write(archive)
		case sumsPath:
			if sums == "" {
				http.NotFound(w, r)
				return
			}
			io.WriteString(w, sums)
		default:
			http.NotFound(w, r)
		}
	}))
	orig := githubDownloadURL
	githubDownloadURL = srv.URL
	t.Cleanup(func() {
		githubDownloadURL = orig
		srv.Close()
	})
}

func TestDownload_SuccessWithChecksum(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	archive := buildKitArchive(t, version)
	sums := sha256Hex(archive) + "  " + testArchiveName() + "\n"
	serveRelease(t, version, archive, sums, nil)

	if err := Download(version); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	cacheDir := CacheDir(version)
	info, err := os.Stat(filepath.Join(cacheDir, "fab-go"))
	if err != nil {
		t.Fatalf("fab-go not installed: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("fab-go should be executable")
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "kit", "VERSION")); err != nil {
		t.Errorf("kit/VERSION not installed: %v", err)
	}

	// No temp dirs or temp archives left behind; lock file remains by design.
	entries, _ := os.ReadDir(filepath.Dir(cacheDir))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") || strings.Contains(e.Name(), "-archive-") {
			t.Errorf("temp artifact left behind: %s", e.Name())
		}
	}
	if _, err := os.Stat(cacheDir + ".lock"); err != nil {
		t.Errorf("expected version lock file to exist: %v", err)
	}
}

func TestDownload_ChecksumMismatchRefusesExtraction(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	archive := buildKitArchive(t, version)
	sums := strings.Repeat("ab", 32) + "  " + testArchiveName() + "\n" // wrong digest
	serveRelease(t, version, archive, sums, nil)

	err := Download(version)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got: %v", err)
	}
	if _, statErr := os.Stat(CacheDir(version)); !os.IsNotExist(statErr) {
		t.Error("cache dir must not exist after checksum mismatch")
	}
}

func TestDownload_MissingChecksumsWarnsAndProceeds(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	archive := buildKitArchive(t, version)
	serveRelease(t, version, archive, "", nil) // no SHA256SUMS asset

	if err := Download(version); err != nil {
		t.Fatalf("Download should proceed without SHA256SUMS (pre-checksum release), got: %v", err)
	}
	if _, found := ResolveBinary(version); !found {
		t.Error("expected binary to be installed")
	}
}

func TestDownload_ChecksumsMissingEntryFails(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	archive := buildKitArchive(t, version)
	sums := sha256Hex(archive) + "  some-other-file.tar.gz\n"
	serveRelease(t, version, archive, sums, nil)

	err := Download(version)
	if err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Fatalf("expected missing-entry error, got: %v", err)
	}
	if _, statErr := os.Stat(CacheDir(version)); !os.IsNotExist(statErr) {
		t.Error("cache dir must not exist when SHA256SUMS lacks the archive entry")
	}
}

func TestDownload_ExtractionFailureLeavesExistingDirUntouched(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	garbage := []byte("this is not a gzip archive")
	sums := sha256Hex(garbage) + "  " + testArchiveName() + "\n" // digest matches, extraction fails
	serveRelease(t, version, garbage, sums, nil)

	// Pre-existing stale cache dir WITHOUT a resolvable binary (so Download
	// does not early-return) holding a marker file.
	cacheDir := CacheDir(version)
	if err := os.MkdirAll(filepath.Join(cacheDir, "kit"), 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(cacheDir, "kit", "marker.md")
	if err := os.WriteFile(marker, []byte("live\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Download(version)
	if err == nil || !strings.Contains(err.Error(), "extraction failed") {
		t.Fatalf("expected extraction failure, got: %v", err)
	}

	// Cleanup must be scoped to the temp dir: the existing dir is untouched.
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Errorf("pre-existing cache dir content was removed on failure: %v", statErr)
	}
	entries, _ := os.ReadDir(filepath.Dir(cacheDir))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp dir left behind: %s", e.Name())
		}
	}
}

func TestDownload_ConcurrentCallsFetchOnce(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	const version = "0.50.0"
	archive := buildKitArchive(t, version)
	sums := sha256Hex(archive) + "  " + testArchiveName() + "\n"
	var hits atomic.Int32
	serveRelease(t, version, archive, sums, &hits)

	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = Download(version)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Download #%d failed: %v", i, err)
		}
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("expected exactly 1 archive fetch across %d concurrent downloads, got %d", n, got)
	}
	if _, found := ResolveBinary(version); !found {
		t.Error("expected binary to be installed")
	}
}

func TestParseChecksums(t *testing.T) {
	content := "abc123  kit-linux-amd64.tar.gz\n" +
		"DEF456 *kit-darwin-arm64.tar.gz\n" +
		"\n" +
		"malformed-line\n"
	sums := parseChecksums(content)
	if sums["kit-linux-amd64.tar.gz"] != "abc123" {
		t.Errorf("standard format not parsed: %v", sums)
	}
	if sums["kit-darwin-arm64.tar.gz"] != "def456" {
		t.Errorf("binary-mode format not parsed (and lowercased): %v", sums)
	}
	if len(sums) != 2 {
		t.Errorf("expected 2 entries, got %d: %v", len(sums), sums)
	}
}

func TestHTTPClientsAreBounded(t *testing.T) {
	if apiClient.Timeout <= 0 {
		t.Error("apiClient must have a flat timeout")
	}
	if downloadClient.Timeout != 0 {
		t.Error("downloadClient must NOT have a flat timeout (streaming path)")
	}
	tr, ok := downloadClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("downloadClient must use a configured *http.Transport")
	}
	if tr.ResponseHeaderTimeout <= 0 {
		t.Error("downloadClient transport must bound time-to-response-headers")
	}
}

func TestLatestVersion_HTTPTestServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/"+githubRepo+"/releases/latest" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, `{"tag_name": "v0.51.0"}`)
	}))
	defer srv.Close()
	orig := githubAPIURL
	githubAPIURL = srv.URL
	defer func() { githubAPIURL = orig }()

	v, err := LatestVersion()
	if err != nil {
		t.Fatalf("LatestVersion failed: %v", err)
	}
	if v != "0.51.0" {
		t.Errorf("expected 0.51.0, got %s", v)
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
