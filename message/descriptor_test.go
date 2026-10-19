package message

import (
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

	cases := []Descriptor{
		{ID: "Hint.Bad", Text: "x"},
		{ID: "hint.bad", Text: ""},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "digit", Kind: ArgumentDigit, Digit: ptr(0)}}},
		{ID: "hint.bad", Text: "x", Arguments: []Argument{{Name: "same", Kind: ArgumentDigit, Digit: ptr(1)}, {Name: "same", Kind: ArgumentDigit, Digit: ptr(2)}}},
	}
	for _, descriptor := range cases {
		if err := descriptor.Validate(); err == nil {
			t.Fatalf("expected invalid descriptor: %#v", descriptor)
		}
	}
}

func TestCatalogRendersValidatedFallback(t *testing.T) {
	catalog, err := NewCatalog(map[string]Definition{
		"hint.place": {Arguments: []ArgumentKind{ArgumentCell, ArgumentDigit}, Render: func(arguments []Argument) (string, error) {
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
	if _, err := catalog.Render("hint.place", Argument{Name: "digit", Kind: ArgumentDigit, Digit: ptr(9)}); err == nil {
		t.Fatal("expected contract mismatch")
	}
}
