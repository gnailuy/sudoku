package solver

// Define the solver store type containing the list of solvers.
type SudokuSolverStore map[string]ISudokuSolver

// Function to initialize the solver store.
func NewSudokuSolverStore() SudokuSolverStore {
	store := make(SudokuSolverStore)

	// Register the default solver.
	defaultSolver := NewDefaultSolver()
	store[defaultSolver.Properties.Key] = defaultSolver

	return store
}

// Function to get the solver by key from the store.
func (store SudokuSolverStore) GetSolverByKey(key string) ISudokuSolver {
	if solver, ok := store[key]; ok {
		return solver
	}

	return nil
}

// Function to get the default backtracking solver from the store.
func (store SudokuSolverStore) GetDefaultSolver() ISudokuSolver {
	defaultSolver := store.GetSolverByKey("default")

	if defaultSolver == nil {
		panic("Bug: Default solver not found in the store")
	}

	return defaultSolver
}
