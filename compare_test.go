package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"
)

func TestComparisonRoutesUseSeparateModels(t *testing.T) {
	const prompt = "A blues groove with 1 bass hit"
	var baseCalls, adaptedCalls atomic.Int32
	stub := func(model string, calls *atomic.Int32, responses ...string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Model    string
				Messages []message
			}
			if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload.Messages) != 2 {
				t.Error("expected two chat messages")
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			call := int(calls.Add(1))
			if payload.Model != model || payload.Messages[0] != (message{"system", systemPrompt + "\n" + comparisonScoreExample}) ||
				payload.Messages[1].Role != "user" || !strings.HasPrefix(payload.Messages[1].Content, prompt) {
				t.Errorf("wrong model or prompt: %+v", payload)
			}
			if (call > 1) != strings.Contains(payload.Messages[1].Content, "Previous attempt failed validation") {
				t.Errorf("wrong retry feedback on call %d", call)
			}
			if call == 2 && !strings.Contains(payload.Messages[1].Content, "Go main.Tempo") {
				t.Errorf("missing invalid tempo field in retry feedback: %s", payload.Messages[1].Content)
			}
			if call == 3 && !strings.Contains(payload.Messages[1].Content, "expected 1 kick hit, got 0") {
				t.Errorf("missing count failure on retry: %s", payload.Messages[1].Content)
			}
			content := responses[min(call, len(responses))-1]
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
				"message": map[string]string{"content": content},
			}}})
		}))
	}
	withoutKick := strings.Replace(validScore, `,{"tick":0,"instrument":"kick","velocity":110}`, "", 1)
	louder := strings.Replace(validScore, `"velocity":110`, `"velocity":118`, 1)
	wrongShape := strings.Replace(validScore, `"tempo":{"bpm":62,"beat_unit":"dotted_quarter"}`, `"tempo":"slow"`, 1)
	base := stub("base-test", &baseCalls, wrongShape, withoutKick, "malformed", "malformed", validScore)
	defer base.Close()
	adapted := stub("adapted-test", &adaptedCalls, wrongShape, withoutKick, louder)
	defer adapted.Close()

	mux := newHandler(modelClient{})
	registerComparison(mux,
		modelClient{base.URL, "base-test", base.Client()},
		modelClient{adapted.URL, "adapted-test", adapted.Client()},
	)
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest("GET", "/compare", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Header().Get("Content-Type"), "text/html") ||
		!strings.Contains(page.Body.String(), "Compare Qwen scores") {
		t.Fatalf("comparison page unavailable: HTTP %d", page.Code)
	}
	for _, tc := range []struct {
		path, attempts string
		velocity       int
	}{
		{"/api/compare/base", "5", 110},
		{"/api/compare/adapted", "3", 118},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", tc.path, strings.NewReader(`{"prompt":"`+prompt+`"}`))
		r.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(w, r)
		score, err := parseScore(w.Body.Bytes())
		if w.Code != http.StatusOK || err != nil || score.Events[0].Velocity != tc.velocity ||
			w.Header().Get("X-Groove-Attempts") != tc.attempts {
			t.Errorf("%s returned HTTP %d with %s attempts: %s", tc.path, w.Code, w.Header().Get("X-Groove-Attempts"), w.Body)
		}
	}
	if baseCalls.Load() != 5 || adaptedCalls.Load() != 3 {
		t.Errorf("unexpected calls: base=%d adapted=%d", baseCalls.Load(), adaptedCalls.Load())
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/compare/base", strings.NewReader(`{"prompt":"funk","engine":"procedural"}`))
	r.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || baseCalls.Load() != 5 {
		t.Errorf("comparison endpoint accepted a procedural override: HTTP %d", w.Code)
	}
}

func TestScoreDecodeFeedbackIsBounded(t *testing.T) {
	unknown := `"` + strings.Repeat("é", 512) + `":1,"bars":1`
	content := strings.Replace(validScore, `"bars":1`, unknown, 1)
	_, err := parseScore([]byte(content))
	if err == nil || !utf8.ValidString(err.Error()) || len(err.Error()) > 320 ||
		!strings.Contains(err.Error(), "unknown object member") {
		t.Fatalf("expected bounded score diagnostics, got %v", err)
	}
}
