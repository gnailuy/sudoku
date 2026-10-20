package random

import "math/rand"

// Numbers returns the integers in [min, max), optionally shuffled.
func Numbers(min, max int, shuffled bool) []int {
	return NumbersWithSource(min, max, shuffled, nil)
}

// NumbersWithSource uses source when non-nil and otherwise preserves the
// process-wide pseudorandom behavior used by interactive generation.
func NumbersWithSource(min, max int, shuffled bool, source *rand.Rand) []int {
	if min >= max {
		panic("Bug: Invalid range to generate number array: min >= max")
	}

	numbers := make([]int, max-min)
	for i := min; i < max; i++ {
		numbers[i-min] = i
	}

	if shuffled {
		ShuffleWithSource(numbers, source)
	}

	return numbers
}

// Shuffle randomizes values in place.
func Shuffle[T any](values []T) {
	ShuffleWithSource(values, nil)
}

// ShuffleWithSource uses source when non-nil and otherwise uses the
// process-wide pseudorandom source.
func ShuffleWithSource[T any](values []T, source *rand.Rand) {
	swap := func(i, j int) { values[i], values[j] = values[j], values[i] }
	if source != nil {
		source.Shuffle(len(values), swap)
		return
	}
	rand.Shuffle(len(values), swap)
}

// Int returns one pseudorandom integer in [min, max).
func Int(min, max int) int {
	if min >= max {
		panic("Bug: Invalid range to generate random number: min >= max")
	}

	return rand.Intn(max-min) + min
}

// Chance returns true with probability p.
func Chance(p float64) bool {
	return ChanceWithSource(p, nil)
}

// ChanceWithSource uses source when non-nil and otherwise uses the
// process-wide pseudorandom source.
func ChanceWithSource(p float64, source *rand.Rand) bool {
	if source != nil {
		return source.Float64() < p
	}
	return rand.Float64() < p
}
