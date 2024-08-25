package core

import (
	"fmt"
	"testing"
)

// Test the NewCell function.
func TestNewCell(t *testing.T) {
	tests := []struct {
		position Position
		value    int
	}{
		{NewPosition(0, 0), 1},
		{NewPosition(4, 4), 5},
		{NewPosition(8, 8), 9},
	}

	for _, test := range tests {
		cell := NewCell(test.position, test.value)

		if cell.Position != test.position || cell.Value != test.value {
			t.Errorf("Expected cell %s: %d, got %s: %d", test.position.ToString(), test.value, cell.Position.ToString(), cell.Value)
		}
	}
}

// Test the NewCell function with invalid parameters.
func TestNewCellInvalidPosition(t *testing.T) {
	tests := []struct {
		position Position
		value    int
	}{
		{Position{Row: -1, Column: 0}, 1},
		{Position{Row: 0, Column: 9}, 5},
		{NewPosition(0, 0), 10},
		{NewPosition(8, 8), -1},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Invalid(%d,%d)%d", test.position.Row, test.position.Column, test.value), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for invalid input")
				}
			}()

			NewCell(test.position, test.value)
		})
	}
}

// Test the NewCellFromInput function.
func TestNewCellFromInput(t *testing.T) {
	tests := []struct {
		position Position
		value    int
	}{
		{Position{Row: 0, Column: 0}, 1},
		{Position{Row: 4, Column: 4}, 5},
		{Position{Row: 8, Column: 8}, 9},
	}

	for _, test := range tests {
		cell, err := NewCellFromInput(test.position, test.value)

		if err != nil {
			t.Errorf("Unexpected error: %s", err.Error())
		}

		if cell.Position != test.position || cell.Value != test.value {
			t.Errorf("Expected cell %s: %d, got %s: %d", test.position.ToString(), test.value, cell.Position.ToString(), cell.Value)
		}
	}
}

// Test the NewCellFromInput function with invalid parameters.
func TestNewCellFromInputInvalid(t *testing.T) {
	tests := []struct {
		position Position
		value    int
	}{
		{Position{Row: -1, Column: 0}, 1},
		{Position{Row: 0, Column: 9}, 5},
		{NewPosition(0, 0), 10},
		{NewPosition(8, 8), -1},
	}

	for _, test := range tests {
		cell, err := NewCellFromInput(test.position, test.value)

		if err == nil {
			t.Errorf("Expected error for invalid input")
		}

		if cell != nil {
			t.Errorf("Expected nil cell for invalid input")
		}
	}
}
