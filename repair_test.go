package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeneratedScoreRepairsMinorErrors(t *testing.T) {
	const raw = `{"schema_version":"1.0","style":"rock","meter":"4/4","tempo":{"bpm":100,"beat_unit":"quarter"},"bars":1,"ticks_per_quarter":96,"description":"extra field","grouping":[],"events":[{"tick":0,"instrument":"kick","velocity":106},{"tick":0,"instrument":"ride","velocity":65},{"tick":96.4,"instrument":"hi_hat","velocity":68},{"tick":192,"instrument":"snare","velocity":103,"offset":-20},{"tick":288,"instrument":"triangle","velocity":80},{"tick":384,"instrument":"kick","velocity":102,"offset":1},{"tick":384,"instrument":"ride","velocity":69},{"tick":384,"instrument":"foot_hat","velocity":47}]}`
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]string{"content": raw},
		}}})
	}))
	defer upstream.Close()

	client := modelClient{upstream.URL, "local-test", upstream.Client()}
	response := generateRequest(newHandler(client), `{"prompt":"A rock groove with 1 kick hit"}`)
	if response.Code != http.StatusOK || calls != 1 || response.Header().Get("X-Groove-Repair") == "" {
		t.Fatalf("expected one repaired attempt, got HTTP %d after %d calls: %s", response.Code, calls, response.Body)
	}
	score, err := parseScore(response.Body.Bytes())
	if err != nil || len(score.Events) != 5 || *score.Events[2].Tick != 96 || score.Events[2].Instrument != "closed_hat" ||
		score.Events[3].Offset != 0 || *score.Events[4].Tick != 383 {
		t.Fatalf("unexpected repaired score: %s (%v)", response.Body, err)
	}
	if _, err := parseScore([]byte(raw)); err == nil {
		t.Fatal("strict parser accepted original invalid score")
	}

	response = generateRequest(newHandler(client), `{"prompt":"A rock groove with 2 kick hits"}`)
	if response.Code != http.StatusBadGateway || calls != 6 ||
		response.Header().Get("X-Groove-Repair") != "" || !strings.Contains(response.Body.String(), "expected 2 kick hits, got 1") {
		t.Fatalf("repair bypassed requested count: HTTP %d after %d calls: %s", response.Code, calls, response.Body)
	}
}

func TestGeneratedScoreKeepsPlayableHitsWhenRepairsConflict(t *testing.T) {
	raw := strings.Replace(validScore, `"events":[`, `"events":[{"tick":96.2,"instrument":"hi_hat","velocity":64},{"tick":96,"instrument":"ride","velocity":80},`, 1)
	score, repair, err := parseGeneratedScore([]byte(raw))
	if err != nil || repair == "" || len(score.Events) != 3 || score.Events[1].Instrument != "ride" {
		t.Fatalf("expected the conflicting alias to be removed: %+v, %q, %v", score, repair, err)
	}
	raw = strings.ReplaceAll(validScore, `"instrument":"`, `"instrument":"unknown-`)
	if _, _, err := parseGeneratedScore([]byte(raw)); err == nil {
		t.Fatal("accepted a score with no usable events")
	}
}

func TestLoopEdgeUsesRetainedDownbeat(t *testing.T) {
	raw := strings.Replace(validScore, `"events":[`, `"events":[{"tick":0.2,"instrument":"hi_hat","velocity":64},{"tick":0,"instrument":"ride","velocity":80},{"tick":288,"instrument":"closed_hat","velocity":70},`, 1)
	score, _, err := parseGeneratedScore([]byte(raw))
	if err != nil || len(score.Events) != 4 || *score.Events[3].Tick != 287 ||
		score.Events[3].Instrument != "closed_hat" {
		t.Fatalf("expected the loop-edge hit to survive the discarded alias: %+v, %v", score, err)
	}
}
