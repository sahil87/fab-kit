package worktree

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

// Adjectives for random worktree naming (~120 adjectives).
var adjectives = []string{
	// Quality & character
	"swift", "calm", "bold", "keen", "warm", "cool", "bright", "quick",
	"brave", "noble", "wise", "kind", "fair", "proud", "sharp", "clear",
	"vivid", "agile", "sleek", "smart", "crisp", "fresh", "prime", "rapid",
	"witty", "zesty", "lucid", "happy", "sunny", "quiet", "eager", "alert",
	"nimble", "deft", "lively", "merry", "jolly", "steady", "gentle", "hardy",
	// Nature & space
	"cosmic", "stellar", "lunar", "solar", "rustic", "urban", "alpine", "arctic",
	"golden", "silver", "amber", "coral", "misty", "mossy", "sandy", "snowy",
	"stormy", "windy", "frosty", "dewy", "floral", "leafy", "woody", "dusky",
	// Temperament & energy
	"fierce", "loyal", "humble", "plucky", "peppy", "perky", "chipper", "spry",
	"daring", "gutsy", "gritty", "hearty", "mighty", "robust", "rugged", "sturdy",
	// Sensory & aesthetic
	"glossy", "polished", "silky", "smooth", "velvet", "plush", "supple", "tidy",
	"azure", "ivory", "scarlet", "copper", "bronze", "jade", "onyx", "opal",
	// Motion & state
	"soaring", "rising", "roaming", "drifty", "breezy", "bubbly", "bouncy", "zippy",
	"fluid", "flowing", "gliding", "brisk", "snappy", "speedy", "vital", "fleet",
	// Abstract positive
	"clever", "handy", "savvy", "adept", "apt", "able", "ample", "grand",
}

// Nouns (animals) for random worktree naming (~120 nouns).
var nouns = []string{
	// Canids & small mammals
	"fox", "wolf", "jackal", "coyote", "dingo", "fennec", "marmot", "shrew",
	// Raptors & large birds
	"owl", "hawk", "falcon", "raven", "eagle", "condor", "osprey", "kite",
	// Songbirds & small birds
	"crane", "heron", "finch", "robin", "sparrow", "wren", "lark", "thrush",
	// Mustelids & weasel family
	"otter", "beaver", "badger", "ferret", "marten", "stoat", "mink", "weasel",
	// Big cats
	"tiger", "lion", "panther", "jaguar", "leopard", "cheetah", "cougar", "bobcat",
	// Marine mammals
	"dolphin", "whale", "seal", "walrus", "orca", "dugong", "narwhal", "beluga",
	// Reptiles
	"cobra", "viper", "python", "mamba", "gecko", "iguana", "turtle", "chameleon",
	// Bears & large mammals
	"bear", "lynx", "puma", "bison", "moose", "elk", "ibex", "yak",
	// Primates
	"lemur", "gibbon", "tamarin", "macaque", "mandrill", "langur", "bonobo", "tarsier",
	// Insects & arachnids
	"mantis", "beetle", "cicada", "cricket", "hornet", "firefly", "monarch", "earwig",
	// Fish & aquatic
	"trout", "salmon", "marlin", "barracuda", "pike", "perch", "carp", "sturgeon",
	// Ungulates & grazers
	"gazelle", "impala", "oryx", "kudu", "okapi", "tapir", "alpaca", "vicuna",
	// Miscellaneous
	"penguin", "puffin", "pelican", "stork", "hare", "hedgehog", "pangolin", "quokka",
	// Amphibians & oddities
	"newt", "axolotl", "frog", "toad", "skink", "gopher", "mole", "ermine",
	// Additional birds
	"jay", "magpie", "plover", "curlew", "grouse", "quail", "pipit", "tern",
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
