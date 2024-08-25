package util

import "math/rand"

// Function to generate numbers from min to max, including min but excluding max, optionally in a random order.
func GenerateNumberArray(min, max int, randomly bool) []int {
	if min >= max {
		panic("Bug: Invalid range to generate number array: min >= max")
	}

	numbers := make([]int, max-min)
	for i := min; i < max; i++ {
		numbers[i-min] = i
	}

	if randomly {
		ShuffleArray(numbers)
	}

	return numbers
}

// Function to shuffle a slice of arrays in place.
func ShuffleArray[T any](array []T) {
	rand.Shuffle(len(array), func(i, j int) {
		array[i], array[j] = array[j], array[i]
	})
}

// Function to return true with a probability of p.
func RandomBool(p float64) bool {
	return rand.Float64() < p
}
