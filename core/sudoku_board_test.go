package core

import "testing"

// Test the SetCell and Unset function.
func TestSetAndUnset(t *testing.T) {
	board := NewEmptySudokuBoard()

	tests := []struct {
		cell Cell
	}{
		{NewCell(NewPosition(0, 0), 1)},
		{NewCell(NewPosition(4, 4), 5)},
		{NewCell(NewPosition(8, 8), 9)},
	}

	for _, test := range tests {
		board.SetCell(test.cell)

		if board.Get(test.cell.Position) != test.cell.Value {
			t.Errorf("Expected cell value at %s: %d, got %d", test.cell.Position.ToString(), test.cell.Value, board.Get(test.cell.Position))
		}

		if board.filledCellsCount != 1 {
			t.Errorf("Expected filled cells: 1, got %d", board.filledCellsCount)
		}

		board.Unset(test.cell.Position)

		if board.Get(test.cell.Position) != 0 {
			t.Errorf("Expected cell value at %s: 0, got %d", test.cell.Position.ToString(), board.Get(test.cell.Position))
		}

		if board.filledCellsCount != 0 {
			t.Errorf("Expected filled cells: 0, got %d", board.filledCellsCount)
		}
	}
}

// Test the GetRandomPositionWith function.
func TestGetRandomPositionWith(t *testing.T) {
	board := NewEmptySudokuBoard()

	tests := []struct {
		cell Cell
	}{
		{NewCell(NewPosition(0, 0), 1)},
		{NewCell(NewPosition(4, 4), 3)},
		{NewCell(NewPosition(8, 8), 5)},
	}

	for _, test := range tests {
		board.SetCell(test.cell)
	}

	position := board.GetRandomPositionWith(func(value int) bool {
		return value == 5
	})

	if position == nil {
		t.Errorf("Expected a random position, got nil")
	} else if board.Get(*position) != 5 {
		t.Errorf("Expected a position with value == 5, got %d", board.Get(*position))
	}

	position = board.GetRandomPositionWith(func(value int) bool {
		return value > 5
	})

	if position != nil {
		t.Errorf("Expected nil, got %s", position.ToString())
	}
}

// Test the Merge function.
func TestMerge(t *testing.T) {
	board1 := NewEmptySudokuBoard()
	board2 := NewEmptySudokuBoard()

	tests1 := []struct {
		cell Cell
	}{
		{NewCell(NewPosition(0, 0), 1)},
		{NewCell(NewPosition(4, 4), 3)},
		{NewCell(NewPosition(8, 8), 5)},
	}

	for _, test := range tests1 {
		board1.SetCell(test.cell)
	}

	tests2 := []struct {
		cell Cell
	}{
		{NewCell(NewPosition(1, 1), 2)}, // Will merge to board 1
		{NewCell(NewPosition(4, 4), 4)}, // Cannot merge to board 1
	}

	for _, test := range tests2 {
		board2.SetCell(test.cell)
	}

	board1.Merge(board2)

	expected := []struct {
		position Position
		value    int
	}{
		{NewPosition(0, 0), 1},
		{NewPosition(1, 1), 2}, // Merged from board 2
		{NewPosition(4, 4), 3}, // Did not change on merge
		{NewPosition(8, 8), 5},
	}

	for _, e := range expected {
		if board1.Get(e.position) != e.value {
			t.Errorf("Expected cell value at %s: %d, got %d", e.position.ToString(), e.value, board1.Get(e.position))
		}
	}

	if board1.filledCellsCount != 4 {
		t.Errorf("Expected filled cells: 4, got %d", board1.filledCellsCount)
	}
}
