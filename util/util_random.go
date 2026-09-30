package util

import "math/rand"

// Function to generate numbers from min to max, including min but excluding max, optionally in a random order.
func GenerateNumberArray(min, max int, randomly bool) []int {
	return GenerateNumberArrayWithRand(min, max, randomly, nil)
}

// GenerateNumberArrayWithRand uses source when non-nil and otherwise preserves
// the process-wide random behavior used by interactive generation.
func GenerateNumberArrayWithRand(min, max int, randomly bool, source *rand.Rand) []int {
	if min >= max {
		panic("Bug: Invalid range to generate number array: min >= max")
	}

	numbers := make([]int, max-min)
	for i := min; i < max; i++ {
		numbers[i-min] = i
	}

	if randomly {
		ShuffleArrayWithRand(numbers, source)
	}

	return numbers
}

// Function to shuffle a slice of arrays in place.
func ShuffleArray[T any](array []T) {
	ShuffleArrayWithRand(array, nil)
}

// ShuffleArrayWithRand uses source when non-nil and otherwise uses the
// process-wide random source.
func ShuffleArrayWithRand[T any](array []T, source *rand.Rand) {
	swap := func(i, j int) { array[i], array[j] = array[j], array[i] }
	if source != nil {
		source.Shuffle(len(array), swap)
		return
	}
	rand.Shuffle(len(array), swap)
}

// Function to generate a random number from min to max, including min but excluding max.
func RandomInt(min, max int) int {
	if min >= max {
		panic("Bug: Invalid range to generate random number: min >= max")
	}

	return rand.Intn(max-min) + min
}

// Function to return true with a probability of p.
func RandomBool(p float64) bool {
	return RandomBoolWithRand(p, nil)
}

// RandomBoolWithRand uses source when non-nil and otherwise uses the
// process-wide random source.
func RandomBoolWithRand(p float64, source *rand.Rand) bool {
	if source != nil {
		return source.Float64() < p
	}
	return rand.Float64() < p
}
