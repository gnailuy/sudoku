package core

import "testing"

const canonicalTestPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func TestCanonicalPuzzleCollapsesSudokuSymmetries(t *testing.T) {
	want := CanonicalPuzzle(canonicalTestPuzzle)
	transforms := []struct {
		name      string
		rows      [9]int
		columns   [9]int
		transpose bool
		digits    string
	}{
		{"digit relabel", identityAxis(), identityAxis(), false, "987654321"},
		{"rows within band", [9]int{2, 0, 1, 3, 4, 5, 6, 7, 8}, identityAxis(), false, "123456789"},
		{"bands", [9]int{6, 7, 8, 0, 1, 2, 3, 4, 5}, identityAxis(), false, "123456789"},
		{"columns within stack", identityAxis(), [9]int{1, 2, 0, 3, 4, 5, 6, 7, 8}, false, "123456789"},
		{"stacks", identityAxis(), [9]int{3, 4, 5, 6, 7, 8, 0, 1, 2}, false, "123456789"},
		{"transpose", identityAxis(), identityAxis(), true, "123456789"},
		{"combined", [9]int{8, 6, 7, 4, 5, 3, 1, 2, 0}, [9]int{5, 3, 4, 8, 6, 7, 2, 0, 1}, true, "314275968"},
	}

	for _, transform := range transforms {
		t.Run(transform.name, func(t *testing.T) {
			got := CanonicalPuzzle(transformPuzzle(canonicalTestPuzzle, transform.rows, transform.columns, transform.transpose, transform.digits))
			if got != want {
				t.Fatalf("canonical puzzle differs\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

func TestCanonicalPuzzleUsesDotNotationAndIsIdempotent(t *testing.T) {
	canonical := CanonicalPuzzle(canonicalTestPuzzle)
	if len(canonical) != 81 {
		t.Fatalf("length = %d, want 81", len(canonical))
	}
	if got := CanonicalPuzzle(canonical); got != canonical {
		t.Fatalf("canonicalization is not idempotent\n got: %s\nwant: %s", got, canonical)
	}
}

func identityAxis() [9]int { return [9]int{0, 1, 2, 3, 4, 5, 6, 7, 8} }

func transformPuzzle(puzzle string, rows, columns [9]int, transpose bool, digits string) string {
	result := make([]byte, 81)
	for outputRow, inputRow := range rows {
		for outputColumn, inputColumn := range columns {
			row, column := inputRow, inputColumn
			if transpose {
				row, column = column, row
			}
			value := puzzle[row*9+column]
			if value == '.' {
				result[outputRow*9+outputColumn] = '.'
			} else {
				result[outputRow*9+outputColumn] = digits[value-'1']
			}
		}
	}
	return string(result)
}
