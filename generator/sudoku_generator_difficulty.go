package generator

// Define the difficulty levels of a Sudoku problem.
type SudokuDifficulty struct {
	MinimumClues       int      // Inclusive.
	MaximumClues       int      // Exclusive.
	StrategySolverKeys []string // Allowed strategies to solve the problem in this difficulty level. Empty means all strategies are allowed.
}

// Constructor like function to create the easy difficulty level.
func NewEasySudokuDifficulty() SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       45,
		MaximumClues:       60,
		StrategySolverKeys: []string{},
	}
}

// Constructor like function to create the medium difficulty level.
func NewMediumSudokuDifficulty() SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       32,
		MaximumClues:       45,
		StrategySolverKeys: []string{},
	}
}

// Constructor like function to create the hard difficulty level.
func NewHardSudokuDifficulty() SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       25,
		MaximumClues:       32,
		StrategySolverKeys: []string{},
	}
}

// Constructor like function to create the extreme difficulty level.
func NewExtremeSudokuDifficulty() SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       20,
		MaximumClues:       25,
		StrategySolverKeys: []string{},
	}
}

// Constructor like function to create the evil difficulty level.
func NewEvilSudokuDifficulty() SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       17,
		MaximumClues:       20,
		StrategySolverKeys: []string{},
	}
}

// Constructor like function to create the custom difficulty level.
func NewCustomSudokuDifficulty(minimumClues int, maximumClues int, solverKeys []string) SudokuDifficulty {
	return SudokuDifficulty{
		MinimumClues:       minimumClues,
		MaximumClues:       maximumClues,
		StrategySolverKeys: solverKeys,
	}
}

// Function to check if the number of clues is within the difficulty level.
func (difficulty SudokuDifficulty) IsWithinDifficultyLevel(numberOfClues int) bool {
	return numberOfClues >= difficulty.MinimumClues && numberOfClues < difficulty.MaximumClues
}
