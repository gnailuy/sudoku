// Package message defines locale-neutral user-facing message descriptors.
package message

import (
	"fmt"
	"strings"
)

// ArgumentKind identifies the semantic type of a message argument.
type ArgumentKind string

const (
	ArgumentText       ArgumentKind = "text"
	ArgumentInteger    ArgumentKind = "integer"
	ArgumentDigit      ArgumentKind = "digit"
	ArgumentCell       ArgumentKind = "cell"
	ArgumentCandidate  ArgumentKind = "candidate"
	ArgumentRow        ArgumentKind = "row"
	ArgumentColumn     ArgumentKind = "column"
	ArgumentBox        ArgumentKind = "box"
	ArgumentDifficulty ArgumentKind = "difficulty"
	ArgumentStrategy   ArgumentKind = "strategy"
	ArgumentCount      ArgumentKind = "count"
	ArgumentDigits     ArgumentKind = "digits"
	ArgumentCells      ArgumentKind = "cells"
	ArgumentCandidates ArgumentKind = "candidates"
	ArgumentUnits      ArgumentKind = "units"
)

// Cell identifies a one-based Sudoku coordinate.
type Cell struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}

// Candidate identifies a digit at a one-based Sudoku coordinate.
type Candidate struct {
	Cell  Cell `json:"cell"`
	Digit int  `json:"digit"`
}

// Unit identifies one one-based Sudoku row, column, or box.
type Unit struct {
	Kind  string `json:"kind"`
	Index int    `json:"index"`
}

// Argument carries exactly one typed semantic value.
type Argument struct {
	Name       string       `json:"name"`
	Kind       ArgumentKind `json:"kind"`
	Text       *string      `json:"text,omitempty"`
	Integer    *int         `json:"integer,omitempty"`
	Digit      *int         `json:"digit,omitempty"`
	Cell       *Cell        `json:"cell,omitempty"`
	Candidate  *Candidate   `json:"candidate,omitempty"`
	Row        *int         `json:"row,omitempty"`
	Column     *int         `json:"column,omitempty"`
	Box        *int         `json:"box,omitempty"`
	Difficulty *string      `json:"difficulty,omitempty"`
	Strategy   *string      `json:"strategy,omitempty"`
	Count      *int         `json:"count,omitempty"`
	Digits     []int        `json:"digits,omitempty"`
	Cells      []Cell       `json:"cells,omitempty"`
	Candidates []Candidate  `json:"candidates,omitempty"`
	Units      []Unit       `json:"units,omitempty"`
}

// Descriptor is the locale-neutral contract plus deterministic English fallback.
type Descriptor struct {
	ID        string     `json:"id"`
	Arguments []Argument `json:"arguments"`
	Text      string     `json:"text"`
}

// Validate rejects malformed descriptors before they reach an API or terminal.
func (d Descriptor) Validate() error {
	if !validID(d.ID) {
		return fmt.Errorf("invalid message id %q", d.ID)
	}
	if d.Text == "" || len(d.Text) > 4096 {
		return fmt.Errorf("message %q has invalid fallback length", d.ID)
	}
	seen := make(map[string]struct{}, len(d.Arguments))
	for i, argument := range d.Arguments {
		if argument.Name == "" {
			return fmt.Errorf("message %q argument %d has no name", d.ID, i)
		}
		if _, exists := seen[argument.Name]; exists {
			return fmt.Errorf("message %q has duplicate argument %q", d.ID, argument.Name)
		}
		seen[argument.Name] = struct{}{}
		if err := argument.validate(); err != nil {
			return fmt.Errorf("message %q argument %q: %w", d.ID, argument.Name, err)
		}
	}
	return nil
}

func validID(id string) bool {
	if id == "" || strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") || strings.Contains(id, "..") {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' {
			return false
		}
	}
	return true
}

func (a Argument) validate() error {
	values := 0
	for _, present := range []bool{a.Text != nil, a.Integer != nil, a.Digit != nil, a.Cell != nil, a.Candidate != nil, a.Row != nil, a.Column != nil, a.Box != nil, a.Difficulty != nil, a.Strategy != nil, a.Count != nil, a.Digits != nil, a.Cells != nil, a.Candidates != nil, a.Units != nil} {
		if present {
			values++
		}
	}
	if values != 1 {
		return fmt.Errorf("must carry exactly one value")
	}
	validCoordinate := func(value int) bool { return value >= 1 && value <= 9 }
	switch a.Kind {
	case ArgumentText:
		if a.Text == nil || *a.Text == "" {
			return fmt.Errorf("must carry non-empty text")
		}
	case ArgumentInteger:
		if a.Integer == nil {
			return fmt.Errorf("must carry integer")
		}
	case ArgumentDigit:
		if a.Digit == nil || !validCoordinate(*a.Digit) {
			return fmt.Errorf("must carry digit 1 through 9")
		}
	case ArgumentCell:
		if a.Cell == nil || !validCoordinate(a.Cell.Row) || !validCoordinate(a.Cell.Column) {
			return fmt.Errorf("must carry one-based cell")
		}
	case ArgumentCandidate:
		if a.Candidate == nil || !validCoordinate(a.Candidate.Cell.Row) || !validCoordinate(a.Candidate.Cell.Column) || !validCoordinate(a.Candidate.Digit) {
			return fmt.Errorf("must carry one-based candidate")
		}
	case ArgumentRow:
		if a.Row == nil || !validCoordinate(*a.Row) {
			return fmt.Errorf("must carry row 1 through 9")
		}
	case ArgumentColumn:
		if a.Column == nil || !validCoordinate(*a.Column) {
			return fmt.Errorf("must carry column 1 through 9")
		}
	case ArgumentBox:
		if a.Box == nil || !validCoordinate(*a.Box) {
			return fmt.Errorf("must carry box 1 through 9")
		}
	case ArgumentDifficulty:
		if a.Difficulty == nil || *a.Difficulty == "" {
			return fmt.Errorf("must carry difficulty id")
		}
	case ArgumentStrategy:
		if a.Strategy == nil || *a.Strategy == "" {
			return fmt.Errorf("must carry strategy id")
		}
	case ArgumentCount:
		if a.Count == nil || *a.Count < 0 {
			return fmt.Errorf("must carry non-negative count")
		}
	case ArgumentDigits:
		if len(a.Digits) == 0 || len(a.Digits) > 32 {
			return fmt.Errorf("must carry 1 through 32 digits")
		}
		for _, value := range a.Digits {
			if !validCoordinate(value) {
				return fmt.Errorf("digits must be 1 through 9")
			}
		}
	case ArgumentCells:
		if len(a.Cells) == 0 || len(a.Cells) > 32 {
			return fmt.Errorf("must carry 1 through 32 cells")
		}
		for _, cell := range a.Cells {
			if !validCoordinate(cell.Row) || !validCoordinate(cell.Column) {
				return fmt.Errorf("cells must be one-based")
			}
		}
	case ArgumentCandidates:
		if len(a.Candidates) == 0 || len(a.Candidates) > 64 {
			return fmt.Errorf("must carry 1 through 64 candidates")
		}
		for _, candidate := range a.Candidates {
			if !validCoordinate(candidate.Cell.Row) || !validCoordinate(candidate.Cell.Column) || !validCoordinate(candidate.Digit) {
				return fmt.Errorf("candidates must contain one-based cells and digits")
			}
		}
	case ArgumentUnits:
		if len(a.Units) == 0 || len(a.Units) > 32 {
			return fmt.Errorf("must carry 1 through 32 units")
		}
		for _, unit := range a.Units {
			if (unit.Kind != "row" && unit.Kind != "column" && unit.Kind != "box") || !validCoordinate(unit.Index) {
				return fmt.Errorf("units must be one-based rows, columns, or boxes")
			}
		}
	default:
		return fmt.Errorf("unknown kind %q", a.Kind)
	}
	return nil
}
