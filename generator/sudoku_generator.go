package generator

import (
	"errors"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/util"
)

// Function to generate a solved Sudoku board by solving an empty board randomly.
func GenerateSolvedBoard(options SudokuGeneratorOptions) core.SudokuBoard {
	board := core.NewEmptySudokuBoard()

	// To generate a solved board from an empty board, we use the reliable default solver.
	solver := options.solverStore.GetDefaultSolver()
	solver.Solve(&board)

	return board
}

// Function to generate a Sudoku problem from a solved board.
func GenerateSudokuProblemFromSolvedBoard(board core.SudokuBoard, options SudokuGeneratorOptions) core.SudokuBoard {
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
	difficultyReached := false
	for i := 0; i < options.MaximumIterations; i++ {
		// Check if the difficulty level is reached.
		if options.Difficulty.IsWithinDifficultyLevel(board.FilledCells()) {
			difficultyReached = true
		}

		if difficultyReached {
			// Stop if removing more numbers will exceed the difficulty level.
			if !options.Difficulty.IsWithinDifficultyLevel(board.FilledCells() - 1) {
				break
			}

			// Use a simple geometric distribution to stop removing numbers with a probability of P.
			// The expected number of iterations after the difficulty level is reached will be 1/P.
			if util.RandomBool(0.2) {
				break
			}
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
			for _, key := range options.Difficulty.SolverKeys {
				solver := options.solverStore.GetSolverByKey(key)
				if solver == nil {
					panic("Bug: Invalid solver key: " + key)
				}

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
func GenerateSudokuProblem(options SudokuGeneratorOptions) core.SudokuBoard {
	solvedBoard := GenerateSolvedBoard(options)
	return GenerateSudokuProblemFromSolvedBoard(solvedBoard, options)
}

// Function to generate a Sudoku problem from an input string.
func GenerateSudokuProblemFromString(input string) (boardPointer *core.SudokuBoard, err error) {
	if !core.IsValidSudokuString(input) {
		return nil, errors.New("invalid Sudoku string: " + input)
	}

	board := core.NewEmptySudokuBoard()
	board.FromString(input)

	if !board.IsValid() {
		return nil, errors.New("invalid Sudoku board: " + input)
	}

	boardPointer = &board

	return
}
