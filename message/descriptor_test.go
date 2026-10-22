package message

import (
	"encoding/json"
	"fmt"
	"testing"
)

func ptr[T any](value T) *T { return &value }

func TestDescriptorValidation(t *testing.T) {
	valid := Descriptor{ID: "hint.naked-single.conclusion", Text: "Place 9 in row 4, column 7.", Arguments: []Argument{
		{Name: "cell", Kind: ArgumentCell, Cell: &Cell{Row: 4, Column: 7}},
		{Name: "digit", Kind: ArgumentDigit, Digit: ptr(9)},
	}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid descriptor: %v", err)
	}
	validLists := Descriptor{ID: "hint.grouped", Text: "Grouped facts.", Arguments: []Argument{
		{Name: "digits", Kind: ArgumentDigits, Digits: []int{2, 7}},
		{Name: "cells", Kind: ArgumentCells, Cells: []Cell{{Row: 4, Column: 2}, {Row: 4, Column: 7}}},
		{Name: "candidates", Kind: ArgumentCandidates, Candidates: []Candidate{{Cell: Cell{Row: 4, Column: 9}, Digit: 2}}},
		{Name: "units", Kind: ArgumentUnits, Units: []Unit{{Kind: "row", Index: 4}}},
	}}
	if err := validLists.Validate(); err != nil {
		t.Fatalf("valid grouped descriptor: %v", err)
	}

	cases := []Descriptor{
		{ID: "Hint.Bad", Text: "x"},
		{ID: "hint.bad", Text: ""},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "digit", Kind: ArgumentDigit, Digit: ptr(0)}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "same", Kind: ArgumentDigit, Digit: ptr(1)}, {Name: "same", Kind: ArgumentDigit, Digit: ptr(2)}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "digits", Kind: ArgumentDigits, Digits: []int{}}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "cells", Kind: ArgumentCells, Cells: []Cell{{Row: 0, Column: 1}}}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "candidates", Kind: ArgumentCandidates, Candidates: []Candidate{{Cell: Cell{Row: 1, Column: 1}, Digit: 10}}}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "units", Kind: ArgumentUnits, Units: []Unit{{Kind: "diagonal", Index: 1}}}}},
	}
	for _, descriptor := range cases {
		if err := descriptor.Validate(); err == nil {
			t.Fatalf("expected invalid descriptor: %#v", descriptor)
		}
	}
}

func TestCatalogRendersValidatedFallback(t *testing.T) {
	catalog, err := NewCatalog(map[string]Definition{
		"hint.constant": {Arguments: []ArgumentSpec{}, Render: func([]Argument) (string, error) { return "Constant.", nil }},
		"hint.place": {Arguments: []ArgumentSpec{{Name: "cell", Kind: ArgumentCell}, {Name: "digit", Kind: ArgumentDigit}}, Render: func(arguments []Argument) (string, error) {
			return fmt.Sprintf("Place %d in row %d, column %d.", *arguments[1].Digit, arguments[0].Cell.Row, arguments[0].Cell.Column), nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := catalog.Render("hint.place",
		Argument{Name: "cell", Kind: ArgumentCell, Cell: &Cell{Row: 4, Column: 7}},
		Argument{Name: "digit", Kind: ArgumentDigit, Digit: ptr(9)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Text != "Place 9 in row 4, column 7." {
		t.Fatalf("unexpected text %q", descriptor.Text)
	}
	constant, err := catalog.Render("hint.constant")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(constant)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"id":"hint.constant","arguments":[],"text":"Constant."}` {
		t.Fatalf("constant descriptor must emit an empty argument array: %s", encoded)
	}
	if _, err := catalog.Render("hint.place", Argument{Name: "digit", Kind: ArgumentDigit, Digit: ptr(9)}); err == nil {
		t.Fatal("expected contract mismatch")
	}
	if _, err := catalog.Render("hint.place",
		Argument{Name: "position", Kind: ArgumentCell, Cell: &Cell{Row: 4, Column: 7}},
		Argument{Name: "digit", Kind: ArgumentDigit, Digit: ptr(9)},
	); err == nil {
		t.Fatal("expected argument name mismatch")
	}
}
