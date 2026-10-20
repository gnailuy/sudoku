package generator

import (
	"fmt"

	"github.com/gnailuy/sudoku/solver"
)

// Difficulty defines the difficulty level of a Sudoku problem.
//
// SolverKeys lists the strategy solvers introduced at this tier.
// The full set of allowed solvers for a level equals SolverKeys plus
// all solvers from lower tiers, derived from tierRegistry/tierOrder.
// For example:
//   - Easy:   SolverKeys = [naked-single, hidden-single]
//   - Medium: SolverKeys = [naked-pair, naked-triple, pointing-pair, hidden-pair]
//     Allowed = Easy.SolverKeys + Medium.SolverKeys
//
// During generation, the generator uses AllowedSolverKeys (the
// cumulative set) to constrain cell removal, and verifies that
// lower-tier solvers alone cannot solve the puzzle — ensuring the
// puzzle genuinely requires at least one solver from this tier.
//
// Lower-tier solver keys are derived from tierRegistry/tierOrder —
// there is no separate field. The registry is the single source of
// truth for tier ordering.
type Difficulty struct {
	MinimumClues int      // Inclusive.
	MaximumClues int      // Exclusive.
	SolverKeys   []string // Solvers introduced at this tier. Empty means no technique constraint.
}

// tierOrder defines the sequence of difficulty tiers from lowest to
// highest. Used alongside tierRegistry to derive lower-tier solver keys.
var tierOrder = solver.StrategyTierNames()

// tierRegistry is a package-local detached view of the canonical solver tier
// hierarchy. Classification and generation therefore use the same ordering.
var tierRegistry = func() map[string][]string {
	registry := make(map[string][]string, len(tierOrder))
	for _, name := range tierOrder {
		registry[name] = solver.StrategySolverKeysForTier(name)
	}
	return registry
}()

// lowerTierSolverKeys returns the cumulative solver keys from all tiers
// below the tier that owns the given SolverKeys. Returns nil if the
// tier is the lowest or is not found in the registry.
func lowerTierSolverKeys(solverKeys []string) []string {
	if len(solverKeys) == 0 {
		return nil
	}

	for i, name := range tierOrder {
		tier := tierRegistry[name]
		if matchesTier(tier, solverKeys) {
			if i == 0 {
				return nil // lowest tier
			}
			var lower []string
			for _, n := range tierOrder[:i] {
				lower = append(lower, tierRegistry[n]...)
			}
			return lower
		}
	}

	return nil // not in registry (custom/unconstrained)
}

// matchesTier checks whether solverKeys matches the given tier entry.
func matchesTier(tier, solverKeys []string) bool {
	if len(tier) != len(solverKeys) {
		return false
	}
	for i := range tier {
		if tier[i] != solverKeys[i] {
			return false
		}
	}
	return true
}

// AllowedSolverKeys returns the full set of solvers the puzzle may use:
// this tier's SolverKeys plus all lower-tier solvers (derived from
// tierRegistry/tierOrder).
func (d Difficulty) AllowedSolverKeys() []string {
	lower := lowerTierSolverKeys(d.SolverKeys)
	all := make([]string, 0, len(lower)+len(d.SolverKeys))
	all = append(all, lower...)
	all = append(all, d.SolverKeys...)
	return all
}

// LowerTierSolverKeys returns the cumulative solver keys from all tiers
// below this difficulty's tier. Derived from tierRegistry/tierOrder.
func (d Difficulty) LowerTierSolverKeys() []string {
	return lowerTierSolverKeys(d.SolverKeys)
}

// NewEasyDifficulty creates the easy difficulty level.
// Easy puzzles are solvable using only naked singles and hidden singles.
func NewEasyDifficulty() Difficulty {
	return Difficulty{
		MinimumClues: solver.EasyMinClues,
		MaximumClues: solver.EasyMaxClues,
		SolverKeys:   tierRegistry["easy"],
	}
}

// NewMediumDifficulty creates the medium difficulty level.
// Medium puzzles require at least one intermediate technique (naked-pair,
// naked-triple, pointing-pair, or hidden-pair) — basic techniques alone
// won't suffice.
func NewMediumDifficulty() Difficulty {
	return Difficulty{
		MinimumClues: solver.MediumMinClues,
		MaximumClues: solver.MediumMaxClues,
		SolverKeys:   tierRegistry["medium"],
	}
}

// NewHardDifficulty creates the hard difficulty level.
// Hard puzzles require at least one hard technique (X-Wing, XY-Wing, or
// hidden-triple) — medium-tier techniques alone won't suffice.
func NewHardDifficulty() Difficulty {
	return Difficulty{
		MinimumClues: solver.HardMinClues,
		MaximumClues: solver.HardMaxClues,
		SolverKeys:   tierRegistry["hard"],
	}
}

// NewExpertDifficulty creates the expert difficulty level.
// Expert puzzles require at least one expert technique (swordfish,
// naked-quad, simple-coloring, or hidden-quad) — hard-tier techniques
// alone won't suffice.
func NewExpertDifficulty() Difficulty {
	return Difficulty{
		MinimumClues: solver.ExpertMinClues,
		MaximumClues: solver.ExpertMaxClues,
		SolverKeys:   tierRegistry["expert"],
	}
}

// NewEvilDifficulty creates the evil difficulty level.
// Evil puzzles require at least one evil technique (jellyfish) —
// expert-tier techniques alone won't suffice.
func NewEvilDifficulty() Difficulty {
	return Difficulty{
		MinimumClues: solver.EvilMinClues,
		MaximumClues: solver.EvilMaxClues,
		SolverKeys:   tierRegistry["evil"],
	}
}

// DifficultyForName returns the canonical configuration for a player-facing grade.
func DifficultyForName(name string) (Difficulty, error) {
	switch name {
	case "easy":
		return NewEasyDifficulty(), nil
	case "medium":
		return NewMediumDifficulty(), nil
	case "hard":
		return NewHardDifficulty(), nil
	case "expert":
		return NewExpertDifficulty(), nil
	case "evil":
		return NewEvilDifficulty(), nil
	default:
		return Difficulty{}, fmt.Errorf("invalid difficulty level: %s. Options: easy, medium, hard, expert, evil", name)
	}
}

// NewCustomDifficulty creates a custom difficulty level.
func NewCustomDifficulty(minimumClues int, maximumClues int, solverKeys []string) Difficulty {
	return Difficulty{
		MinimumClues: minimumClues,
		MaximumClues: maximumClues,
		SolverKeys:   solverKeys,
	}
}

// Function to check if the number of clues is within the difficulty level.
func (difficulty Difficulty) IsWithinDifficultyLevel(numberOfClues int) bool {
	return numberOfClues >= difficulty.MinimumClues && numberOfClues < difficulty.MaximumClues
}
