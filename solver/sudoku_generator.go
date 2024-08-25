package solver

import (
	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/util"
)

// Define the options to generate a Sudoku problem.
type SudokuProblemOptions struct {
	MinimumFilledCells int
	MaximumIterations  int
	MaximumSolutions   int
	Solvers            []ISudokuSolver
}

// Constructor like function to create a default SudokuProblemOptions.
func NewDefaultSudokuProblemOptions() SudokuProblemOptions {
	return SudokuProblemOptions{
		MinimumFilledCells: 17,
		MaximumIterations:  60,
		MaximumSolutions:   1,
		Solvers:            []ISudokuSolver{NewDefaultSolver()},
	}
}

// Function to generate a solved Sudoku board by solving an empty board randomly.
func GenerateSolvedBoard() core.SudokuBoard {
	board := core.NewEmptySudokuBoard()

	// To generate a solved board from an empty board, we use the reliable default solver.
	solver := NewDefaultSolver()
	solver.Solve(&board)

	return board
}

// Function to generate a Sudoku problem from a solved board.
func GenerateSudokuProblemFromSolvedBoard(board core.SudokuBoard, options SudokuProblemOptions) core.SudokuBoard {
	if !board.IsSolved() || !board.IsValid() {
		panic("Bug: The board is not solved or not valid to generate a problem")
	}

	// Initially, all cells are filled.
	nonEmptyPositions := make([]core.Position, 0)
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			nonEmptyPositions = append(nonEmptyPositions, core.NewPosition(row, col))
		}
	}

	// Remove numbers randomly from the solved board to create a problem.
	for i := 0; i < options.MaximumIterations; i++ {
		// Stop removing numbers because the board has reached the minimum number of filled cells.
		if board.FilledCells() <= options.MinimumFilledCells {
			break
		}

		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells.
		if options.MaximumSolutions == 1 && board.FilledCells() <= 17 {
			break
		}

		// Test the non-empty positions in a random order and unset the first one that can be removed.
		util.ShuffleArray(nonEmptyPositions)

		removedPositionIndex := -1
		for j, position := range nonEmptyPositions {
			// Temporarily store the cell value.
			originalValue := board.Get(position)

			// Update the board.
			board.Unset(position)

			// Find out the maximum number of solutions using all available solvers.
			numberOfSolutions := 0
			for _, solver := range options.Solvers {
				nos := solver.CountSolutions(&board)
				if nos > numberOfSolutions {
					numberOfSolutions = nos
				}
			}

			// If the problem is solvable and has no more than maximum solutions, confirm the removal.
			if numberOfSolutions > 0 && numberOfSolutions <= options.MaximumSolutions {
				removedPositionIndex = j
				break
			}

			// If the problem is not solvable or has more than maximum solutions, revert the removal.
			board.Set(position, originalValue)
		}

		// Remove the position from the non-empty positions list.
		if removedPositionIndex >= 0 {
			nonEmptyPositions = append(nonEmptyPositions[:removedPositionIndex], nonEmptyPositions[removedPositionIndex+1:]...)
		} else {
			// We did not find any position to remove in this iteration, so we stop the process.
			break
		}
	}

	return board
}

// Function to generate a Sudoku problem.
func GenerateSudokuProblem(options SudokuProblemOptions) core.SudokuBoard {
	solvedBoard := GenerateSolvedBoard()
	return GenerateSudokuProblemFromSolvedBoard(solvedBoard, options)
}
