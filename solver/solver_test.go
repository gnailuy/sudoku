package solver_test

import (
	"reflect"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// TestMoveString verifies the Move.String() output format.
func TestMoveString(t *testing.T) {
	move := solver.Move{
		Cell:      core.NewCell(core.NewPosition(2, 4), 7),
		Technique: "naked-single",
		Reason:    "only candidate in cell",
	}

	s := move.String()
	if s == "" {
		t.Fatal("Move.String() returned empty string")
	}
	// Should contain technique, position, value, and reason.
	if !containsAll(s, "naked-single", "(3, 5)", "7", "only candidate in cell") {
		t.Errorf("Move.String() = %q, expected to contain technique, position, value, reason", s)
	}
}

// TestBacktrackerImplementsCompleteSolver verifies the Backtracker satisfies CompleteSolver.
func TestBacktrackerImplementsCompleteSolver(t *testing.T) {
	var _ solver.CompleteSolver = solver.NewBacktracker()
}

// TestBacktrackerHintReturnsMove verifies Hint() returns a Move with technique info.
func TestBacktrackerHintReturnsMove(t *testing.T) {
	board := core.NewEmptyBoard()
	// Fill most of the board to get a deterministic hint.
	// Use a simple partial board: just leave one cell empty.
	bt := solver.NewBacktracker()
	bt.Solve(&board)

	// Now unset one cell to create a puzzle with one empty cell.
	pos := core.NewPosition(4, 4)
	board.Unset(pos)

	move := bt.Hint(&board)
	if move == nil {
		t.Fatal("Hint() returned nil for a solvable board with one empty cell")
	}

	if move.Technique != "backtracker" {
		t.Errorf("Expected technique 'backtracker', got %q", move.Technique)
	}

	if move.Cell.Position != pos {
		t.Errorf("Expected hint at %s, got %s", pos.ToString(), move.Cell.Position.ToString())
	}

	if move.Cell.Value < 1 || move.Cell.Value > 9 {
		t.Errorf("Expected hint value 1-9, got %d", move.Cell.Value)
	}

	if move.Reason == "" {
		t.Error("Expected non-empty reason in hint Move")
	}
}

// TestStoreGetDefaultSolver verifies the store returns a CompleteSolver.
func TestStoreGetDefaultSolver(t *testing.T) {
	store := solver.NewStore()
	defaultSolver := store.GetDefaultSolver()

	if defaultSolver == nil {
		t.Fatal("GetDefaultSolver() returned nil")
	}

	if defaultSolver.GetKey() != "default" {
		t.Errorf("Expected key 'default', got %q", defaultSolver.GetKey())
	}
}

// TestStoreGetStrategySolverByKeyReturnsNil verifies nil for unknown keys.
func TestStoreGetStrategySolverByKeyReturnsNil(t *testing.T) {
	store := solver.NewStore()

	s := store.GetStrategySolverByKey("nonexistent")
	if s != nil {
		t.Error("Expected nil for nonexistent strategy solver key")
	}
}

// TestStoreRegistersStrategySolvers verifies built-in strategy solvers are registered.
func TestStoreRegistersStrategySolvers(t *testing.T) {
	store := solver.NewStore()

	naked := store.GetStrategySolverByKey("naked-single")
	if naked == nil {
		t.Fatal("Expected naked-single solver to be registered")
	}
	if naked.GetKey() != "naked-single" {
		t.Errorf("Expected key 'naked-single', got %q", naked.GetKey())
	}

	hidden := store.GetStrategySolverByKey("hidden-single")
	if hidden == nil {
		t.Fatal("Expected hidden-single solver to be registered")
	}
	if hidden.GetKey() != "hidden-single" {
		t.Errorf("Expected key 'hidden-single', got %q", hidden.GetKey())
	}

	nakedPair := store.GetStrategySolverByKey("naked-pair")
	if nakedPair == nil {
		t.Fatal("Expected naked-pair solver to be registered")
	}
	if nakedPair.GetKey() != "naked-pair" {
		t.Errorf("Expected key 'naked-pair', got %q", nakedPair.GetKey())
	}

	pointing := store.GetStrategySolverByKey("pointing-pair")
	if pointing == nil {
		t.Fatal("Expected pointing-pair solver to be registered")
	}
	if pointing.GetKey() != "pointing-pair" {
		t.Errorf("Expected key 'pointing-pair', got %q", pointing.GetKey())
	}
}

// TestStoreGetAllStrategySolverKeys verifies all registered keys are returned.
func TestStoreGetAllStrategySolverKeys(t *testing.T) {
	store := solver.NewStore()
	keys := store.GetAllStrategySolverKeys()

	if len(keys) != 23 {
		t.Fatalf("Expected 23 strategy solver keys, got %d: %v", len(keys), keys)
	}

	expected := []string{
		"naked-single", "hidden-single",
		"naked-pair", "naked-triple", "pointing-pair", "hidden-pair",
		"x-wing", "xy-wing", "hidden-triple", "w-wing",
		"swordfish", "naked-quad", "simple-coloring", "hidden-quad", "xyz-wing",
		"jellyfish", "bug-plus-one", "unique-rectangle", "unique-rectangle-2",
		"unique-rectangle-3", "unique-rectangle-4", "x-cycles", "xy-chain",
	}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("strategy solver order = %v, want %v", keys, expected)
	}

	keys[0] = "mutated"
	if got := store.GetAllStrategySolverKeys()[0]; got != "naked-single" {
		t.Fatalf("returned key slice mutates store order: first key = %q", got)
	}
}

// TestBacktrackerSolveStillWorks verifies Solve() still works after the refactor.
func TestBacktrackerSolveStillWorks(t *testing.T) {
	board := core.NewEmptyBoard()
	bt := solver.NewBacktracker()

	result := bt.Solve(&board)
	if !result {
		t.Fatal("Backtracker.Solve() returned false for empty board")
	}

	if !board.IsSolved() {
		t.Fatal("Board is not solved after Backtracker.Solve()")
	}
}

// TestBacktrackerCountSolutionsStillWorks verifies CountSolutions() after refactor.
func TestBacktrackerCountSolutionsStillWorks(t *testing.T) {
	board := core.NewEmptyBoard()
	bt := solver.NewBacktracker()
	bt.Solve(&board)

	// Solved board has exactly 1 solution.
	count := bt.CountSolutions(&board)
	if count != 1 {
		t.Errorf("Expected 1 solution for solved board, got %d", count)
	}

	// Unset one cell — should still have 1 solution (unique).
	board.Unset(core.NewPosition(0, 0))
	count = bt.CountSolutions(&board)
	if count != 1 {
		t.Errorf("Expected 1 solution for board with one empty cell, got %d", count)
	}
}

func TestBacktrackerBoundedSolutionQueriesStopAtTheirCutoff(t *testing.T) {
	board := core.NewEmptyBoard()
	before := board.ToString()
	bt := solver.NewBacktracker()

	if got := bt.CountSolutionsUpTo(&board, 2); got != 2 {
		t.Fatalf("CountSolutionsUpTo(empty, 2) = %d, want 2", got)
	}
	if !bt.HasSolution(&board) {
		t.Fatal("empty board should have at least one solution")
	}
	if bt.HasUniqueSolution(&board) {
		t.Fatal("empty board should not have a unique solution")
	}
	if got := board.ToString(); got != before {
		t.Fatalf("bounded queries mutated board: got %q want %q", got, before)
	}
}

// TestBaseImplementsSolver verifies Base satisfies the Solver interface.
func TestBaseImplementsSolver(t *testing.T) {
	base := solver.Base{
		Key:         "test",
		DisplayName: "Test Solver",
		Description: "A test solver",
	}

	var _ solver.Solver = base

	if base.GetKey() != "test" {
		t.Errorf("Expected key 'test', got %q", base.GetKey())
	}
	if base.GetDisplayName() != "Test Solver" {
		t.Errorf("Expected name 'Test Solver', got %q", base.GetDisplayName())
	}
	if base.GetDescription() != "A test solver" {
		t.Errorf("Expected description 'A test solver', got %q", base.GetDescription())
	}
}

// containsAll checks if s contains all the given substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
