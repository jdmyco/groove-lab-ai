package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const validScore = `{"schema_version":"1.0","style":"blues","meter":"6/8","tempo":{"bpm":62,"beat_unit":"dotted_quarter"},"bars":1,"ticks_per_quarter":96,"events":[{"tick":144,"instrument":"snare","velocity":100},{"tick":0,"instrument":"kick","velocity":110}]}`

func TestParseScore(t *testing.T) {
	score, err := parseScore([]byte(validScore))
	if err != nil || len(score.Events) != 2 || *score.Events[0].Tick != 0 || *score.Events[1].Tick != 144 {
		t.Fatalf("expected canonical 6/8 score, got %+v, %v", score, err)
	}
	for name, invalid := range map[string]string{
		"missing tick":     strings.Replace(validScore, `"tick":0,`, "", 1),
		"null tick":        strings.Replace(validScore, `"tick":0`, `"tick":null`, 1),
		"late event":       strings.Replace(validScore, `"tick":144`, `"tick":288`, 1),
		"unknown field":    strings.Replace(validScore, `"instrument":"kick"`, `"instrument":"kick","note":1`, 1),
		"duplicate key":    strings.Replace(validScore, `"tick":0`, `"tick":0,"tick":1`, 1),
		"duplicate hit":    strings.Replace(validScore, `"tick":144,"instrument":"snare"`, `"tick":0,"instrument":"kick"`, 1),
		"hat conflict":     strings.NewReplacer(`"tick":144`, `"tick":0`, `"snare"`, `"open_hat"`, `"kick"`, `"closed_hat"`).Replace(validScore),
		"trailing content": validScore + "{}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseScore([]byte(invalid)); err == nil {
				t.Fatal("accepted invalid score")
			}
		})
	}
}

func TestGenerateRetriesMalformedScores(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Model    string
			Messages []struct{ Role, Content string }
		}
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" ||
			json.NewDecoder(r.Body).Decode(&request) != nil ||
			request.Model != "local-test" || len(request.Messages) != 2 {
			t.Error("invalid upstream request")
			http.Error(w, "invalid request", 400)
			return
		}
		call := calls.Add(1)
		if request.Messages[0].Content != systemPrompt || !strings.HasPrefix(request.Messages[1].Content, "A 6/8 blues groove") {
			t.Error("upstream request lost the score instructions or prompt")
		}
		content := validScore
		switch call {
		case 1:
			content = "oops"
		case 2:
			content = strings.Replace(validScore, `"tick":144`, `"tick":288`, 1)
		}
		if call > 1 && !strings.Contains(request.Messages[1].Content, "Previous attempt") {
			t.Error("retry missing validation feedback")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]string{"content": content},
		}}})
	}))
	defer upstream.Close()

	handler := newHandler(modelClient{upstream.URL + "/v1/chat/completions", "local-test", upstream.Client()})
	response := generateRequest(handler, `{"prompt":"A 6/8 blues groove"}`)
	if response.Code != 200 || calls.Load() != 3 {
		t.Fatalf("got HTTP %d after %d calls: %s", response.Code, calls.Load(), response.Body)
	}
	if _, err := parseScore(response.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateBoundsRetries(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"choices":[]}`)
	}))
	defer upstream.Close()

	handler := newHandler(modelClient{upstream.URL, "local-test", upstream.Client()})
	response := generateRequest(handler, `{"prompt":" "}`)
	if response.Code != 400 || calls.Load() != 0 {
		t.Fatalf("invalid prompt: got HTTP %d after %d calls", response.Code, calls.Load())
	}
	response = generateRequest(handler, `{"prompt":"A shuffle"}`)
	if response.Code != 502 || calls.Load() != 3 {
		t.Fatalf("invalid scores: got HTTP %d after %d calls", response.Code, calls.Load())
	}
	upstream.Close()
	response = generateRequest(handler, `{"prompt":"A shuffle"}`)
	if response.Code != 503 {
		t.Fatalf("offline model: got HTTP %d", response.Code)
	}
}

func generateRequest(handler http.Handler, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/generate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestTrainingCorpus(t *testing.T) {
	handler := newHandler(modelClient{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/training/train.jsonl", nil))
	if response.Code != 200 || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/x-ndjson") {
		t.Fatal("gallery corpus route did not return JSONL")
	}
	rows := strings.Split(strings.TrimSpace(response.Body.String()), "\n")
	if len(rows) != 100 {
		t.Fatalf("found %d rows, want 100", len(rows))
	}

	prompts := make(map[string]bool)
	patterns := make(map[string]bool)
	styles := make(map[string]int)
	meters := make(map[string]bool)
	lengths := make(map[int]bool)
	for index, line := range rows {
		var record struct {
			Messages []message `json:"messages"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("row %d: %v", index+1, err)
		}
		if len(record.Messages) != 3 || record.Messages[0] != (message{"system", systemPrompt}) ||
			record.Messages[1].Role != "user" || record.Messages[2].Role != "assistant" {
			t.Fatalf("row %d has unexpected messages", index+1)
		}
		prompt := strings.ToLower(strings.TrimSpace(record.Messages[1].Content))
		if prompt == "" || prompts[prompt] {
			t.Fatalf("row %d has a blank or repeated prompt", index+1)
		}
		prompts[prompt] = true

		rawScore := record.Messages[2].Content
		score, err := parseScore([]byte(rawScore))
		if err != nil {
			t.Fatalf("row %d: %v", index+1, err)
		}
		canonical, err := json.Marshal(score)
		if err != nil || string(canonical) != rawScore {
			t.Fatalf("row %d is not a compact, sorted score", index+1)
		}
		var pattern strings.Builder
		fmt.Fprintf(&pattern, "%s/%d ", score.Meter, score.Bars)
		for _, event := range score.Events {
			fmt.Fprintf(&pattern, "%d/%s ", *event.Tick, event.Instrument)
		}
		if patterns[pattern.String()] {
			t.Fatalf("row %d repeats an existing rhythm", index+1)
		}
		patterns[pattern.String()] = true
		styles[score.Style]++
		meters[score.Meter] = true
		lengths[score.Bars] = true
	}
	for _, style := range []string{"rock", "funk", "blues", "jazz"} {
		if styles[style] != 25 {
			t.Errorf("found %d %s scores, want 25", styles[style], style)
		}
	}
	if len(meters) != 4 || len(lengths) != 4 {
		t.Error("corpus should cover all supported meters and bar counts")
	}
}

func TestLoopbackHost(t *testing.T) {
	handler := loopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for host, want := range map[string]int{
		"localhost:8080": 204,
		"127.0.0.1:8080": 204,
		"localhost":      204,
		"other.example":  403,
	} {
		request := httptest.NewRequest("POST", "/api/generate", nil)
		request.Host = host
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != want {
			t.Errorf("host %s: got HTTP %d, want %d", host, response.Code, want)
		}
	}
}
