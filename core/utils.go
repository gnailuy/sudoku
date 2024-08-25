package core

import "math/rand"

// Function to generate numbers 1 to 9, optionally in a random order
func generateCellCandidates(randomly bool) []int {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	if randomly {
		rand.Shuffle(len(numbers), func(i, j int) {
			numbers[i], numbers[j] = numbers[j], numbers[i]
		})
	}

	return numbers
}

// Function to generate a random number between min and max
func generateRandomNumber(min, max int) int {
	return rand.Intn(max-min) + min
}
