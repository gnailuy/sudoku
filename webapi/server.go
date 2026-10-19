package webapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	usermessage "github.com/gnailuy/sudoku/message"
)

var canonicalEnglishMessages = func() *usermessage.Catalog {
	catalog, err := usermessage.NewEnglishCatalog()
	if err != nil {
		panic(fmt.Sprintf("construct canonical English catalog: %v", err))
	}
	return catalog
}()

const MaxRequestBytes = 1 << 20

type Server struct {
	guest    *GuestRuntime
	accounts *AccountRuntime
}

func NewServer() *Server { return &Server{} }

func (*Server) GetHealth(context.Context, GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "healthy"}, nil
}

func NewHandler(server *Server, origins []string) http.Handler {
	strict := NewStrictHandlerWithOptions(server, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, 400, ErrorCodeInvalidJson, "invalid JSON request")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, 500, ErrorCodeInternalError, "internal server error")
		},
	})
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return middleware(Handler(strict), allowed)
}

func middleware(next http.Handler, origins map[string]struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := origins[origin]; !ok {
				writeError(w, 403, ErrorCodeForbiddenOrigin, "origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				if !validPreflight(r) {
					writeError(w, 403, ErrorCodeForbiddenOrigin, "preflight request is not allowed")
					return
				}
				w.WriteHeader(204)
				return
			}
		}
		if r.Method == http.MethodPost && postRequiresBody(r.URL.Path) {
			want := "application/json"
			got := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
			if got != want {
				writeError(w, 415, ErrorCodeUnsupportedMediaType, "unsupported content type")
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
			if err != nil {
				writeError(w, 400, ErrorCodeInvalidRequest, "unable to read request")
				return
			}
			if len(data) > MaxRequestBytes {
				writeError(w, 413, ErrorCodePayloadTooLarge, "request exceeds 1 MiB")
				return
			}
			if want == "application/json" {
				if err := validateJSONRequest(r.URL.Path, data); err != nil {
					writeError(w, 400, ErrorCodeInvalidJson, err.Error())
					return
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(data))
		}
		next.ServeHTTP(w, r)
	})
}

func postRequiresBody(path string) bool {
	switch path {
	case "/api/v1/guest/games",
		"/api/v1/guest/games/actions",
		"/api/v1/account/games",
		"/api/v1/account/games/claim":
		return true
	default:
		return strings.HasSuffix(path, "/actions")
	}
}

func validPreflight(r *http.Request) bool {
	switch r.Header.Get("Access-Control-Request-Method") {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}
		if !strings.EqualFold(header, "Authorization") && !strings.EqualFold(header, "Content-Type") {
			return false
		}
	}
	return true
}

func translateAction(body ActionRequest) (game.Action, int64, error) {
	kind, err := body.Discriminator()
	if err != nil {
		return nil, 0, err
	}
	position := func(row, column int) (core.Position, error) {
		value, err := core.NewPositionFromInput(row, column)
		if err != nil {
			return core.Position{}, err
		}
		return *value, nil
	}
	validRevision := func(value int64) error {
		if value < 0 {
			return errors.New("expected_revision must be non-negative")
		}
		return nil
	}
	switch kind {
	case "set-value":
		v, e := body.AsSetValueAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil || v.Value < 1 || v.Value > 9 {
			return nil, 0, errors.New("invalid set-value action")
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.SetValue{Position: p, Value: v.Value}, v.ExpectedRevision, nil
	case "clear-value":
		v, e := body.AsClearValueAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.ClearValue{Position: p}, v.ExpectedRevision, nil
	case "set-candidates":
		v, e := body.AsSetCandidatesAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.SetCandidates{Position: p, Values: append([]int(nil), v.Values...)}, v.ExpectedRevision, nil
	case "reset":
		v, e := body.AsResetAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Reset{}, v.ExpectedRevision, nil
	case "restart":
		v, e := body.AsRestartAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Restart{}, v.ExpectedRevision, nil
	case "undo":
		v, e := body.AsUndoAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Undo{}, v.ExpectedRevision, nil
	case "redo":
		v, e := body.AsRedoAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Redo{}, v.ExpectedRevision, nil
	case "apply-hint":
		v, e := body.AsApplyHintAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.ApplyHint{PlanID: v.PlanId}, v.ExpectedRevision, nil
	case "repair":
		v, e := body.AsRepairAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Repair{}, v.ExpectedRevision, nil
	case "solve":
		v, e := body.AsSolveAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Solve{}, v.ExpectedRevision, nil
	default:
		return nil, 0, errors.New("unknown action kind")
	}
}

func apiStatus(v game.Status) GameStatus { return GameStatus(v) }
func digits(set core.CandidateSet) []int { return set.Values() }
func apiSnapshot(v game.Snapshot) Snapshot {
	o := Snapshot{Status: apiStatus(v.Status), Mistakes: v.Mistakes, CanUndo: v.CanUndo, CanRedo: v.CanRedo, Givens: make(ValueGrid, 9), Values: make(ValueGrid, 9), Invalid: make(BooleanGrid, 9), Candidates: make(DigitSetGrid, 9)}
	for r := 0; r < 9; r++ {
		o.Givens[r] = make(ValueRow, 9)
		o.Values[r] = make(ValueRow, 9)
		o.Invalid[r] = make(BooleanRow, 9)
		o.Candidates[r] = make(DigitSetRow, 9)
		for c := 0; c < 9; c++ {
			o.Givens[r][c] = v.Givens[r][c]
			o.Values[r][c] = v.Values[r][c]
			o.Invalid[r][c] = v.Invalid[r][c]
			o.Candidates[r][c] = digits(v.Candidates[r][c])
		}
	}
	return o
}
func apiResult(v game.Result) ActionResult {
	o := ActionResult{Action: ActionKind(v.Action), Status: apiStatus(v.Status), CanUndo: v.CanUndo, CanRedo: v.CanRedo, Changes: make([]CellChange, len(v.Changes))}
	for i, c := range v.Changes {
		o.Changes[i] = CellChange{Row: c.Position.Row + 1, Column: c.Position.Column + 1, Before: c.Before, After: c.After, InvalidBefore: c.InvalidBefore, InvalidAfter: c.InvalidAfter, CandidatesBefore: digits(c.CandidatesBefore), CandidatesAfter: digits(c.CandidatesAfter)}
	}
	if v.Hint != nil {
		hint := apiHintPlan(v.Hint)
		o.Hint = &hint
	}
	return o
}

func apiHintPlan(plan *game.HintPlan) HintPlan {
	steps := make([]HintStep, len(plan.Steps))
	for i, step := range plan.Steps {
		marks := make([]HintMark, len(step.Marks))
		for j, mark := range step.Marks {
			marks[j] = HintMark{Target: apiHintTarget(mark.Target), Role: HintRole(mark.Role)}
		}
		nodes := make([]HintNode, len(step.Nodes))
		for j, node := range step.Nodes {
			nodes[j] = HintNode{Id: node.ID, Target: apiHintTarget(node.Target)}
			if node.Group != "" {
				group := HintNodeGroup(node.Group)
				nodes[j].Group = &group
			}
			if node.Order != 0 {
				order := node.Order
				nodes[j].Order = &order
			}
		}
		links := make([]HintLink, len(step.Links))
		for j, link := range step.Links {
			links[j] = HintLink{From: link.From, To: link.To, Kind: HintLinkKind(link.Kind)}
		}
		steps[i] = HintStep{Id: step.ID, Kind: HintStepKind(step.Kind), Message: apiMessageDescriptor(step.Message), Marks: marks, Nodes: nodes, Links: links}
		if step.HasValidNarration() {
			narration := make([]HintNarrationChunk, len(step.Narration))
			valid := true
			for j, chunk := range step.Narration {
				var ok bool
				narration[j], ok = apiHintNarrationChunk(chunk)
				if !ok {
					valid = false
					break
				}
			}
			if valid {
				steps[i].Narration = &narration
			}
		}
		if step.Effect != nil {
			effect := apiHintEffect(*step.Effect)
			steps[i].Effect = &effect
		}
	}
	out := HintPlan{
		PlanId:     plan.PlanID,
		Kind:       HintPlanKind(plan.Kind),
		Title:      apiMessageDescriptor(plan.Title),
		Summary:    apiMessageDescriptor(plan.Summary),
		Steps:      steps,
		Conclusion: apiHintEffect(game.HintEffect{Placement: plan.Conclusion.Placement, Eliminations: plan.Conclusion.Eliminations, ClearValue: plan.Conclusion.ClearValue, SetCandidates: plan.Conclusion.SetCandidates}),
	}
	if plan.Strategy.ID != "" {
		out.Strategy = &HintStrategy{Id: plan.Strategy.ID, Name: apiMessageDescriptor(plan.Strategy.Name), Grade: Difficulty(plan.Strategy.Grade), GradeName: apiMessageDescriptor(plan.Strategy.GradeName)}
	}
	return out
}

func mustAPIMessageDescriptor(descriptor usermessage.Descriptor, err error) MessageDescriptor {
	if err != nil {
		panic(fmt.Sprintf("render canonical English message: %v", err))
	}
	return apiMessageDescriptor(descriptor)
}

func apiMessageDescriptor(descriptor usermessage.Descriptor) MessageDescriptor {
	arguments := make([]MessageArgument, len(descriptor.Arguments))
	for i, argument := range descriptor.Arguments {
		arguments[i] = mustAPIMessageArgument(argument)
	}
	return MessageDescriptor{Id: descriptor.ID, Arguments: arguments, Text: descriptor.Text}
}

func mustAPIMessageArgument(argument usermessage.Argument) MessageArgument {
	var out MessageArgument
	var err error
	switch argument.Kind {
	case usermessage.ArgumentText:
		err = out.FromMessageTextArgument(MessageTextArgument{Name: argument.Name, Kind: string(argument.Kind), Text: *argument.Text})
	case usermessage.ArgumentInteger:
		err = out.FromMessageIntegerArgument(MessageIntegerArgument{Name: argument.Name, Kind: string(argument.Kind), Integer: *argument.Integer})
	case usermessage.ArgumentDigit:
		err = out.FromMessageDigitArgument(MessageDigitArgument{Name: argument.Name, Kind: string(argument.Kind), Digit: Digit(*argument.Digit)})
	case usermessage.ArgumentCell:
		err = out.FromMessageCellArgument(MessageCellArgument{Name: argument.Name, Kind: string(argument.Kind), Cell: CellPosition{Row: argument.Cell.Row, Column: argument.Cell.Column}})
	case usermessage.ArgumentCandidate:
		value := MessageCandidateArgument{Name: argument.Name, Kind: string(argument.Kind)}
		value.Candidate.Cell = CellPosition{Row: argument.Candidate.Cell.Row, Column: argument.Candidate.Cell.Column}
		value.Candidate.Digit = Digit(argument.Candidate.Digit)
		err = out.FromMessageCandidateArgument(value)
	case usermessage.ArgumentRow:
		err = out.FromMessageRowArgument(MessageRowArgument{Name: argument.Name, Kind: string(argument.Kind), Row: *argument.Row})
	case usermessage.ArgumentColumn:
		err = out.FromMessageColumnArgument(MessageColumnArgument{Name: argument.Name, Kind: string(argument.Kind), Column: *argument.Column})
	case usermessage.ArgumentBox:
		err = out.FromMessageBoxArgument(MessageBoxArgument{Name: argument.Name, Kind: string(argument.Kind), Box: *argument.Box})
	case usermessage.ArgumentDifficulty:
		err = out.FromMessageDifficultyArgument(MessageDifficultyArgument{Name: argument.Name, Kind: string(argument.Kind), Difficulty: Difficulty(*argument.Difficulty)})
	case usermessage.ArgumentStrategy:
		err = out.FromMessageStrategyArgument(MessageStrategyArgument{Name: argument.Name, Kind: string(argument.Kind), Strategy: *argument.Strategy})
	case usermessage.ArgumentCount:
		err = out.FromMessageCountArgument(MessageCountArgument{Name: argument.Name, Kind: string(argument.Kind), Count: *argument.Count})
	default:
		panic(fmt.Sprintf("unsupported canonical message argument kind %q", argument.Kind))
	}
	if err != nil {
		panic(fmt.Sprintf("encode canonical message argument %q: %v", argument.Name, err))
	}
	return out
}

func apiHintNarrationChunk(chunk game.HintNarrationChunk) (HintNarrationChunk, bool) {
	var out HintNarrationChunk
	var err error
	switch chunk.Kind {
	case game.HintNarrationMessage:
		if chunk.Message == nil {
			return HintNarrationChunk{}, false
		}
		err = out.FromHintNarrationMessage(HintNarrationMessage{Message: apiMessageDescriptor(*chunk.Message)})
	case game.HintNarrationTarget:
		err = out.FromHintNarrationTarget(HintNarrationTarget{Target: apiHintTarget(*chunk.Target)})
	case game.HintNarrationDigit:
		err = out.FromHintNarrationDigit(HintNarrationDigit{Value: Digit(chunk.Value)})
	default:
		return HintNarrationChunk{}, false
	}
	return out, err == nil
}

func apiHintTarget(target game.HintTarget) HintTarget {
	out := HintTarget{Kind: HintTargetKind(target.Kind)}
	if target.Position != nil {
		row, column := target.Position.Row+1, target.Position.Column+1
		out.Row, out.Column = &row, &column
	}
	if target.Value != 0 {
		value := Digit(target.Value)
		out.Value = &value
	}
	if target.Kind == game.HintTargetRow || target.Kind == game.HintTargetColumn || target.Kind == game.HintTargetBox {
		index := target.Index + 1
		out.Index = &index
	}
	return out
}

func apiHintEffect(effect game.HintEffect) HintEffect {
	out := HintEffect{}
	switch {
	case effect.ClearValue != nil:
		out.Kind = HintEffectKindClearValue
		out.ClearValue = &CellPosition{Row: effect.ClearValue.Row + 1, Column: effect.ClearValue.Column + 1}
	case effect.SetCandidates != nil:
		out.Kind = HintEffectKindSetCandidates
		out.SetCandidates = &struct {
			Column int      `json:"column"`
			Row    int      `json:"row"`
			Values DigitSet `json:"values"`
		}{Column: effect.SetCandidates.Position.Column + 1, Row: effect.SetCandidates.Position.Row + 1, Values: digits(effect.SetCandidates.Values)}
	case effect.Placement != nil:
		out.Kind = HintEffectKindPlaceValue
		out.Placement = &HintPlacement{Row: effect.Placement.Position.Row + 1, Column: effect.Placement.Position.Column + 1, Value: effect.Placement.Value}
	default:
		out.Kind = HintEffectKindEliminateCandidates
		eliminations := make([]HintElimination, len(effect.Eliminations))
		for i, elimination := range effect.Eliminations {
			eliminations[i] = HintElimination{Row: elimination.Position.Row + 1, Column: elimination.Position.Column + 1, Value: elimination.Value}
		}
		out.Eliminations = &eliminations
	}
	return out
}

func apiError(code ErrorCode, _ string) Error {
	message := mustAPIMessageDescriptor(canonicalEnglishMessages.Render("error." + string(code)))
	return Error{Error: ErrorDetail{Code: code, Message: message}}
}
func engineAPIError(e *game.EngineError) Error {
	out := apiError(ErrorCode(e.Code), e.Error())
	if e.Position != nil {
		out.Error.Cell = &CellPosition{Row: e.Position.Row + 1, Column: e.Position.Column + 1}
	}
	return out
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, apiError(code, message))
}
func decodeStrict(data []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validateJSONRequest(path string, data []byte) error {
	var object map[string]json.RawMessage
	if err := decodeStrict(data, &object); err != nil {
		return err
	}
	switch {
	case path == "/api/v1/guest/games" || path == "/api/v1/account/games":
		if err := requireKeys(object, "source"); err != nil {
			return err
		}
		var source map[string]json.RawMessage
		if err := decodeStrict(object["source"], &source); err != nil {
			return err
		}
		var kind string
		if err := json.Unmarshal(source["kind"], &kind); err != nil {
			return errors.New("source kind is required")
		}
		switch kind {
		case "difficulty":
			return requireKeys(source, "kind", "difficulty")
		case "puzzle":
			return requireKeys(source, "kind", "puzzle")
		default:
			return errors.New("unknown source kind")
		}
	case path == "/api/v1/account/games/claim":
		return requireKeys(object, "document")
	case path == "/api/v1/guest/games/actions":
		if err := requireKeys(object, "document", "action"); err != nil {
			return err
		}
		var action map[string]json.RawMessage
		if err := decodeStrict(object["action"], &action); err != nil {
			return err
		}
		return validateActionObject(action)
	case strings.HasSuffix(path, "/actions"):
		return validateActionObject(object)
	default:
		return nil
	}
}

func validateActionObject(object map[string]json.RawMessage) error {
	var kind string
	if err := json.Unmarshal(object["kind"], &kind); err != nil {
		return errors.New("action kind is required")
	}
	keys := []string{"kind", "expected_revision"}
	switch kind {
	case "set-value":
		keys = append(keys, "row", "column", "value")
	case "set-candidates":
		keys = append(keys, "row", "column", "values")
	case "clear-value":
		keys = append(keys, "row", "column")
	case "apply-hint":
		keys = append(keys, "plan_id")
	case "reset", "restart", "undo", "redo", "repair", "solve":
	default:
		return errors.New("unknown action kind")
	}
	return requireKeys(object, keys...)
}

func requireKeys(object map[string]json.RawMessage, keys ...string) error {
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
		if _, ok := object[key]; !ok {
			return fmt.Errorf("field %q is required", key)
		}
	}
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}
