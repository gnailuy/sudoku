package random

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestNumbersWithSourceIsDeterministic(t *testing.T) {
	first := NumbersWithSource(1, 10, true, rand.New(rand.NewSource(42)))
	second := NumbersWithSource(1, 10, true, rand.New(rand.NewSource(42)))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("seeded orders differ: %v != %v", first, second)
	}
	seen := make(map[int]bool, len(first))
	for _, value := range first {
		seen[value] = true
	}
	for value := 1; value < 10; value++ {
		if !seen[value] {
			t.Fatalf("shuffled range omitted %d: %v", value, first)
		}
	}
}

func TestChanceWithSourceBoundaries(t *testing.T) {
	source := rand.New(rand.NewSource(42))
	if ChanceWithSource(0, source) {
		t.Fatal("zero probability returned true")
	}
	if !ChanceWithSource(1, source) {
		t.Fatal("unit probability returned false")
	}
}
