package solver

import "github.com/gnailuy/sudoku/core"

// UnitKind identifies a Sudoku unit used by a strategy deduction.
type UnitKind string

const (
	UnitRow    UnitKind = "row"
	UnitColumn UnitKind = "column"
	UnitBox    UnitKind = "box"
)

// UnitRef identifies one zero-based row, column, or box.
type UnitRef struct {
	Kind  UnitKind
	Index int
}

// CandidateGroup records the candidate facts established for one cell.
type CandidateGroup struct {
	Position core.Position
	Values   []int
}

// CandidateRef identifies one candidate within one cell.
type CandidateRef struct {
	Position core.Position
	Value    int
}

// Evidence is renderer-neutral strategy evidence for one deduction.
type Evidence struct {
	Unit         *UnitRef
	Premises     []CandidateGroup
	RuledOut     []CandidateRef
	Eliminations []CandidateRef
}
