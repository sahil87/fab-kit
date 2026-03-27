package internal

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	githubRepo   = "wvrdz/fab-kit"
	archiveBase  = "kit"
)

// DownloadURL returns the GitHub release URL for a platform-specific kit archive.
func DownloadURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/v%s/%s-%s-%s.tar.gz",
		githubRepo, version, archiveBase, runtime.GOOS, runtime.GOARCH,
	)
}

// EnsureCached downloads and caches a version if not already present.
// Returns the path to the cached runtime binary.
func EnsureCached(version string) (string, error) {
	cached, err := IsCached(version)
	if err != nil {
		return "", err
	}
	if cached {
		return RuntimePath(version)
	}

	fmt.Fprintf(os.Stderr, "Downloading fab-kit %s (%s/%s)...\n", version, runtime.GOOS, runtime.GOARCH)

	vdir, err := VersionDir(version)
	if err != nil {
		return "", err
	}

	if err := downloadAndExtract(version, vdir); err != nil {
		return "", err
	}

	return RuntimePath(version)
}

func downloadAndExtract(version, targetDir string) error {
	url := DownloadURL(version)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP %d\nCheck that version %s exists: https://github.com/%s/releases/tag/v%s",
			url, resp.StatusCode, version, githubRepo, version)
	}

	// Extract to a temp directory first, then rename for atomicity.
	tmpDir, err := os.MkdirTemp(filepath.Dir(targetDir), ".fab-kit-download-*")
	if err != nil {
		// Parent dir might not exist yet.
		if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
			return fmt.Errorf("creating cache directory: %w", err)
		}
		tmpDir, err = os.MkdirTemp(filepath.Dir(targetDir), ".fab-kit-download-*")
		if err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
	}
	defer os.RemoveAll(tmpDir) // Clean up on failure; no-op after successful rename.

	if err := extractTarGz(resp.Body, tmpDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	// Atomic move: rename temp dir to target.
	// Remove target first in case of partial previous download.
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return fmt.Errorf("installing to cache: %w", err)
	}

	return nil
}

func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		// Archives are rooted at .kit/ — prefix with fab/ to match expected layout.
		name := hdr.Name
		if strings.HasPrefix(name, ".kit/") {
			name = "fab/" + name
		}

		target := filepath.Join(destDir, name)

		// Prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}
