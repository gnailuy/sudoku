package message

import "fmt"

var englishConstantMessages = map[string]string{
	"difficulty.easy.name":   "Easy",
	"difficulty.medium.name": "Medium",
	"difficulty.hard.name":   "Hard",
	"difficulty.expert.name": "Expert",
	"difficulty.evil.name":   "Evil",

	"strategy.naked-single.name":       "Naked Single",
	"strategy.hidden-single.name":      "Hidden Single",
	"strategy.naked-pair.name":         "Naked Pair",
	"strategy.naked-triple.name":       "Naked Triple",
	"strategy.pointing-pair.name":      "Pointing Pairs / Box-Line Reduction",
	"strategy.hidden-pair.name":        "Hidden Pair",
	"strategy.x-wing.name":             "X-Wing",
	"strategy.xy-wing.name":            "XY-Wing",
	"strategy.hidden-triple.name":      "Hidden Triple",
	"strategy.w-wing.name":             "W-Wing",
	"strategy.swordfish.name":          "Swordfish",
	"strategy.naked-quad.name":         "Naked Quad",
	"strategy.simple-coloring.name":    "Simple Coloring",
	"strategy.hidden-quad.name":        "Hidden Quad",
	"strategy.xyz-wing.name":           "XYZ-Wing",
	"strategy.jellyfish.name":          "Jellyfish",
	"strategy.bug-plus-one.name":       "BUG+1",
	"strategy.unique-rectangle.name":   "Unique Rectangle Type 1",
	"strategy.unique-rectangle-2.name": "Unique Rectangle Type 2",
	"strategy.unique-rectangle-3.name": "Unique Rectangle Type 3",
	"strategy.unique-rectangle-4.name": "Unique Rectangle Type 4",
	"strategy.x-cycles.name":           "X-Cycles",
	"strategy.xy-chain.name":           "XY-Chain",

	"hint.value-correction.title":     "Fix this value",
	"hint.candidate-correction.title": "Fix this candidate cell",

	"error.invalid-request":         "The request is invalid.",
	"error.invalid-json":            "The request contains invalid JSON.",
	"error.unsupported-media-type":  "The request Content-Type is not supported.",
	"error.payload-too-large":       "The request is too large.",
	"error.unauthorized":            "Authorization is required.",
	"error.forbidden-origin":        "This request origin is not allowed.",
	"error.session-not-found":       "The game session was not found.",
	"error.revision-conflict":       "The game changed before this action was applied.",
	"error.invalid-session":         "The game session is invalid.",
	"error.invalid-action":          "This game action is invalid.",
	"error.invalid-cell":            "This cell is invalid.",
	"error.immutable-cell":          "A given cell cannot be changed.",
	"error.note-not-allowed":        "Candidates cannot be changed in this cell.",
	"error.no-undo":                 "There is no action to undo.",
	"error.no-redo":                 "There is no action to redo.",
	"error.no-hint":                 "No hint is available.",
	"error.stale-hint":              "The puzzle changed after this hint was prepared.",
	"error.persistence-failed":      "The game could not be saved.",
	"error.internal-error":          "The request could not be completed.",
	"error.account-unavailable":     "Account features are unavailable.",
	"error.account-unauthorized":    "A valid application session is required.",
	"error.csrf-failed":             "The request proof is invalid.",
	"error.account-game-not-found":  "The saved game was not found.",
	"error.guest-revision-conflict": "The local game changed before this action was applied.",
	"error.guest-claim-conflict":    "This local game was already saved to an account.",
}

// NewEnglishCatalog returns the canonical English catalog shared by API and terminal adapters.
func NewEnglishCatalog() (*Catalog, error) {
	definitions := make(map[string]Definition, len(englishConstantMessages))
	for id, text := range englishConstantMessages {
		text := text
		definitions[id] = Definition{Render: func([]Argument) (string, error) { return text, nil }}
	}
	return NewCatalog(definitions)
}

// RenderEnglishName renders a stable difficulty or strategy identifier.
func RenderEnglishName(catalog *Catalog, domain, id string) (Descriptor, error) {
	if domain != "difficulty" && domain != "strategy" {
		return Descriptor{}, fmt.Errorf("unsupported name domain %q", domain)
	}
	return catalog.Render(domain + "." + id + ".name")
}
