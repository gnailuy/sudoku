package message

import (
	"fmt"
	"strings"
)

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
	list := func(items []string) string {
		switch len(items) {
		case 0:
			return ""
		case 1:
			return items[0]
		case 2:
			return items[0] + " and " + items[1]
		default:
			return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
		}
	}
	cells := func(argument Argument) string {
		items := make([]string, len(argument.Cells))
		for i, value := range argument.Cells {
			items[i] = fmt.Sprintf("row %d, column %d", value.Row, value.Column)
		}
		return list(items)
	}
	digits := func(argument Argument) string {
		items := make([]string, len(argument.Digits))
		for i, value := range argument.Digits {
			items[i] = fmt.Sprint(value)
		}
		return list(items)
	}
	units := func(argument Argument) string {
		items := make([]string, len(argument.Units))
		for i, value := range argument.Units {
			items[i] = fmt.Sprintf("%s %d", value.Kind, value.Index)
		}
		return list(items)
	}
	candidates := func(argument Argument) string {
		items := make([]string, len(argument.Candidates))
		for i, value := range argument.Candidates {
			items[i] = fmt.Sprintf("candidate %d at row %d, column %d", value.Digit, value.Cell.Row, value.Cell.Column)
		}
		return list(items)
	}
	many := func(id string, specs []ArgumentSpec, render func([]Argument) string) {
		definitions[id] = Definition{Arguments: specs, Render: func(arguments []Argument) (string, error) { return render(arguments), nil }}
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
	many("hint.naked-single.constraints", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "excluded_digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Start with %s. Its row, column, and box already contain %s.", cell(a[0]), digits(a[1]))
	})
	many("hint.naked-single.only", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Therefore, %d is the only digit that can go in %s.", *a[1].Digit, cell(a[0]))
	})
	many("hint.naked-single.apply", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. A cell with only one candidate is a %s.", *a[1].Digit, cell(a[0]), strategy(a[2]))
	})
	many("hint.hidden-single.inspect", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, check where %d can go.", units(a[0]), *a[1].Digit)
	})
	many("hint.hidden-single.blockers", []ArgumentSpec{{Name: "blockers", Kind: ArgumentCells}, {Name: "digit", Kind: ArgumentDigit}, {Name: "unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("The placed %ds at %s rule out %d from every other empty cell in %s.", *a[1].Digit, cells(a[0]), *a[1].Digit, units(a[2]))
	})
	many("hint.hidden-single.only", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Only %s can still contain %d.", cell(a[0]), *a[1].Digit)
	})
	many("hint.hidden-single.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. When a digit has only one possible cell in a row, column, or box, it is a %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.naked-pair.inspect", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Start in %s: %s can contain only %d or %d.", units(a[0]), cell(a[1]), *a[2].Digit, *a[3].Digit)
	})
	many("hint.naked-pair.match", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("The matching cell, %s, has the same two candidates: %d and %d.", cell(a[0]), *a[1].Digit, *a[2].Digit)
	})
	many("hint.naked-pair.reserve", []ArgumentSpec{{Name: "digits", Kind: ArgumentDigits}, {Name: "digit", Kind: ArgumentDigit}, {Name: "unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("These two cells must take %s in some order, so candidate %d cannot stay in any other cell in %s.", digits(a[0]), *a[1].Digit, units(a[2]))
	})
	many("hint.naked-pair.apply", []ArgumentSpec{{Name: "candidates", Kind: ArgumentCandidates}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove %s; this step does not place a value. This pattern is a %s.", candidates(a[0]), strategy(a[1]))
	})
	many("hint.naked-triple.first", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Start in %s: %s can contain only %d or %d.", units(a[0]), cell(a[1]), *a[2].Digit, *a[3].Digit)
	})
	many("hint.naked-triple.second", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Add %s; it can contain only %d or %d.", cell(a[0]), *a[1].Digit, *a[2].Digit)
	})
	many("hint.naked-triple.third", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}, {Name: "reserved_digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Add %s; it can contain only %d or %d. The three cells now contain only %s.", cell(a[0]), *a[1].Digit, *a[2].Digit, digits(a[3]))
	})
	many("hint.naked-triple.consequence", []ArgumentSpec{{Name: "reserved_digits", Kind: ArgumentDigits}, {Name: "removed_digits", Kind: ArgumentDigits}, {Name: "cell", Kind: ArgumentCell}, {Name: "remaining_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("These three cells must take %s in some order. That removes %s from %s, leaving only %d.", digits(a[0]), digits(a[1]), cell(a[2]), *a[3].Digit)
	})
	many("hint.naked-triple.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. Three cells whose candidates are limited to the same three digits form a %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})

	many("hint.subset-reservation.observe", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "cells", Kind: ArgumentCells}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, the cells at %s contain only %s.", units(a[0]), cells(a[1]), digits(a[2]))
	})
	many("hint.subset-reservation.explain", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "cells", Kind: ArgumentCells}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Those %d cells reserve %s in %s, so no other cell there can contain those digits.", len(a[1].Cells), digits(a[2]), units(a[0]))
	})
	many("hint.subset-reservation.apply", []ArgumentSpec{{Name: "candidates", Kind: ArgumentCandidates}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove %s. This pattern is called %s.", candidates(a[0]), strategy(a[1]))
	})
	many("hint.subset-reservation.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. This pattern is called %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.hidden-subset-confinement.observe", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "digits", Kind: ArgumentDigits}, {Name: "cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, the digits %s can appear only in %s.", units(a[0]), digits(a[1]), cells(a[2]))
	})
	many("hint.hidden-subset-confinement.explain", []ArgumentSpec{{Name: "digits", Kind: ArgumentDigits}, {Name: "unit", Kind: ArgumentUnits}, {Name: "cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("Because %s are confined to those %d cells in %s, those cells cannot contain any other candidates.", digits(a[0]), len(a[2].Cells), units(a[1]))
	})
	many("hint.hidden-subset-confinement.apply", []ArgumentSpec{{Name: "candidates", Kind: ArgumentCandidates}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove %s from the confined cells. This pattern is called %s.", candidates(a[0]), strategy(a[1]))
	})
	many("hint.hidden-subset-confinement.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s after removing the other candidates. This pattern is called %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.intersection-confinement.observe", []ArgumentSpec{{Name: "source_unit", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}, {Name: "intersection_unit", Kind: ArgumentUnits}, {Name: "cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, candidate %d is confined to %s at %s.", units(a[0]), *a[1].Digit, units(a[2]), cells(a[3]))
	})
	many("hint.intersection-confinement.explain", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "source_unit", Kind: ArgumentUnits}, {Name: "intersection_unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("Because every %d in %s lies in %s, cells outside the source but inside the intersection cannot contain it.", *a[0].Digit, units(a[1]), units(a[2]))
	})
	many("hint.intersection-confinement.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove candidate %d from %s. This pattern is called %s.", *a[0].Digit, cells(a[1]), strategy(a[2]))
	})
	many("hint.intersection-confinement.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s after applying the intersection. This pattern is called %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.fish-confinement.observe", []ArgumentSpec{{Name: "base_units", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cover_units", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, candidate %d appears only in %s.", units(a[0]), *a[1].Digit, units(a[2]))
	})
	many("hint.fish-confinement.explain", []ArgumentSpec{{Name: "base_units", Kind: ArgumentUnits}, {Name: "cover_units", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Whichever highlighted position contains %d in each of %s, %s must each receive that digit in one of those base units.", *a[2].Digit, units(a[0]), units(a[1]))
	})
	many("hint.fish-confinement.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove candidate %d from %s. This pattern is called %s.", *a[0].Digit, cells(a[1]), strategy(a[2]))
	})
	many("hint.pivot-wing.observe", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "pivot_digits", Kind: ArgumentDigits}, {Name: "wing_cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("At %s, candidates %s connect the wing cells at %s.", cell(a[0]), digits(a[1]), cells(a[2]))
	})
	many("hint.pivot-wing.explain", []ArgumentSpec{{Name: "shared_digit", Kind: ArgumentDigit}, {Name: "pivot", Kind: ArgumentCell}, {Name: "wing_cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("Each wing at %s shares one candidate with the pivot at %s, and both wings contain %d. Whichever pivot candidate is true, one wing must contain %d.", cells(a[2]), cell(a[1]), *a[0].Digit, *a[0].Digit)
	})
	many("hint.xy-wing.pivot", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("%s has only candidates %s. It must be one of them.", cell(a[0]), digits(a[1]))
	})
	many("hint.xy-wing.case", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "pivot_digit", Kind: ArgumentDigit}, {Name: "wing", Kind: ArgumentCell}, {Name: "shared_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("If %s is %d, then %s cannot be %d. Its only other candidate is %d, so %s must be %d.", cell(a[0]), *a[1].Digit, cell(a[2]), *a[1].Digit, *a[3].Digit, cell(a[2]), *a[3].Digit)
	})
	many("hint.xy-wing.guarantee", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "wing_cells", Kind: ArgumentCells}, {Name: "shared_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Whichever candidate is true at %s, at least one wing—%s—must be %d. We do not need to know which wing contains it.", cell(a[0]), cells(a[1]), *a[2].Digit)
	})
	many("hint.xy-wing.apply", []ArgumentSpec{{Name: "cells", Kind: ArgumentCells}, {Name: "wing_cells", Kind: ArgumentCells}, {Name: "shared_digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("%s see both wings at %s. Because at least one wing is %d, remove candidate %d from those cells. This pattern is called %s.", cells(a[0]), cells(a[1]), *a[2].Digit, *a[2].Digit, strategy(a[3]))
	})
	many("hint.xy-wing.place", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "wing_cells", Kind: ArgumentCells}, {Name: "shared_digit", Kind: ArgumentDigit}, {Name: "digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("%s sees both wings at %s. Because at least one wing is %d, %s cannot be %d. Its only other candidate is %d, so place %d there. This pattern is called %s.", cell(a[0]), cells(a[1]), *a[2].Digit, cell(a[0]), *a[2].Digit, *a[3].Digit, *a[3].Digit, strategy(a[4]))
	})

	many("hint.xyz-wing.pivot", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("%s has only candidates %s. It must be one of them.", cell(a[0]), digits(a[1]))
	})
	many("hint.xyz-wing.case", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "pivot_digit", Kind: ArgumentDigit}, {Name: "wing", Kind: ArgumentCell}, {Name: "unit", Kind: ArgumentUnits}, {Name: "shared_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("If %s is %d, then %s shares %s with it and cannot be %d. Its only other candidate is %d, so %s must be %d.", cell(a[0]), *a[1].Digit, cell(a[2]), units(a[3]), *a[1].Digit, *a[4].Digit, cell(a[2]), *a[4].Digit)
	})
	many("hint.xyz-wing.pivot-case", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "shared_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("If %s is %d, the pivot itself already contains %d.", cell(a[0]), *a[1].Digit, *a[1].Digit)
	})
	many("hint.xyz-wing.guarantee", []ArgumentSpec{{Name: "pivot", Kind: ArgumentCell}, {Name: "pattern_cells", Kind: ArgumentCells}, {Name: "shared_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Whichever candidate is true at %s, at least one of %s must be %d. We do not need to know which cell contains it.", cell(a[0]), cells(a[1]), *a[2].Digit)
	})
	many("hint.xyz-wing.apply", []ArgumentSpec{{Name: "target", Kind: ArgumentCell}, {Name: "first_pattern_cell", Kind: ArgumentCell}, {Name: "first_unit", Kind: ArgumentUnits}, {Name: "second_pattern_cell", Kind: ArgumentCell}, {Name: "second_unit", Kind: ArgumentUnits}, {Name: "third_pattern_cell", Kind: ArgumentCell}, {Name: "third_unit", Kind: ArgumentUnits}, {Name: "shared_digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("%s shares %s with %s, %s with %s, and %s with %s. Because at least one of those three cells is %d, %s cannot be %d. Remove candidate %d from it. This pattern is called %s.", cell(a[0]), units(a[2]), cell(a[1]), units(a[4]), cell(a[3]), units(a[6]), cell(a[5]), *a[7].Digit, cell(a[0]), *a[7].Digit, *a[7].Digit, strategy(a[8]))
	})

	many("hint.strong-link-wing.observe", []ArgumentSpec{{Name: "wing_cells", Kind: ArgumentCells}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("The cells at %s can each contain only %s.", cells(a[0]), digits(a[1]))
	})
	many("hint.strong-link-wing.bridge", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "link_digit", Kind: ArgumentDigit}, {Name: "link_cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, candidate %d can appear only at %s, so at least one of those cells must contain %d.", units(a[0]), *a[1].Digit, cells(a[2]), *a[1].Digit)
	})
	many("hint.strong-link-wing.force", []ArgumentSpec{{Name: "link_digit", Kind: ArgumentDigit}, {Name: "first_link_cell", Kind: ArgumentCell}, {Name: "first_wing_cell", Kind: ArgumentCell}, {Name: "second_link_cell", Kind: ArgumentCell}, {Name: "second_wing_cell", Kind: ArgumentCell}, {Name: "forced_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("If %s is %d, then %s cannot be %d and must be %d. If %s is %d, then %s must be %d. Therefore, at least one of those two cells is %d.", cell(a[1]), *a[0].Digit, cell(a[2]), *a[0].Digit, *a[5].Digit, cell(a[3]), *a[0].Digit, cell(a[4]), *a[5].Digit, *a[5].Digit)
	})
	many("hint.strong-link-wing.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "wing_cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("At least one of %s is %d. Remove candidate %d from %s because those cells see both of them. This pattern is called %s.", cells(a[2]), *a[0].Digit, *a[0].Digit, cells(a[1]), strategy(a[3]))
	})
	many("hint.strong-link-wing.place", []ArgumentSpec{{Name: "removed_digit", Kind: ArgumentDigit}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "wing_cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("At least one of %s is %d. Remove candidate %d from %s because it sees both of them, so place %d there. This pattern is called %s.", cells(a[3]), *a[0].Digit, *a[0].Digit, cell(a[2]), *a[1].Digit, strategy(a[4]))
	})
	many("hint.relational-wing.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove candidate %d from %s. This pattern is called %s.", *a[0].Digit, cells(a[1]), strategy(a[2]))
	})
	many("hint.relational-wing.place", []ArgumentSpec{{Name: "removed_digit", Kind: ArgumentDigit}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove candidate %d from %s, so place %d there. This pattern is called %s.", *a[0].Digit, cell(a[2]), *a[1].Digit, strategy(a[3]))
	})
	many("hint.bug-plus-one.observe", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Every other unsolved cell has two candidates, while %s has %s.", cell(a[0]), digits(a[1]))
	})
	many("hint.bug-plus-one.explain", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "units", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("Removing %d from %s would leave every unsolved cell bivalue and make the solution interchangeable across %s, so %d must be true.", *a[0].Digit, cell(a[1]), units(a[2]), *a[0].Digit)
	})
	many("hint.bug-plus-one.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. This uniqueness exception is called %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.unique-rectangle.observe", []ArgumentSpec{{Name: "cells", Kind: ArgumentCells}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("The cells at %s form a rectangle and all contain the pair %s.", cells(a[0]), digits(a[1]))
	})
	many("hint.unique-rectangle.type1", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Three corners contain exactly %s. If %s also used either digit, the rectangle would allow two interchangeable solutions.", digits(a[1]), cell(a[0]))
	})
	many("hint.unique-rectangle.type2", []ArgumentSpec{{Name: "cells", Kind: ArgumentCells}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("The two expanded corners at %s share extra candidate %d, so any cell that sees both corners cannot contain %d.", cells(a[0]), *a[1].Digit, *a[1].Digit)
	})
	many("hint.unique-rectangle.type3", []ArgumentSpec{{Name: "digits", Kind: ArgumentDigits}, {Name: "cells", Kind: ArgumentCells}, {Name: "unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("The rectangle's extra digits %s combine with %s as a reserved subset in %s, so those digits can be removed elsewhere in that unit.", digits(a[0]), cells(a[1]), units(a[2]))
	})
	many("hint.unique-rectangle.type4", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "unit", Kind: ArgumentUnits}, {Name: "removed_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Candidate %d appears in %s as a conjugate pair in %s, so candidate %d cannot occupy either expanded corner without creating interchangeable solutions.", *a[0].Digit, cells(a[1]), units(a[2]), *a[3].Digit)
	})
	many("hint.unique-rectangle.apply", []ArgumentSpec{{Name: "candidates", Kind: ArgumentCandidates}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove %s. This uniqueness relationship is called %s.", candidates(a[0]), strategy(a[1]))
	})
	many("hint.unique-rectangle.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s after applying the uniqueness relationship. This pattern is called %s.", *a[0].Digit, cell(a[1]), strategy(a[2]))
	})
	many("hint.coloring.pair", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, candidate %d can appear only in %s, so exactly one of those cells contains %d. Mark them as opposite possibilities A and B.", units(a[0]), *a[1].Digit, cells(a[2]), *a[1].Digit)
	})
	many("hint.coloring.chain", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "group_a", Kind: ArgumentCells}, {Name: "group_b", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("Follow each solid link. It joins the only two positions for %d in a row, column, or box, so the labels alternate through the chain: A at %s and B at %s.", *a[0].Digit, cells(a[1]), cells(a[2]))
	})
	one("hint.coloring.alternatives", "digit", ArgumentDigit, func(a Argument) string {
		return fmt.Sprintf("Either every A candidate contains %d and every B candidate does not, or every B candidate contains %d and every A candidate does not. We do not need to know which possibility is true.", *a.Digit, *a.Digit)
	})
	many("hint.coloring.elimination", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "group_a_cell", Kind: ArgumentCell}, {Name: "group_b_cell", Kind: ArgumentCell}}, func(a []Argument) string {
		return fmt.Sprintf("%s sees the A candidate at %s and the B candidate at %s. Whichever group contains %d, this cell sees a %d, so it cannot contain %d.", cell(a[1]), cell(a[2]), cell(a[3]), *a[0].Digit, *a[0].Digit, *a[0].Digit)
	})
	many("hint.coloring.apply", []ArgumentSpec{{Name: "candidates", Kind: ArgumentCandidates}, {Name: "colored_digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Remove %s. Alternating two possibilities through forced pairs and removing a candidate that sees both possibilities is called %s.", candidates(a[0]), strategy(a[2]))
	})
	many("hint.coloring.place", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "colored_digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("Place %d in %s. Alternating two possibilities through forced pairs and resolving the result is called %s.", *a[0].Digit, cell(a[1]), strategy(a[3]))
	})
	many("hint.x-cycle.opening", []ArgumentSpec{{Name: "unit", Kind: ArgumentUnits}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}}, func(a []Argument) string {
		return fmt.Sprintf("In %s, %d can go only in %s. One of these two cells must contain %d.", units(a[0]), *a[1].Digit, cells(a[2]), *a[1].Digit)
	})
	many("hint.x-cycle.assume", []ArgumentSpec{{Name: "start", Kind: ArgumentCell}, {Name: "forced", Kind: ArgumentCell}, {Name: "blocked", Kind: ArgumentCell}, {Name: "unit", Kind: ArgumentUnits}, {Name: "result", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Suppose %s were not %d. Then %s would have to be %d. That means %s cannot be %d, so %s forces %d into %s.", cell(a[0]), *a[5].Digit, cell(a[1]), *a[5].Digit, cell(a[2]), *a[5].Digit, units(a[3]), *a[5].Digit, cell(a[4]))
	})
	many("hint.x-cycle.follow", []ArgumentSpec{{Name: "true_cell", Kind: ArgumentCell}, {Name: "blocked", Kind: ArgumentCell}, {Name: "unit", Kind: ArgumentUnits}, {Name: "result", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Because %s is %d, %s cannot be %d. %s therefore forces %d into %s.", cell(a[0]), *a[4].Digit, cell(a[1]), *a[4].Digit, units(a[2]), *a[4].Digit, cell(a[3]))
	})
	many("hint.x-cycle.close", []ArgumentSpec{{Name: "true_cell", Kind: ArgumentCell}, {Name: "blocked", Kind: ArgumentCell}, {Name: "unit", Kind: ArgumentUnits}, {Name: "result", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Because %s is %d, %s cannot be %d. In %s, that leaves %s as the only place for %d.", cell(a[0]), *a[4].Digit, cell(a[1]), *a[4].Digit, units(a[2]), cell(a[3]), *a[4].Digit)
	})
	many("hint.x-cycle.place", []ArgumentSpec{{Name: "assumption", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("The starting assumption that %s was not %d leads back to %s being %d, so the assumption cannot be true. Place %d in %s. This pattern is called %s.", cell(a[0]), *a[1].Digit, cell(a[2]), *a[1].Digit, *a[1].Digit, cell(a[2]), strategy(a[3]))
	})
	many("hint.xy-chain.start", []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "first_digit", Kind: ArgumentDigit}, {Name: "second_digit", Kind: ArgumentDigit}, {Name: "endpoint_digit", Kind: ArgumentDigit}, {Name: "other_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("%s has only candidates %d and %d. Either it is %d, or—if it is not %d—it must be %d.", cell(a[0]), *a[1].Digit, *a[2].Digit, *a[3].Digit, *a[3].Digit, *a[4].Digit)
	})
	many("hint.xy-chain.assume", []ArgumentSpec{{Name: "start", Kind: ArgumentCell}, {Name: "endpoint_digit", Kind: ArgumentDigit}, {Name: "start_value", Kind: ArgumentDigit}, {Name: "middle", Kind: ArgumentCell}, {Name: "middle_value", Kind: ArgumentDigit}, {Name: "result", Kind: ArgumentCell}, {Name: "result_value", Kind: ArgumentDigit}, {Name: "first_unit", Kind: ArgumentUnits}, {Name: "second_unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("Suppose %s is not %d. It must then be %d. Because %s and %s share %s, %s cannot be %d and must be %d. That makes %s unable to be %d in %s, so it must be %d.", cell(a[0]), *a[1].Digit, *a[2].Digit, cell(a[0]), cell(a[3]), units(a[7]), cell(a[3]), *a[2].Digit, *a[4].Digit, cell(a[5]), *a[4].Digit, units(a[8]), *a[6].Digit)
	})
	many("hint.xy-chain.follow", []ArgumentSpec{{Name: "start", Kind: ArgumentCell}, {Name: "start_value", Kind: ArgumentDigit}, {Name: "middle", Kind: ArgumentCell}, {Name: "middle_value", Kind: ArgumentDigit}, {Name: "result", Kind: ArgumentCell}, {Name: "result_value", Kind: ArgumentDigit}, {Name: "first_unit", Kind: ArgumentUnits}, {Name: "second_unit", Kind: ArgumentUnits}}, func(a []Argument) string {
		return fmt.Sprintf("Because %s is %d, %s in %s cannot be %d and must be %d. Because %s and %s share %s, %s cannot be %d and must be %d.", cell(a[0]), *a[1].Digit, cell(a[2]), units(a[6]), *a[1].Digit, *a[3].Digit, cell(a[2]), cell(a[4]), units(a[7]), cell(a[4]), *a[3].Digit, *a[5].Digit)
	})
	many("hint.xy-chain.reach", []ArgumentSpec{{Name: "start", Kind: ArgumentCell}, {Name: "start_value", Kind: ArgumentDigit}, {Name: "endpoint", Kind: ArgumentCell}, {Name: "shared_digit", Kind: ArgumentDigit}, {Name: "endpoint_digit", Kind: ArgumentDigit}}, func(a []Argument) string {
		return fmt.Sprintf("Because %s is %d, %s cannot be %d. Its only other candidate is %d, so it must be %d.", cell(a[0]), *a[1].Digit, cell(a[2]), *a[3].Digit, *a[4].Digit, *a[4].Digit)
	})
	many("hint.xy-chain.place", []ArgumentSpec{{Name: "first_endpoint", Kind: ArgumentCell}, {Name: "second_endpoint", Kind: ArgumentCell}, {Name: "endpoint_digit", Kind: ArgumentDigit}, {Name: "cell", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("If %s is %d, the first endpoint contains %d. If it is not %d, the chain forces %d into the other endpoint, %s. Therefore at least one endpoint is %d. %s sees both endpoints, so it cannot be %d. Place %d in %s. This pattern is called %s.", cell(a[0]), *a[2].Digit, *a[2].Digit, *a[2].Digit, *a[2].Digit, cell(a[1]), *a[2].Digit, cell(a[3]), *a[2].Digit, *a[4].Digit, cell(a[3]), strategy(a[5]))
	})
	many("hint.implication-chain.observe", []ArgumentSpec{{Name: "cell", Kind: ArgumentCells}, {Name: "digits", Kind: ArgumentDigits}}, func(a []Argument) string {
		return fmt.Sprintf("Start at %s, which has only candidates %s.", cells(a[0]), digits(a[1]))
	})
	many("hint.implication-chain.explain", []ArgumentSpec{{Name: "from", Kind: ArgumentCandidate}, {Name: "to", Kind: ArgumentCandidate}}, func(a []Argument) string {
		return fmt.Sprintf("The bivalue cells at row %d, column %d and row %d, column %d share candidate %d and see each other, so that candidate cannot be true in both and the implication continues.", a[0].Candidate.Cell.Row, a[0].Candidate.Cell.Column, a[1].Candidate.Cell.Row, a[1].Candidate.Cell.Column, a[0].Candidate.Digit)
	})
	many("hint.implication-chain.apply", []ArgumentSpec{{Name: "digit", Kind: ArgumentDigit}, {Name: "cells", Kind: ArgumentCells}, {Name: "strategy", Kind: ArgumentStrategy}}, func(a []Argument) string {
		return fmt.Sprintf("One endpoint must contain %d, so remove candidate %d from %s. This pattern is called %s.", *a[0].Digit, *a[0].Digit, cells(a[1]), strategy(a[2]))
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
