package cmd

import (
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/calibration"
)

const sudokuExchangePuzzle = "083020090000800100029300008000098700070000060006740000300006980002005000010030540"

func TestParseImportRecordRetainsSudokuExchangeRating(t *testing.T) {
	puzzle, sourceRef, rating, err := parseImportRecord("00015097c6c3 "+sudokuExchangePuzzle+" 7.2", "sudoku-exchange")
	if err != nil {
		t.Fatal(err)
	}
	if puzzle != sudokuExchangePuzzle || sourceRef != "sha1:00015097c6c3" || rating != 7.2 {
		t.Fatalf("record = (%q, %q, %v)", puzzle, sourceRef, rating)
	}
}

func TestParseImportRecordRejectsNonFiniteSudokuExchangeRating(t *testing.T) {
	for _, rating := range []string{"NaN", "+Inf", "-Inf"} {
		if _, _, _, err := parseImportRecord("00015097c6c3 "+sudokuExchangePuzzle+" "+rating, "sudoku-exchange"); err == nil || !strings.Contains(err.Error(), "rating") {
			t.Fatalf("rating %q error = %v, want invalid rating", rating, err)
		}
	}
}

func TestSudokuExchangeAnalysisManifestPreservesSourceIdentityAndRating(t *testing.T) {
	const fileHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	manifest := sudokuExchangeAnalysisManifest(fileHash, []importResult{
		{puzzle: sudokuExchangePuzzle, sourceRef: "sha1:00015097c6c3", originalRating: 7.2},
		{err: errTestInvalidRecord},
	})
	if manifest.Version != calibration.Version || manifest.Name != "sudoku-exchange/sha256:"+fileHash || len(manifest.Puzzles) != 1 {
		t.Fatalf("unexpected manifest header: %+v", manifest)
	}
	puzzle := manifest.Puzzles[0]
	if puzzle.ID != "sha1:00015097c6c3" || puzzle.SourceID != "sudoku-exchange:sha256:"+fileHash+"#sha1:00015097c6c3" {
		t.Fatalf("unexpected source identity: %+v", puzzle)
	}
	if puzzle.OriginalRating == nil || puzzle.OriginalRating.System != "Sukaku Explainer" || puzzle.OriginalRating.Label != "7.2" {
		t.Fatalf("unexpected original rating: %+v", puzzle.OriginalRating)
	}
	if puzzle.Split != "exploratory" || puzzle.License != "public-domain" || puzzle.Redistribution != "permitted" {
		t.Fatalf("unexpected provenance: %+v", puzzle)
	}
}

var errTestInvalidRecord = &testImportError{}

type testImportError struct{}

func (*testImportError) Error() string { return "invalid" }
