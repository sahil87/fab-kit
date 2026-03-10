package main

import "github.com/wvrdz/fab-kit/src/go/idea/internal/idea"

func resolveFile() (string, error) {
	repoRoot, err := idea.GitRepoRoot()
	if err != nil {
		return "", err
	}
	return idea.ResolveFilePath(repoRoot, fileFlag), nil
}
