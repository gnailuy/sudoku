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
	"strategy.unique-rectangle.name":   "Unique Rectangle",
	"strategy.unique-rectangle-2.name": "Unique Rectangle Type 2",
	"strategy.unique-rectangle-3.name": "Unique Rectangle Type 3",
	"strategy.unique-rectangle-4.name": "Unique Rectangle Type 4",
	"strategy.x-cycles.name":           "X-Cycles",
	"strategy.xy-chain.name":           "XY-Chain",

	"hint.value-correction.title":     "Correct a value first",
	"hint.candidate-correction.title": "Reset one candidate cell first",
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
