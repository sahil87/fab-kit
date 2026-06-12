package internal

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const githubRepo = "sahil87/fab-kit"

// URL bases are package vars so tests can point them at httptest servers.
var (
	githubAPIURL      = "https://api.github.com"
	githubDownloadURL = "https://github.com"
)

// HTTP bounds for the auto-download path. LatestVersion and the SHA256SUMS
// fetch carry small bodies, so a flat end-to-end timeout is safe. The archive
// download streams a large body — a flat timeout would abort legitimately
// slow downloads, so it is bounded by time-to-response-headers plus a
// generous overall context deadline instead.
const (
	apiTimeout             = 30 * time.Second
	downloadConnectTimeout = 30 * time.Second // dial + TLS handshake
	downloadHeaderTimeout  = 30 * time.Second
	downloadTotalTimeout   = 10 * time.Minute
)

// Naming for the version cache's sibling artifacts under versions/: the
// per-version download lock, the hashed temp archive, and the temp extraction
// dir that is renamed into place.
const (
	lockFileSuffix    = ".lock"        // versions/<version>.lock
	tmpArchivePattern = ".%s-archive-" // CreateTemp prefix; %s = version
	tmpDirPattern     = "%s.tmp-%d"    // %s = CacheDir(version); %d = pid
)

// apiClient serves small-body requests (release metadata, checksums).
var apiClient = &http.Client{Timeout: apiTimeout}

// downloadClient serves the streaming archive download. No flat Timeout —
// the overall deadline comes from the request context in Download.
var downloadClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: downloadConnectTimeout}).DialContext,
		TLSHandshakeTimeout:   downloadConnectTimeout,
		ResponseHeaderTimeout: downloadHeaderTimeout,
	},
}

// Download fetches the platform-specific release archive from GitHub,
// verifies its SHA-256 digest against the release's SHA256SUMS asset, and
// atomically installs it into the version cache directory.
//
// Concurrency: downloaders of the same version serialize on an exclusive
// advisory flock on versions/<version>.lock. Extraction happens in a
// temp dir (versions/<version>.tmp-<pid>) that is renamed into place only
// after the archive is fully verified and extracted, so readiness of the
// cache dir is all-or-nothing. Error cleanup is scoped to the temp dir —
// a failed download never removes a live cache dir.
func Download(version string) error {
	archiveName := fmt.Sprintf("kit-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("%s/%s/releases/download/v%s/%s", githubDownloadURL, githubRepo, version, archiveName)

	cacheDir := CacheDir(version)
	versionsDir := filepath.Dir(cacheDir)
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("cannot create cache directory: %w", err)
	}

	// Serialize concurrent downloaders on a version-keyed sibling lock file.
	unlock, err := acquireLock(cacheDir + lockFileSuffix)
	if err != nil {
		return err
	}
	defer unlock()

	// Re-check after waiting on the lock — a concurrent process may have
	// completed the download while we blocked.
	if _, found := ResolveBinary(version); found {
		return nil
	}

	// Fetch expected digests from the same release. nil sums (asset absent)
	// means a pre-checksum release — skip verification with a warning.
	sums, err := fetchChecksums(version)
	if err != nil {
		return err
	}
	wantDigest := ""
	if sums == nil {
		fmt.Fprintf(os.Stderr, "WARNING: release v%s publishes no SHA256SUMS asset — skipping checksum verification\n", version)
	} else {
		var ok bool
		wantDigest, ok = sums[archiveName]
		if !ok {
			return fmt.Errorf("SHA256SUMS for v%s has no entry for %s — refusing to install", version, archiveName)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTotalTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed (check network): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, url)
	}

	// Stream the archive to a temp file while hashing, so the digest is
	// verified BEFORE any byte is extracted.
	tmpArchive, err := os.CreateTemp(versionsDir, fmt.Sprintf(tmpArchivePattern, version))
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmpArchive.Name())
	defer tmpArchive.Close()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmpArchive, hasher), resp.Body); err != nil {
		return fmt.Errorf("download failed (check network): %w", err)
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if wantDigest != "" && digest != wantDigest {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s — refusing to extract", archiveName, wantDigest, digest)
	}
	if _, err := tmpArchive.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("cannot rewind temp file: %w", err)
	}

	// Extract into a temp dir, then atomically rename into place.
	tmpDir := fmt.Sprintf(tmpDirPattern, cacheDir, os.Getpid())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("cannot create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir) // no-op after a successful rename; never touches cacheDir

	if err := extractArchive(tmpArchive, tmpDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// A pre-existing cache dir here is stale (it has no resolvable fab-go —
	// checked above under the lock) — replace it so the rename can land.
	if dirExists(cacheDir) {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("cannot replace stale cache directory %s: %w", cacheDir, err)
		}
	}
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		return fmt.Errorf("cannot finalize cache directory: %w", err)
	}

	return nil
}

// fetchChecksums downloads the SHA256SUMS asset for a release and returns a
// map of asset filename -> expected hex digest. A 404 means the release
// predates checksum publishing: returns (nil, nil) so the caller can skip
// verification with a warning. Any other failure is an error.
func fetchChecksums(version string) (map[string]string, error) {
	url := fmt.Sprintf("%s/%s/releases/download/v%s/SHA256SUMS", githubDownloadURL, githubRepo, version)

	resp, err := apiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch SHA256SUMS (check network): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SHA256SUMS fetch failed: HTTP %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("cannot read SHA256SUMS: %w", err)
	}
	return parseChecksums(string(body)), nil
}

// parseChecksums parses sha256sum-format lines ("<hex>  <name>", optionally
// "<hex> *<name>" for binary mode) into a filename -> digest map.
func parseChecksums(content string) map[string]string {
	sums := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		sums[name] = strings.ToLower(fields[0])
	}
	return sums
}

// LatestVersion queries GitHub for the latest release tag.
func LatestVersion() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIURL, githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("could not parse GitHub response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("could not parse latest release tag from GitHub response")
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// extractArchive extracts a .tar.gz archive into the given directory.
// Archive contains .kit/ with fab-go at .kit/bin/fab-go and content under .kit/.
// The shim extracts:
//   - .kit/bin/fab-go -> {destDir}/fab-go
//   - .kit/**         -> {destDir}/kit/**
func extractArchive(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip error: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	kitDir := filepath.Join(destDir, "kit")

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		name := filepath.Clean(hdr.Name)

		// .kit/bin/fab-go -> {destDir}/fab-go
		if name == ".kit/bin/fab-go" || name == "kit/bin/fab-go" {
			dest := filepath.Join(destDir, "fab-go")
			if err := writeFile(dest, tr, hdr.FileInfo().Mode()|0111); err != nil {
				return err
			}
			continue
		}

		// Skip other binaries in .kit/bin/ (wt, idea are system-only)
		if isInBinDir(name) {
			continue
		}

		// .kit/** -> {destDir}/kit/**
		var relPath string
		if strings.HasPrefix(name, ".kit/") {
			relPath = strings.TrimPrefix(name, ".kit/")
		} else if strings.HasPrefix(name, "kit/") {
			relPath = strings.TrimPrefix(name, "kit/")
		} else {
			continue // skip files outside .kit/
		}

		dest := filepath.Join(kitDir, relPath)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			if err := writeFile(dest, tr, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}

func isInBinDir(name string) bool {
	return strings.HasPrefix(name, ".kit/bin/") || strings.HasPrefix(name, "kit/bin/")
}

func writeFile(path string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
