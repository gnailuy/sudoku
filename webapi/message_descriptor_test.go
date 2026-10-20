package webapi

import (
	"encoding/json"
	"testing"

	"github.com/gnailuy/sudoku/game"
	usermessage "github.com/gnailuy/sudoku/message"
)

func TestAPIHintPlanUsesStableNameDescriptors(t *testing.T) {
	name := usermessage.Descriptor{ID: "strategy.naked-single.name", Arguments: []usermessage.Argument{}, Text: "Naked Single"}
	gradeName := usermessage.Descriptor{ID: "difficulty.easy.name", Arguments: []usermessage.Argument{}, Text: "Easy"}
	plan := apiHintPlan(&game.HintPlan{
		PlanID:  "plan",
		Kind:    game.HintKindStrategy,
		Title:   name,
		Summary: usermessage.Descriptor{ID: "test.summary", Arguments: []usermessage.Argument{}, Text: "summary"},
		Strategy: game.HintStrategy{
			ID:        "naked-single",
			Name:      name,
			Grade:     "easy",
			GradeName: gradeName,
		},
	})
	if plan.Title.Id != "strategy.naked-single.name" || plan.Title.Text != "Naked Single" || len(plan.Title.Arguments) != 0 {
		t.Fatalf("title = %#v", plan.Title)
	}
	if plan.Strategy == nil || plan.Strategy.Name.Id != plan.Title.Id || plan.Strategy.Name.Text != plan.Title.Text || len(plan.Strategy.Name.Arguments) != 0 || plan.Strategy.Grade != Easy || plan.Strategy.GradeName.Id != "difficulty.easy.name" || plan.Strategy.GradeName.Text != "Easy" {
		t.Fatalf("strategy = %#v", plan.Strategy)
	}
}

func TestAPIHintPlanUsesCorrectionTitleDescriptors(t *testing.T) {
	cases := []struct {
		kind game.HintKind
		id   string
		text string
	}{
		{game.HintKindValueCorrection, "hint.value-correction.title", "Fix this value"},
		{game.HintKindCandidateCorrection, "hint.candidate-correction.title", "Fix this candidate cell"},
	}
	for _, testCase := range cases {
		plan := apiHintPlan(&game.HintPlan{
			PlanID:  "plan",
			Kind:    testCase.kind,
			Title:   usermessage.Descriptor{ID: testCase.id, Arguments: []usermessage.Argument{}, Text: testCase.text},
			Summary: usermessage.Descriptor{ID: "test.summary", Arguments: []usermessage.Argument{}, Text: "summary"},
		})
		if plan.Title.Id != testCase.id || plan.Title.Text != testCase.text || len(plan.Title.Arguments) != 0 || plan.Strategy != nil {
			t.Fatalf("%s title/strategy = %#v/%#v", testCase.kind, plan.Title, plan.Strategy)
		}
	}
}

func TestAPIErrorsUseCanonicalDescriptors(t *testing.T) {
	codes := []ErrorCode{
		ErrorCodeInvalidRequest, ErrorCodeInvalidJson, ErrorCodeUnsupportedMediaType, ErrorCodePayloadTooLarge,
		ErrorCodeUnauthorized, ErrorCodeForbiddenOrigin, ErrorCodeSessionNotFound, ErrorCodeRevisionConflict,
		ErrorCodeInvalidSession, ErrorCodeInvalidAction, ErrorCodeInvalidCell, ErrorCodeImmutableCell,
		ErrorCodeNoteNotAllowed, ErrorCodeNoUndo, ErrorCodeNoRedo, ErrorCodeNoHint, ErrorCodeStaleHint,
		ErrorCodePersistenceFailed, ErrorCodeInternalError, ErrorCodeAccountUnavailable, ErrorCodeAccountUnauthorized,
		ErrorCodeCsrfFailed, ErrorCodeAccountGameNotFound, ErrorCodeGuestRevisionConflict, ErrorCodeGuestClaimConflict,
	}
	for _, code := range codes {
		detail := apiError(code).Error
		if detail.Message.Id != "error."+string(code) || detail.Message.Text == "" || len(detail.Message.Arguments) != 0 {
			t.Fatalf("error %q = %#v", code, detail.Message)
		}
	}
}

func TestAPIMessageDescriptorPreservesClosedArgumentKinds(t *testing.T) {
	text, integer, digit, row, column, box, difficulty, strategy, count := "safe", 42, 9, 1, 2, 3, "easy", "naked-single", 4
	descriptor := usermessage.Descriptor{ID: "test.arguments", Text: "fallback", Arguments: []usermessage.Argument{
		{Name: "text", Kind: usermessage.ArgumentText, Text: &text},
		{Name: "integer", Kind: usermessage.ArgumentInteger, Integer: &integer},
		{Name: "digit", Kind: usermessage.ArgumentDigit, Digit: &digit},
		{Name: "cell", Kind: usermessage.ArgumentCell, Cell: &usermessage.Cell{Row: 4, Column: 5}},
		{Name: "candidate", Kind: usermessage.ArgumentCandidate, Candidate: &usermessage.Candidate{Cell: usermessage.Cell{Row: 6, Column: 7}, Digit: 8}},
		{Name: "row", Kind: usermessage.ArgumentRow, Row: &row},
		{Name: "column", Kind: usermessage.ArgumentColumn, Column: &column},
		{Name: "box", Kind: usermessage.ArgumentBox, Box: &box},
		{Name: "difficulty", Kind: usermessage.ArgumentDifficulty, Difficulty: &difficulty},
		{Name: "strategy", Kind: usermessage.ArgumentStrategy, Strategy: &strategy},
		{Name: "count", Kind: usermessage.ArgumentCount, Count: &count},
	}}
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(mustAPIMessageDescriptor(descriptor, nil))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Arguments []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Arguments) != len(descriptor.Arguments) {
		t.Fatalf("arguments = %d, want %d", len(decoded.Arguments), len(descriptor.Arguments))
	}
	for i, argument := range decoded.Arguments {
		if argument.Name != descriptor.Arguments[i].Name || argument.Kind != string(descriptor.Arguments[i].Kind) {
			t.Fatalf("argument %d = %#v", i, argument)
		}
	}
}
