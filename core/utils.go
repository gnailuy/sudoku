package core

import "math/rand"

// Function to generate numbers 1 to 9, optionally in a random order
func generateCellCandidates(randomly bool) []int {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	if randomly {
		shuffleArray(numbers)
	}

	return numbers
}

// Function to shuffle a slice of arrays in place
func shuffleArray[T any](array []T) {
	rand.Shuffle(len(array), func(i, j int) {
		array[i], array[j] = array[j], array[i]
	})
}
