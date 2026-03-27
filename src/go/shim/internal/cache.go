package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDir        = ".fab-kit"
	versionsSubdir  = "versions"
	runtimeRelPath  = "fab/.kit/bin/fab"
)

// CacheDir returns the root cache directory (~/.fab-kit/versions/).
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, cacheDir, versionsSubdir), nil
}

// VersionDir returns the path to a specific cached version.
func VersionDir(version string) (string, error) {
	cache, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, version), nil
}

// RuntimePath returns the path to the per-repo runtime within a cached version.
func RuntimePath(version string) (string, error) {
	vdir, err := VersionDir(version)
	if err != nil {
		return "", err
	}
	return filepath.Join(vdir, runtimeRelPath), nil
}

// IsCached checks whether a version is cached and has a valid runtime binary.
func IsCached(version string) (bool, error) {
	rpath, err := RuntimePath(version)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(rpath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
