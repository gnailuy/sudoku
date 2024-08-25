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
