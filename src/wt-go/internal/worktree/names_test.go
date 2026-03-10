package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRandomName_Format(t *testing.T) {
	for i := 0; i < 50; i++ {
		name := GenerateRandomName()
		parts := strings.SplitN(name, "-", 2)
		if len(parts) != 2 {
			t.Errorf("GenerateRandomName() = %q, want adjective-noun format", name)
		}
		if parts[0] == "" || parts[1] == "" {
			t.Errorf("GenerateRandomName() = %q, parts cannot be empty", name)
		}
	}
}

func TestGenerateRandomName_Variety(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := GenerateRandomName()
		seen[name] = true
	}
	// With 48*48=2304 combos, 100 draws should produce at least 50 unique names
	if len(seen) < 50 {
		t.Errorf("GenerateRandomName() produced only %d unique names in 100 draws, expected more variety", len(seen))
	}
}

func TestGenerateUniqueName_Success(t *testing.T) {
	dir := t.TempDir()
	name, err := GenerateUniqueName(dir, 10)
	if err != nil {
		t.Fatalf("GenerateUniqueName() error: %v", err)
	}
	if name == "" {
		t.Error("GenerateUniqueName() returned empty name")
	}
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 {
		t.Errorf("GenerateUniqueName() = %q, want adjective-noun format", name)
	}
}

func TestGenerateUniqueName_RetryExhaustion(t *testing.T) {
	dir := t.TempDir()

	// Create directories for all possible name combinations
	for _, adj := range adjectives {
		for _, noun := range nouns {
			name := adj + "-" + noun
			os.MkdirAll(filepath.Join(dir, name), 0755)
		}
	}

	_, err := GenerateUniqueName(dir, 10)
	if err == nil {
		t.Error("GenerateUniqueName() should fail when all names collide")
	}
}

func TestWordListsNonEmpty(t *testing.T) {
	if len(adjectives) == 0 {
		t.Error("adjectives list is empty")
	}
	if len(nouns) == 0 {
		t.Error("nouns list is empty")
	}
	if len(adjectives) < 40 {
		t.Errorf("adjectives list has only %d entries, expected ~50", len(adjectives))
	}
	if len(nouns) < 40 {
		t.Errorf("nouns list has only %d entries, expected ~50", len(nouns))
	}
}
