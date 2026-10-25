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

// LinkKind describes the logical relationship between two candidate nodes.
type LinkKind string

const (
	// LinkStrong means exactly one endpoint must contain the candidate.
	LinkStrong LinkKind = "strong"
	// LinkWeak means the two endpoints cannot both contain the candidate.
	LinkWeak LinkKind = "weak"
	// LinkImplication connects consecutive bivalue cells in an implication chain.
	LinkImplication LinkKind = "implication"
)

// CandidateLink records one renderer-neutral relationship between candidates.
type CandidateLink struct {
	From CandidateRef
	To   CandidateRef
	Kind LinkKind
}

// CandidatePartition records one side of an alternating candidate grouping.
type CandidatePartition struct {
	Candidates []CandidateRef
}

// Evidence is renderer-neutral strategy evidence for one deduction.
type Evidence struct {
	// Unit is retained for common single-unit techniques. Units contains every
	// typed unit involved in the deduction and is the renderer-neutral form
	// consumed by generic teaching plans.
	Unit       *UnitRef
	Units      []UnitRef
	Premises   []CandidateGroup
	Chain      []CandidateGroup
	Links      []CandidateLink
	Partitions []CandidatePartition
	// Witnesses records placed values that directly establish a deduction.
	// It is currently used by Naked Single to retain one deterministic peer
	// for every excluded digit without pretending that filled cells still have
	// candidates.
	Witnesses    []core.Cell
	RuledOut     []CandidateRef
	Eliminations []CandidateRef
}
