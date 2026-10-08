package evilcohort

import "testing"

func TestFrozenEligibilityUsesEveryGate(t *testing.T) {
	base := Evidence{OriginalRating: 7.0, AdvancedMoveDensity: 1.0 / 6.0, EvilMoveCount: 2, AdvancedTechniqueDiversity: 3}
	if !eligible(base) {
		t.Fatal("boundary evidence should be eligible")
	}
	cases := []Evidence{
		{OriginalRating: 6.9, AdvancedMoveDensity: base.AdvancedMoveDensity, EvilMoveCount: 2, AdvancedTechniqueDiversity: 3},
		{OriginalRating: 7.0, AdvancedMoveDensity: 0.16, EvilMoveCount: 2, AdvancedTechniqueDiversity: 3},
		{OriginalRating: 7.0, AdvancedMoveDensity: base.AdvancedMoveDensity, EvilMoveCount: 1, AdvancedTechniqueDiversity: 3},
		{OriginalRating: 7.0, AdvancedMoveDensity: base.AdvancedMoveDensity, EvilMoveCount: 2, AdvancedTechniqueDiversity: 2},
	}
	for i, item := range cases {
		if eligible(item) {
			t.Fatalf("case %d should be ineligible", i)
		}
	}
}

func TestFrozenPartitionUsesFirst64HashBits(t *testing.T) {
	held, err := heldOut("0000000000000005ffffffffffffffffffffffffffffffffffffffffffffffff")
	if err != nil || !held {
		t.Fatalf("heldOut = %v, %v", held, err)
	}
	held, err = heldOut("0000000000000006ffffffffffffffffffffffffffffffffffffffffffffffff")
	if err != nil || held {
		t.Fatalf("heldOut = %v, %v", held, err)
	}
}
