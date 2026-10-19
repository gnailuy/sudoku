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
	"strategy.backtracker.name":        "Backtracker",

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
	addEnglishHintMessages(definitions)
	return NewCatalog(definitions)
}

func addEnglishHintMessages(definitions map[string]Definition) {
	constant := func(id, text string) {
		definitions[id] = Definition{Render: func([]Argument) (string, error) { return text, nil }}
	}
	one := func(id, name string, kind ArgumentKind, render func(Argument) string) {
		definitions[id] = Definition{
			Arguments: []ArgumentSpec{{Name: name, Kind: kind}},
			Render:    func(arguments []Argument) (string, error) { return render(arguments[0]), nil },
		}
	}
	two := func(id string, first ArgumentSpec, second ArgumentSpec, render func(Argument, Argument) string) {
		definitions[id] = Definition{
			Arguments: []ArgumentSpec{first, second},
			Render:    func(arguments []Argument) (string, error) { return render(arguments[0], arguments[1]), nil },
		}
	}
	cell := func(argument Argument) string {
		return fmt.Sprintf("row %d, column %d", argument.Cell.Row, argument.Cell.Column)
	}
	strategy := func(argument Argument) string {
		if name, ok := englishConstantMessages["strategy."+*argument.Strategy+".name"]; ok {
			return name
		}
		return *argument.Strategy
	}

	one("hint.value-correction.summary", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("The value at %s conflicts with the puzzle. Clear it before continuing.", cell(argument))
	})
	one("hint.candidate-correction.summary", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("Reset the candidate set at %s from the accepted row, column, and box values before continuing.", cell(argument))
	})
	two("hint.strategy.summary", ArgumentSpec{Name: "steps", Kind: ArgumentCount}, ArgumentSpec{Name: "strategy", Kind: ArgumentStrategy}, func(steps, name Argument) string {
		return fmt.Sprintf("Follow the %d teaching steps to apply the %s deduction.", *steps.Count, strategy(name))
	})
	one("hint.value-correction.observe", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("Look at %s. Its entered value conflicts with the puzzle.", cell(argument))
	})
	one("hint.value-correction.conclude", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("Clear the value at %s so the candidate grid can be checked next.", cell(argument))
	})
	one("hint.candidate-correction.observe", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("Look at %s. Its candidate state is inconsistent with the puzzle.", cell(argument))
	})
	constant("hint.candidate-correction.compare", "Discard this cell's deductions and rebuild them from the accepted row, column, and box values.")
	one("hint.candidate-correction.conclude", "cell", ArgumentCell, func(argument Argument) string {
		return fmt.Sprintf("Reset the candidate set at %s to every digit allowed by the accepted row, column, and box values.", cell(argument))
	})
	constant("hint.strategy.observe", "Examine the highlighted cells, candidates, and units.")
	constant("hint.strategy.compare", "Compare the highlighted constraints to identify the forced result.")
	one("hint.strategy.conclude", "strategy", ArgumentStrategy, func(argument Argument) string {
		return fmt.Sprintf("Apply the highlighted deduction. This pattern is called %s.", strategy(argument))
	})
	constant("hint.narration.apply", "Apply the deduction: ")
	constant("hint.narration.look-at", "Look at ")
	constant("hint.narration.value-conflict", ". Its entered value conflicts with the puzzle.")
	constant("hint.narration.clear-value", "Clear the value at ")
	constant("hint.narration.clear-value-tail", " so the candidate grid can be checked next.")
	constant("hint.narration.candidate-inconsistent", ". Its candidate state is inconsistent with the puzzle.")
	constant("hint.narration.reset-candidates", "Reset the candidate set at ")
	constant("hint.narration.reset-candidates-tail", " to every digit allowed by the accepted row, column, and box values.")
	constant("hint.narration.only-candidate", ". It has only one candidate: ")
	constant("hint.narration.period", ".")
	constant("hint.narration.no-other-digit", "Because no other digit can go in ")
	constant("hint.narration.place", ", place ")
	constant("hint.narration.naked-single-tail", " there. This is called a Naked Single.")
	constant("hint.narration.in-unit", "In ")
	constant("hint.narration.consider-digit", ", consider where ")
	constant("hint.narration.can-appear", " can appear.")
	constant("hint.narration.other-cells", "Every other empty cell in ")
	constant("hint.narration.cannot-contain", " cannot contain ")
	constant("hint.narration.leaving-only", ", leaving only ")
	constant("hint.narration.because-unit", "Because ")
	constant("hint.narration.still-needs", " still needs ")
	constant("hint.narration.only-cell", " and ")
	constant("hint.narration.hidden-single-tail", " is the only possible cell, place ")
	constant("hint.narration.hidden-single-name", " there. This is called a Hidden Single.")
	constant("hint.narration.place-result", "place ")
	constant("hint.narration.in", " in ")
	constant("hint.narration.remove", "remove candidate ")
	constant("hint.narration.from", " from ")
	constant("hint.narration.separator", ", ")
	one("hint.narration.strategy-result", "strategy", ArgumentStrategy, func(argument Argument) string {
		return fmt.Sprintf(". This is the %s result.", strategy(argument))
	})
}

// RenderEnglishName renders a stable difficulty or strategy identifier.
func RenderEnglishName(catalog *Catalog, domain, id string) (Descriptor, error) {
	if domain != "difficulty" && domain != "strategy" {
		return Descriptor{}, fmt.Errorf("unsupported name domain %q", domain)
	}
	return catalog.Render(domain + "." + id + ".name")
}
