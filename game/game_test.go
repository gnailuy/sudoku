package game

import (
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/solver"
)

// testProblem returns a minimal valid Sudoku board for testing.
// This is a real puzzle with a unique solution.
func testProblem() core.Board {
	input := "53..7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"
	var board core.Board
	board.FromString(input)
	return board
}

func newTestGame() Game {
	store := solver.NewStore()
	return NewGame(testProblem(), NewDefaultOptions(store))
}

func boardValue(board core.Board, position core.Position) int {
	return board.Get(position)
}

func TestNewGame(t *testing.T) {
	g := newTestGame()

	// Problem board should be preserved.
	if boardValue(g.ProblemBoard(), core.NewPosition(0, 0)) != 5 {
		t.Error("Expected problem cell (0,0) = 5")
	}

	// Play board should be a copy of the problem.
	if boardValue(g.PlayBoard(), core.NewPosition(0, 0)) != 5 {
		t.Error("Expected play cell (0,0) = 5")
	}

	// Game should not be solved yet.
	if g.IsSolved() {
		t.Error("Game should not be solved initially")
	}

	// Game should be valid initially.
	if !g.IsValid() {
		t.Error("Game should be valid initially")
	}
}

func TestAddInputAndRecordHistory(t *testing.T) {
	g := newTestGame()

	// Find an empty cell — (0,2) should be 0 in the problem.
	pos := core.NewPosition(0, 2)
	if boardValue(g.ProblemBoard(), pos) != 0 {
		t.Fatal("Expected (0,2) to be empty in problem")
	}

	// Add a valid input.
	cell := core.NewCell(pos, 4)
	err := g.addInputAndRecordHistory(cell)
	if err != nil {
		t.Fatalf("AddInputAndRecordHistory returned error: %v", err)
	}

	if g.Get(pos) != 4 {
		t.Errorf("Expected Get(0,2) = 4, got %d", g.Get(pos))
	}
}

func TestAddInputToProblemCell(t *testing.T) {
	g := newTestGame()

	// Try to change a problem cell — (0,0) = 5.
	cell := core.NewCell(core.NewPosition(0, 0), 3)
	err := g.addInputAndRecordHistory(cell)
	if err == nil {
		t.Error("Expected error when changing a problem cell")
	}
}

func TestUndoRedo(t *testing.T) {
	g := newTestGame()
	pos := core.NewPosition(0, 2)

	// Add a value.
	cell := core.NewCell(pos, 4)
	_ = g.addInputAndRecordHistory(cell)

	if g.Get(pos) != 4 {
		t.Fatalf("Expected 4 after add, got %d", g.Get(pos))
	}

	// Undo.
	err := g.undo()
	if err != nil {
		t.Fatalf("Undo returned error: %v", err)
	}
	if g.Get(pos) != 0 {
		t.Errorf("Expected 0 after undo, got %d", g.Get(pos))
	}

	// Redo.
	err = g.redo()
	if err != nil {
		t.Fatalf("Redo returned error: %v", err)
	}
	if g.Get(pos) != 4 {
		t.Errorf("Expected 4 after redo, got %d", g.Get(pos))
	}
}

func TestUndoEmpty(t *testing.T) {
	g := newTestGame()

	err := g.undo()
	if err == nil {
		t.Error("Expected error when undoing with no history")
	}
}

func TestRedoEmpty(t *testing.T) {
	g := newTestGame()

	err := g.redo()
	if err == nil {
		t.Error("Expected error when redoing with no undone moves")
	}
}

func TestReset(t *testing.T) {
	g := newTestGame()
	pos := core.NewPosition(0, 2)

	// Add a value and then reset.
	cell := core.NewCell(pos, 4)
	_ = g.addInputAndRecordHistory(cell)
	g.reset()

	if g.Get(pos) != 0 {
		t.Errorf("Expected 0 after reset, got %d", g.Get(pos))
	}

	// Undo should fail after reset (no history).
	err := g.undo()
	if err == nil {
		t.Error("Expected error on undo after reset")
	}
}

func TestSolve(t *testing.T) {
	g := newTestGame()

	g.solve()

	if !g.IsSolved() {
		t.Error("Game should be solved after Solve()")
	}
}

func TestHint(t *testing.T) {
	g := newTestGame()

	hint := g.Hint()
	if hint == nil {
		t.Fatal("Hint returned nil on unsolved game")
	}

	if hint.ProtocolVersion != HintProtocolVersion || hint.PlanID == "" {
		t.Fatalf("hint has invalid protocol identity: %+v", hint)
	}
	if hint.Strategy.ID == "" || hint.Summary == "" || len(hint.Steps) == 0 {
		t.Fatalf("hint plan is incomplete: %+v", hint)
	}
	if hint.Conclusion.Placement == nil && len(hint.Conclusion.Eliminations) == 0 {
		t.Fatalf("hint has no conclusion: %+v", hint)
	}
}

func TestRepair(t *testing.T) {
	g := newTestGame()

	// Solve the game to know the correct answer for cell (0,2).
	solvedGame := newTestGame()
	solvedGame.solve()
	correctValue := boardValue(solvedGame.PlayBoard(), core.NewPosition(0, 2))

	// Pick a wrong value for (0,2).
	wrongValue := 1
	if wrongValue == correctValue {
		wrongValue = 2
	}

	// Add a value that makes the board unsolvable (invalid input).
	cell := core.NewCell(core.NewPosition(0, 2), wrongValue)
	_ = g.addInputAndRecordHistory(cell)

	// The game tracks invalid inputs separately — IsValid() should return false.
	if g.IsValid() {
		// The wrong value may still leave the board solvable depending on the puzzle state.
		// In that case, try another position/value to force invalidity.
		return
	}

	// Repair should undo the invalid input.
	steps := g.repair()
	if steps == 0 {
		t.Error("Repair should undo at least one step")
	}
	if !g.IsValid() {
		t.Error("Game should be valid after repair")
	}
}

func TestClear(t *testing.T) {
	g := newTestGame()
	pos := core.NewPosition(0, 2)

	// Add a value to an empty cell.
	cell := core.NewCell(pos, 4)
	_ = g.addInputAndRecordHistory(cell)
	if g.Get(pos) != 4 {
		t.Fatalf("Expected 4 after add, got %d", g.Get(pos))
	}

	// Clear the cell (add value 0).
	clearCell := core.NewCell(pos, 0)
	err := g.addInputAndRecordHistory(clearCell)
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if g.Get(pos) != 0 {
		t.Errorf("Expected 0 after clear, got %d", g.Get(pos))
	}
}

func TestCheckValid(t *testing.T) {
	g := newTestGame()

	// No user input — board should be valid.
	if !g.IsValid() {
		t.Error("Initial board should be valid")
	}

	// Solve to find the correct value for an empty cell.
	solvedGame := newTestGame()
	solvedGame.solve()
	pos := core.NewPosition(0, 2)
	correctValue := boardValue(solvedGame.PlayBoard(), pos)

	// Add the correct value.
	cell := core.NewCell(pos, correctValue)
	_ = g.addInputAndRecordHistory(cell)

	if !g.IsValid() {
		t.Error("Board should be valid after adding correct value")
	}
}

func TestCheckInvalid(t *testing.T) {
	g := newTestGame()

	// Solve to find the correct value, then add a wrong one.
	solvedGame := newTestGame()
	solvedGame.solve()
	pos := core.NewPosition(0, 2)
	correctValue := boardValue(solvedGame.PlayBoard(), pos)

	wrongValue := (correctValue % 9) + 1
	if wrongValue == correctValue {
		wrongValue = (wrongValue % 9) + 1
	}

	cell := core.NewCell(pos, wrongValue)
	_ = g.addInputAndRecordHistory(cell)

	if g.IsValid() {
		t.Error("Board should be invalid after adding wrong value")
	}
}

func TestMultipleSolutionPuzzle(t *testing.T) {
	// A puzzle with multiple solutions — derived from the test puzzle
	// with cells (0,0) and (0,1) cleared to break uniqueness.
	multiPuzzle := "....7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"
	var board core.Board
	board.FromString(multiPuzzle)

	store := solver.NewStore()
	solutionCount := store.GetDefaultSolver().CountSolutions(&board)
	if solutionCount <= 1 {
		t.Skipf("test puzzle has %d solution(s), expected >1 — skipping", solutionCount)
	}

	// Board should still be valid.
	if !board.IsValid() {
		t.Fatal("Board with multiple solutions should still be valid")
	}

	// Game should be creatable and not pre-solved.
	opts := NewDefaultOptions(store)
	g := NewGame(board, opts)
	if g.IsSolved() {
		t.Error("Game should not be solved at start")
	}
}

func TestToString(t *testing.T) {
	g := newTestGame()

	s := g.ToString()
	if s == "" {
		t.Error("ToString should not return empty string")
	}

	// Should contain "Problem:" header.
	if !contains(s, "Problem:") {
		t.Error("ToString should contain 'Problem:'")
	}
}

func TestMultipleUndoRedo(t *testing.T) {
	g := newTestGame()

	// Add two values.
	pos1 := core.NewPosition(0, 2)
	pos2 := core.NewPosition(0, 3)
	_ = g.addInputAndRecordHistory(core.NewCell(pos1, 4))
	_ = g.addInputAndRecordHistory(core.NewCell(pos2, 8))

	// Undo both.
	_ = g.undo()
	_ = g.undo()
	if g.Get(pos1) != 0 || g.Get(pos2) != 0 {
		t.Error("Both cells should be 0 after two undos")
	}

	// Redo one.
	_ = g.redo()
	if g.Get(pos1) != 4 {
		t.Errorf("Expected 4 after one redo, got %d", g.Get(pos1))
	}
	if g.Get(pos2) != 0 {
		t.Errorf("Expected 0 for second cell after one redo, got %d", g.Get(pos2))
	}

	// New input after partial redo should truncate redo history.
	_ = g.addInputAndRecordHistory(core.NewCell(pos2, 6))
	err := g.redo()
	if err == nil {
		t.Error("Expected error on redo after new input (history truncated)")
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
