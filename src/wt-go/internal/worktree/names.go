package worktree

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

// Adjectives for random worktree naming (~50 adjectives).
var adjectives = []string{
	"swift", "calm", "bold", "keen", "warm", "cool", "bright", "quick",
	"brave", "noble", "wise", "kind", "fair", "proud", "sharp", "clear",
	"vivid", "agile", "sleek", "smart", "crisp", "fresh", "prime", "rapid",
	"witty", "zesty", "lucid", "happy", "sunny", "quiet", "eager", "alert",
	"nimble", "deft", "lively", "merry", "jolly", "golden", "silver", "amber",
	"cosmic", "stellar", "lunar", "solar", "rustic", "urban", "alpine", "arctic",
}

// Nouns (animals) for random worktree naming (~50 nouns).
var nouns = []string{
	"fox", "owl", "wolf", "bear", "hawk", "lynx", "puma", "orca",
	"falcon", "raven", "eagle", "crane", "heron", "finch", "robin", "sparrow",
	"otter", "beaver", "badger", "ferret", "marten", "stoat", "mink", "weasel",
	"tiger", "lion", "panther", "jaguar", "leopard", "cheetah", "cougar", "bobcat",
	"dolphin", "whale", "seal", "walrus", "penguin", "puffin", "pelican", "albatross",
	"cobra", "viper", "python", "mamba", "gecko", "iguana", "turtle", "tortoise",
}

// GenerateRandomName returns an adjective-noun combo (e.g., "swift-fox").
func GenerateRandomName() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return adj + "-" + noun
}

// GenerateUniqueName generates a random name that doesn't collide with existing
// directories in worktreesDir. It retries up to maxRetries times.
func GenerateUniqueName(worktreesDir string, maxRetries int) (string, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		name := GenerateRandomName()
		path := filepath.Join(worktreesDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return name, nil
		}
	}
	return "", fmt.Errorf("could not find unique worktree name after %d attempts", maxRetries)
}
