package generator

import (
	"errors"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/util"
)

// Function to generate a solved Sudoku board by solving an empty normalized board randomly.
func GenerateNormalizedSolvedBoard(options SudokuGeneratorOptions) core.SudokuBoard {
	// The first row of a normalize empty board is always from 1 to 9.
	board := core.NewEmptySudokuBoard()
	for col := 0; col < 9; col++ {
		board.Set(core.NewPosition(0, col), col+1)
	}

	// To generate a solved board from an empty normalized board, we use the reliable default solver.
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
	cluesNumberReached := false
	for i := 0; i < options.MaximumIterations; i++ {
		// Check if the number of clues reached the difficulty level.
		if options.Difficulty.IsWithinDifficultyLevel(board.GetFilledCellsCount()) {
			cluesNumberReached = true
		}

		if cluesNumberReached {
			// Stop if removing more numbers will exceed the difficulty level.
			if !options.Difficulty.IsWithinDifficultyLevel(board.GetFilledCellsCount() - 1) {
				break
			}

			// Use a simple geometric distribution to stop removing numbers with a probability of P.
			// The expected number of iterations after the difficulty level is reached will be 1/P.
			if util.RandomBool(0.125) {
				break
			}
		}

		// Stop removing numbers because it is impossible to have a unique solution with less than 17 filled cells.
		if options.MaximumSolutions == 1 && board.GetFilledCellsCount() <= 17 {
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

			// Find out the maximum number of solutions using the default solver.
			numberOfSolutions := options.solverStore.GetDefaultSolver().CountSolutions(&board)

			// Check if the problem is solvable and has no more than maximum solutions.
			if numberOfSolutions > 0 && numberOfSolutions <= options.MaximumSolutions {
				canHint := false

				if len(options.Difficulty.StrategySolverKeys) > 0 {
					// If there are strategy solvers configured, we limit the problem to be solvable with the specified strategies.
					// Test the strategy solvers to ensure that at least one of them can give a hint.
					for _, key := range options.Difficulty.StrategySolverKeys {
						solver := options.solverStore.GetSolverByKey(key)
						if solver == nil {
							panic("Bug: Invalid strategy solver key: " + key)
						}

						hint := solver.Hint(&board)
						if hint != nil {
							canHint = true
							break
						}
					}
				} else {
					// If there are no strategy solvers configured, we don't care about limiting the problem to specific strategies.
					// And the default solver can always give a hint.
					canHint = true
				}

				// Confirm the removal.
				if canHint {
					removedPositionIndex = j
					break
				}
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
	solvedBoard := GenerateNormalizedSolvedBoard(options)

	problem := GenerateSudokuProblemFromSolvedBoard(solvedBoard, options)
	problem.Randomize()

	return problem
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
