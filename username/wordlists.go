package username

// Wordlists for auto-assigned addresses (adjective-noun-NNNN). Every word is
// lowercase, [a-z] only (no hyphens — the shape uses '-' as the separator), and >=3
// characters so a generated name always matches autoAssignedPattern and is never in
// the paid short-name namespace. Kept deliberately friendly/neutral.
var adjectives = []string{
	"amber", "brave", "bright", "calm", "clever", "cosmic", "crisp", "daring",
	"eager", "electric", "fancy", "fuzzy", "gentle", "golden", "happy", "humble",
	"jolly", "keen", "lively", "lucky", "mellow", "merry", "mighty", "nimble",
	"noble", "polar", "proud", "quick", "quiet", "royal", "rustic", "sandy",
	"shiny", "silent", "silver", "smooth", "snowy", "solar", "spry", "stellar",
	"sunny", "swift", "tidy", "vivid", "warm", "wise", "witty", "zesty",
}

var nouns = []string{
	"otter", "badger", "falcon", "heron", "lynx", "marten", "osprey", "raven",
	"sparrow", "walrus", "beacon", "cedar", "comet", "delta", "ember", "fjord",
	"harbor", "meadow", "nebula", "orchard", "pebble", "quartz", "ridge", "summit",
	"thicket", "tundra", "valley", "willow", "anchor", "brook", "canyon", "dune",
	"forest", "geyser", "island", "jungle", "lagoon", "maple", "prairie", "reef",
	"savanna", "terrace", "vista", "wharf", "aspen", "birch", "coral", "pine",
}
