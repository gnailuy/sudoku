package core

var threePermutations = [][3]int{
	{0, 1, 2}, {0, 2, 1}, {1, 0, 2},
	{1, 2, 0}, {2, 0, 1}, {2, 1, 0},
}

var canonicalAxisPermutations = sudokuAxisPermutations()

// CanonicalPuzzle returns the lexicographically smallest dot-notation puzzle
// under digit relabelling, band/stack permutation, row/column permutation
// within a band/stack, and transposition. Callers must provide a validated
// 81-cell puzzle.
func CanonicalPuzzle(puzzle string) string {
	if len(puzzle) != 81 {
		return puzzle
	}

	best := ""
	for transpose := 0; transpose < 2; transpose++ {
		for _, columns := range canonicalAxisPermutations {
			rowMasks := [9]uint16{}
			for row := 0; row < 9; row++ {
				for outputColumn, inputColumn := range columns {
					if canonicalCell(puzzle, row, inputColumn, transpose) != '.' {
						rowMasks[row] |= 1 << (8 - outputColumn)
					}
				}
			}

			for _, rows := range minimalRowOrders(rowMasks) {
				best = canonicalCandidate(puzzle, rows, columns, transpose, best)
			}
		}
	}
	return best
}

func canonicalCell(puzzle string, row, column, transpose int) byte {
	if transpose == 1 {
		row, column = column, row
	}
	value := puzzle[row*9+column]
	if value == '0' {
		return '.'
	}
	return value
}

func canonicalCandidate(puzzle string, rows, columns [9]int, transpose int, best string) string {
	var candidate [81]byte
	var digitMap [10]byte
	nextDigit := byte('1')
	position := 0
	relation := 0
	for _, row := range rows {
		for _, column := range columns {
			value := canonicalCell(puzzle, row, column, transpose)
			if value != '.' {
				index := value - '0'
				if digitMap[index] == 0 {
					digitMap[index] = nextDigit
					nextDigit++
				}
				value = digitMap[index]
			}
			candidate[position] = value
			if relation == 0 && best != "" {
				if value > best[position] {
					return best
				}
				if value < best[position] {
					relation = -1
				}
			}
			position++
		}
	}
	if best != "" && relation == 0 {
		return best
	}
	return string(candidate[:])
}

func sudokuAxisPermutations() [][9]int {
	result := make([][9]int, 0, 1296)
	for _, groups := range threePermutations {
		for _, first := range threePermutations {
			for _, second := range threePermutations {
				for _, third := range threePermutations {
					within := [3][3]int{first, second, third}
					var order [9]int
					position := 0
					for _, group := range groups {
						for _, item := range within[group] {
							order[position] = group*3 + item
							position++
						}
					}
					result = append(result, order)
				}
			}
		}
	}
	return result
}

func minimalRowOrders(masks [9]uint16) [][9]int {
	bandRows := [3][][3]int{}
	bandKeys := [3][3]uint16{}
	for band := 0; band < 3; band++ {
		for _, permutation := range threePermutations {
			order := [3]int{band*3 + permutation[0], band*3 + permutation[1], band*3 + permutation[2]}
			key := [3]uint16{masks[order[0]], masks[order[1]], masks[order[2]]}
			if len(bandRows[band]) == 0 || lessMaskTriple(key, bandKeys[band]) {
				bandKeys[band] = key
				bandRows[band] = [][3]int{order}
			} else if key == bandKeys[band] {
				bandRows[band] = append(bandRows[band], order)
			}
		}
	}

	var result [][9]int
	var bestBandKey [9]uint16
	for _, bandPermutation := range threePermutations {
		key := [9]uint16{}
		for outputBand, inputBand := range bandPermutation {
			copy(key[outputBand*3:], bandKeys[inputBand][:])
		}
		if len(result) == 0 || lessMaskNine(key, bestBandKey) {
			bestBandKey = key
			result = nil
		} else if key != bestBandKey {
			continue
		}

		firstBand := bandRows[bandPermutation[0]]
		secondBand := bandRows[bandPermutation[1]]
		thirdBand := bandRows[bandPermutation[2]]
		for _, first := range firstBand {
			for _, second := range secondBand {
				for _, third := range thirdBand {
					result = append(result, [9]int{
						first[0], first[1], first[2],
						second[0], second[1], second[2],
						third[0], third[1], third[2],
					})
				}
			}
		}
	}
	return result
}

func lessMaskTriple(left, right [3]uint16) bool {
	for index := range left {
		if left[index] != right[index] {
			return left[index] < right[index]
		}
	}
	return false
}

func lessMaskNine(left, right [9]uint16) bool {
	for index := range left {
		if left[index] != right[index] {
			return left[index] < right[index]
		}
	}
	return false
}
