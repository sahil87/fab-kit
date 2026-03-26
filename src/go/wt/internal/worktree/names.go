package worktree

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

// Adjectives for random worktree naming (~200 adjectives).
var adjectives = []string{
	// Quality & character
	"swift", "calm", "bold", "keen", "warm", "cool", "bright", "quick",
	"brave", "noble", "wise", "kind", "fair", "proud", "sharp", "clear",
	"vivid", "agile", "sleek", "smart", "crisp", "fresh", "prime", "rapid",
	"witty", "zesty", "lucid", "happy", "sunny", "quiet", "eager", "alert",
	"nimble", "deft", "lively", "merry", "jolly", "steady", "gentle", "hardy",
	"honest", "stable", "earnest", "candid", "astute", "poised", "frank", "stout",
	// Nature & space
	"cosmic", "stellar", "lunar", "solar", "rustic", "urban", "alpine", "arctic",
	"golden", "silver", "amber", "coral", "misty", "mossy", "sandy", "snowy",
	"stormy", "windy", "frosty", "dewy", "floral", "leafy", "woody", "dusky",
	"earthy", "marine", "polar", "boreal", "verdant", "sylvan", "astral", "coastal",
	// Temperament & energy
	"fierce", "loyal", "humble", "plucky", "peppy", "perky", "chipper", "spry",
	"daring", "gutsy", "gritty", "hearty", "mighty", "robust", "rugged", "sturdy",
	"feisty", "driven", "ardent", "staunch", "lofty", "tireless", "gallant", "valiant",
	// Sensory & aesthetic
	"glossy", "polished", "silky", "smooth", "velvet", "plush", "supple", "tidy",
	"azure", "ivory", "scarlet", "copper", "bronze", "jade", "onyx", "opal",
	"pastel", "tinted", "matte", "pearly", "gilded", "dainty", "ornate", "prismic",
	// Motion & state
	"soaring", "rising", "roaming", "drifty", "breezy", "bubbly", "bouncy", "zippy",
	"fluid", "flowing", "gliding", "brisk", "snappy", "speedy", "vital", "fleet",
	"surging", "darting", "coasting", "spinning", "pacing", "racing", "winding", "arcing",
	// Abstract positive
	"clever", "handy", "savvy", "adept", "apt", "able", "ample", "grand",
	"dapper", "suave", "pliant", "benign", "genial", "cordial", "refined", "serene",
	// Time & weather
	"vernal", "autumn", "balmy", "sultry", "briny", "humid", "torrid", "frigid",
	"nightly", "twilit", "dawning", "sunlit", "cloudy", "blustery", "wintry", "mild",
	// Texture & material
	"woven", "braided", "knitted", "felted", "hewn", "carved", "molten", "tempered",
	"grainy", "chalky", "pebbly", "glazed", "buffed", "frosted", "russet", "dappled",
}

// Nouns for random worktree naming (~200 nouns).
var nouns = []string{
	// Canids & small mammals
	"fox", "wolf", "jackal", "coyote", "dingo", "fennec", "marmot", "shrew",
	"vole", "pika", "stoat", "badger",
	// Raptors & large birds
	"owl", "hawk", "falcon", "raven", "eagle", "condor", "osprey", "kite",
	"harrier", "merlin", "goshawk", "buzzard",
	// Songbirds & small birds
	"crane", "heron", "finch", "robin", "sparrow", "wren", "lark", "thrush",
	"warbler", "swift", "starling", "oriole",
	// Mustelids & weasel family
	"otter", "beaver", "ferret", "marten", "mink", "weasel", "wolverine", "fisher",
	"sable", "ermine", "polecat",
	// Big cats
	"tiger", "lion", "panther", "jaguar", "leopard", "cheetah", "cougar", "bobcat",
	"ocelot", "serval", "caracal", "margay",
	// Marine animals
	"dolphin", "whale", "seal", "walrus", "orca", "dugong", "narwhal", "beluga",
	"manatee", "porpoise", "seastar", "urchin",
	// Reptiles
	"cobra", "viper", "python", "mamba", "gecko", "iguana", "turtle", "chameleon",
	"monitor", "tortoise", "anole", "taipan",
	// Bears & large mammals
	"bear", "lynx", "puma", "bison", "moose", "elk", "ibex", "yak",
	"wisent", "buffalo", "mammoth", "rhino",
	// Primates
	"lemur", "gibbon", "tamarin", "macaque", "mandrill", "langur", "bonobo", "tarsier",
	"howler", "capuchin", "marmoset", "spider",
	// Insects & arachnids
	"mantis", "beetle", "cicada", "cricket", "hornet", "firefly", "monarch", "earwig",
	"damsel", "mayfly", "katydid", "weevil",
	// Fish & aquatic
	"trout", "salmon", "marlin", "barracuda", "pike", "perch", "carp", "sturgeon",
	"minnow", "guppy", "tetra", "darter",
	// Ungulates & grazers
	"gazelle", "impala", "oryx", "kudu", "okapi", "tapir", "alpaca", "vicuna",
	"chamois", "bharal", "saola", "eland",
	// Miscellaneous
	"penguin", "puffin", "pelican", "stork", "hare", "hedgehog", "pangolin", "quokka",
	"platypus", "numbat", "bilby", "wombat",
	// Amphibians & oddities
	"newt", "axolotl", "frog", "toad", "skink", "gopher", "mole", "salamander",
	"bullfrog", "treefrog", "mudpup", "caecilian",
	// Additional birds
	"jay", "magpie", "plover", "curlew", "grouse", "quail", "pipit", "tern",
	"dunlin", "avocet", "fulmar", "gannet",
	// Nature & geography
	"river", "summit", "canyon", "reef", "grove", "meadow", "delta", "ridge",
	"lagoon", "tundra", "steppe", "butte", "fjord", "ravine", "basin", "marsh",
	"bluff", "dune", "atoll", "geyser", "glacier",
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
