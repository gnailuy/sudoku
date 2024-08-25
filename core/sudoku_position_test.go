package core

import (
	"fmt"
	"testing"
)

// Test the NewPosition function.
func TestNewPosition(t *testing.T) {
	tests := []struct {
		row    int
		column int
	}{
		{0, 0},
		{4, 4},
		{8, 8},
	}

	for _, test := range tests {
		position := NewPosition(test.row, test.column)

		if position.Row != test.row || position.Column != test.column {
			t.Errorf("Expected position (%d, %d), got (%d, %d)", test.row, test.column, position.Row, position.Column)
		}
	}
}

// Test the NewPosition function with invalid input.
func TestNewPositionInvalid(t *testing.T) {
	tests := []struct {
		row    int
		column int
	}{
		{-1, 0},
		{0, -1},
		{9, 0},
		{0, 9},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Invalid(%d,%d)", test.row, test.column), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected NewPosition to panic with invalid input")
				}
			}()

			NewPosition(test.row, test.column)
		})
	}
}

// Test the NewPositionFromInput function.
func TestNewPositionFromInput(t *testing.T) {
	tests := []struct {
		row    int
		column int
	}{
		{1, 1},
		{5, 5},
		{9, 9},
	}

	for _, test := range tests {
		position, err := NewPositionFromInput(test.row, test.column)

		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if position.Row != test.row-1 || position.Column != test.column-1 {
			t.Errorf("Expected position (%d, %d), got (%d, %d)", test.row-1, test.column-1, position.Row, position.Column)
		}
	}
}

// Test the NewPositionFromInput function with invalid input.
func TestNewPositionFromInputInvalid(t *testing.T) {
	tests := []struct {
		row    int
		column int
	}{
		{0, 1},
		{1, 0},
		{10, 1},
		{1, 10},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Invalid(%d,%d)", test.row, test.column), func(t *testing.T) {
			position, err := NewPositionFromInput(test.row, test.column)

			if err == nil {
				t.Errorf("Expected an error, got position: %s", position.ToString())
			}
		})
	}
}
