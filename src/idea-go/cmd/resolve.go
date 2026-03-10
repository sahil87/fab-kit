package main

import "github.com/wvrdz/fab-kit/src/idea-go/internal/idea"

func resolveFile() (string, error) {
	repoRoot, err := idea.GitRepoRoot()
	if err != nil {
		return "", err
	}
	return idea.ResolveFilePath(repoRoot, fileFlag), nil
}
