package internal

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubRepo   = "sahil87/fab-kit"
	githubAPIURL = "https://api.github.com"
)

// Download fetches the platform-specific release archive from GitHub and
// extracts it into the version cache directory.
func Download(version string) error {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	archiveName := fmt.Sprintf("kit-%s-%s.tar.gz", osName, archName)
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", githubRepo, version, archiveName)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed (check network): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, url)
	}

	cacheDir := CacheDir(version)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("cannot create cache directory: %w", err)
	}

	if err := extractArchive(resp.Body, cacheDir); err != nil {
		// Clean up partial extraction
		os.RemoveAll(cacheDir)
		return fmt.Errorf("extraction failed: %w", err)
	}

	return nil
}

// LatestVersion queries GitHub for the latest release tag.
func LatestVersion() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIURL, githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
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

// extractArchive extracts a .tar.gz archive into the version cache directory.
// Archive contains .kit/ with fab-go at .kit/bin/fab-go and content under .kit/.
// The shim extracts:
//   - .kit/bin/fab-go -> {cacheDir}/fab-go
//   - .kit/**         -> {cacheDir}/kit/**
func extractArchive(r io.Reader, cacheDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip error: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	kitDir := filepath.Join(cacheDir, "kit")

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		name := filepath.Clean(hdr.Name)

		// .kit/bin/fab-go -> {cacheDir}/fab-go
		if name == ".kit/bin/fab-go" || name == "kit/bin/fab-go" {
			dest := filepath.Join(cacheDir, "fab-go")
			if err := writeFile(dest, tr, hdr.FileInfo().Mode()|0111); err != nil {
				return err
			}
			continue
		}

		// Skip other binaries in .kit/bin/ (wt, idea are system-only)
		if isInBinDir(name) {
			continue
		}

		// .kit/** -> {cacheDir}/kit/**
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
