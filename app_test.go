package main

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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
			content = strings.Replace(validScore, `"bpm":62`, `"bpm":12`, 1)
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
	if score, err := parseScore(response.Body.Bytes()); err != nil || score.Metadata == nil ||
		score.Metadata.Feel == "" || score.Metadata.Timing == "" {
		t.Fatalf("model response should include score metadata: %v", err)
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
	if response.Code != 502 || calls.Load() != 5 || response.Header().Get("X-Groove-Attempts") != "5" {
		t.Fatalf("invalid scores: got HTTP %d after %d calls, header %s", response.Code, calls.Load(), response.Header().Get("X-Groove-Attempts"))
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
	if len(rows) != 1330 {
		t.Fatalf("found %d rows, want 1330", len(rows))
	}

	prompts := make(map[string]bool)
	patterns := make(map[string]bool)
	families := make(map[string]bool)
	styleCounts := make(map[string]int)
	meterCounts := make(map[string]bool)
	lengths := make(map[int]bool)
	countedRows := 0
	for index, line := range rows {
		var record catalogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("row %d: %v", index+1, err)
		}
		if len(record.Messages) != 3 || record.Messages[0] != (message{"system", systemPrompt}) ||
			record.Messages[1].Role != "user" || record.Messages[2].Role != "assistant" {
			t.Fatalf("row %d has unexpected messages", index+1)
		}
		prompt := strings.TrimSpace(record.Messages[1].Content)
		if prompt == "" || prompts[strings.ToLower(prompt)] {
			t.Fatalf("row %d has a blank or repeated prompt", index+1)
		}
		prompts[strings.ToLower(prompt)] = true
		rawScore := record.Messages[2].Content
		score, err := parseScore([]byte(rawScore))
		if err != nil {
			t.Fatalf("row %d: %v", index+1, err)
		}
		if score.Metadata == nil || score.Metadata.Feel == "" || score.Metadata.Timing == "" {
			t.Fatalf("row %d is missing musical metadata", index+1)
		}
		canonical, err := jsonv2.Marshal(score)
		if err != nil || string(canonical) != rawScore {
			t.Fatalf("row %d is not a compact, sorted score", index+1)
		}
		brief, err := parseBrief(prompt)
		if err != nil ||
			brief.meter != "" && brief.meter != score.Meter ||
			brief.bars != 0 && brief.bars != score.Bars ||
			brief.bpm != 0 && brief.bpm != score.Tempo.BPM ||
			brief.beatUnit != "" && brief.beatUnit != score.Tempo.BeatUnit ||
			len(brief.grouping) != 0 && !slices.Equal(brief.grouping, score.Grouping) {
			t.Fatalf("row %d score does not match its prompt: %s (%v)", index+1, prompt, err)
		}

		counts, _, err := parseHitCounts(prompt)
		if err != nil {
			t.Fatalf("row %d could not parse counts: %v", index+1, err)
		}
		if len(counts) > 0 {
			countedRows++
			if err := verifyHitCounts(score, counts); err != nil {
				t.Fatalf("row %d: %s: %v", index+1, prompt, err)
			}
			payload, _ := json.Marshal(map[string]string{"engine": "procedural", "prompt": prompt})
			response := generateRequest(handler, string(payload))
			if response.Code != http.StatusOK {
				t.Fatalf("row %d: %s: HTTP %d: %s", index+1, prompt, response.Code, response.Body)
			}
			generated, err := parseScore(response.Body.Bytes())
			if err != nil {
				t.Fatalf("row %d generated invalid score: %v", index+1, err)
			}
			if generated.Style != score.Style || generated.Meter != score.Meter || generated.Bars != score.Bars || generated.Tempo != score.Tempo {
				t.Fatalf("row %d generated incorrect style, meter, length, or tempo", index+1)
			}
			if err := verifyHitCounts(generated, counts); err != nil {
				t.Fatalf("row %d generated incorrect hit totals: %v", index+1, err)
			}
		}

		if record.Groove.CycleBars < 1 || score.Bars%record.Groove.CycleBars != 0 || record.Groove.Family == "" {
			t.Fatalf("row %d has invalid cycle or family metadata", index+1)
		}
		barTicks := meters[score.Meter].ticks()
		cycleTicks := barTicks * record.Groove.CycleBars
		var pattern strings.Builder
		fmt.Fprintf(&pattern, "%s ", score.Meter)
		cycle := make([]Event, 0)
		for _, event := range score.Events {
			if *event.Tick < cycleTicks {
				cycle = append(cycle, event)
				fmt.Fprintf(&pattern, "%d/%s ", *event.Tick, event.Instrument)
			}
		}
		if patterns[pattern.String()] {
			t.Fatalf("row %d repeats an existing rhythm at another tempo or length", index+1)
		}
		patterns[pattern.String()] = true
		if len(cycle)*score.Bars/record.Groove.CycleBars != len(score.Events) {
			t.Fatalf("row %d changes density on the way to the loop seam", index+1)
		}
		for sequence, event := range score.Events {
			first := cycle[sequence%len(cycle)]
			if *event.Tick != *first.Tick+(sequence/len(cycle))*cycleTicks ||
				event.Instrument != first.Instrument || event.Offset != first.Offset || event.Velocity != first.Velocity {
				t.Fatalf("row %d changes the groove on a repeated cycle", index+1)
			}
		}
		for bar := 0; bar < score.Bars; bar++ {
			keeper, integratedTom := false, false
			hasTom := false
			for _, event := range score.Events {
				if *event.Tick/barTicks != bar {
					continue
				}
				withinBar := *event.Tick % barTicks
				switch event.Instrument {
				case "ride", "closed_hat", "open_hat", "cowbell", "shaker", "clave":
					keeper = keeper || withinBar >= barTicks-144
				case "tom_high", "tom_mid", "tom_low":
					hasTom = true
					integratedTom = integratedTom || withinBar < barTicks-96
				}
			}
			if !keeper || hasTom && !integratedTom {
				t.Fatalf("row %d bar %d drops timekeeping or leaves toms for the ending", index+1, bar+1)
			}
		}
		families[record.Groove.Family] = true
		styleCounts[score.Style]++
		meterCounts[score.Meter] = true
		lengths[score.Bars] = true
	}
	if countedRows != 240 {
		t.Errorf("found %d count-focused presets, want 240", countedRows)
	}
	if len(styleCounts) != len(styles) || len(meterCounts) != len(meters) || len(lengths) != 4 || len(families) != 74 {
		t.Error("corpus should cover ten styles, seven meters, four bar counts and 74 groove families")
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

func TestProceduralGeneration(t *testing.T) {
	handler := newHandler(modelClient{})
	for _, tc := range []struct {
		prompt, style, meter string
		bpm, bars            int
	}{
		{"Play a Steve Gadd pocket at 103 BPM for two bars", "funk", "4/4", 103, 2},
		{"A funky upbeat song", "funk", "4/4", 0, 0},
		{"A 7/8 fusion groove grouped 2+2+3 at 120 BPM, 2 bars", "fusion", "7/8", 120, 2},
		{"A 5/4 jazz groove grouped 2+3, BPM 132", "jazz", "5/4", 132, 0},
		{"Funk with four on the floor at 112 bpm", "funk", "4/4", 112, 0},
		{"One bar of reggae one drop at 82 BPM", "reggae", "4/4", 82, 1},
		{"A jazzy 3/4 swing groove at 145 bpm", "jazz", "3/4", 145, 0},
	} {
		t.Run(tc.prompt, func(t *testing.T) {
			request, _ := json.Marshal(map[string]string{"prompt": tc.prompt, "engine": "procedural"})
			response := generateRequest(handler, string(request))
			if response.Code != 200 {
				t.Fatalf("got HTTP %d: %s", response.Code, response.Body)
			}
			score, err := parseScore(response.Body.Bytes())
			if err != nil || score.Style != tc.style || score.Meter != tc.meter ||
				tc.bpm != 0 && score.Tempo.BPM != tc.bpm || tc.bars != 0 && score.Bars != tc.bars {
				t.Fatalf("did not honor prompt, got %+v, %v", score, err)
			}
			if strings.Contains(tc.prompt, "four on the floor") {
				for tick := 0; tick < score.Bars*384; tick += 96 {
					if !slices.ContainsFunc(score.Events, func(event Event) bool {
						return event.Instrument == "kick" && *event.Tick == tick
					}) {
						t.Fatalf("missing four-on-the-floor kick at tick %d", tick)
					}
				}
			}
			if again := generateRequest(handler, string(request)); again.Body.String() != response.Body.String() {
				t.Fatal("same prompt changed the returned score")
			}
		})
	}
	for _, prompt := range []string{
		"funk 11/8", "reggae at 420 BPM", "7/8 grouped 3+3", "funk 9 bars",
		"reggae at -82 BPM", "funk at −82 bpm", "funk BPM 120.82", "4/4 jazz at 105 BPM with cowbell",
		"funk at 110 BPM with shaker and floor tom",
		"4/4 jazz swing with no hi-hat", "Steve Gadd in 7/8",
	} {
		request, _ := json.Marshal(map[string]string{"prompt": prompt, "engine": "procedural"})
		if response := generateRequest(handler, string(request)); response.Code != 422 {
			t.Errorf("expected unsupported request to return 422: %s, got %d", prompt, response.Code)
		}
	}
}

func TestGroupedScoreAndTimingOffset(t *testing.T) {
	raw := `{"schema_version":"1.0","style":"fusion","meter":"7/8","grouping":[2,2,3],"tempo":{"bpm":118,"beat_unit":"quarter"},"bars":1,"ticks_per_quarter":96,"events":[{"tick":0,"instrument":"kick","velocity":100},{"tick":96,"instrument":"snare","velocity":99,"offset":3}]}`
	if _, err := parseScore([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(raw, "[2,2,3]", "[2,2,2]", 1),
		strings.Replace(raw, `"offset":3`, `"offset":8`, 1),
		strings.NewReplacer(`"tick":96`, `"tick":0`, `"offset":3`, `"offset":-1`).Replace(raw),
	} {
		if _, err := parseScore([]byte(bad)); err == nil {
			t.Fatal("accepted invalid group or offset")
		}
	}
}

func TestCatalogPromptsGenerate(t *testing.T) {
	catalog, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range strings.Split(strings.TrimSpace(string(corpus)), "\n") {
		var record catalogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		prompt := record.Messages[1].Content
		_, seed, err := parseHitCounts(prompt)
		if err != nil {
			t.Fatalf("row %d cannot be parsed: %v", index+1, err)
		}
		brief, err := parseBrief(seed)
		if err != nil {
			t.Errorf("row %d cannot be parsed: %v", index+1, err)
			continue
		}
		if _, ok := ranking(brief, catalog[index]); !ok {
			t.Errorf("row %d is incompatible with its own prompt: %s", index+1, prompt)
		}
		if _, err := chooseGroove(brief, catalog); err != nil {
			t.Errorf("row %d does not match the catalog: %s: %v", index+1, prompt, err)
		}
	}
}

func TestProceduralLanguageClauses(t *testing.T) {
	handler := newHandler(modelClient{})
	for _, prompt := range []string{
		"4/4 rock with no toms, snare on two and four",
		"7/8 fusion without floor toms",
		"No four on the floor in 4/4 funk",
		"Jazz with ride or shaker",
		"Dusty hip-hop at 90 BPM and lay the backbeat slightly late",
	} {
		request, _ := json.Marshal(map[string]string{"prompt": prompt, "engine": "procedural"})
		response := generateRequest(handler, string(request))
		if response.Code != 200 {
			t.Errorf("prompt %s got HTTP %d: %s", prompt, response.Code, response.Body)
			continue
		}
		score, err := parseScore(response.Body.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(prompt, "No four on the floor") {
			four := 0
			for _, event := range score.Events {
				if event.Instrument == "kick" && *event.Tick%96 == 0 {
					four++
				}
			}
			if four == score.Bars*4 {
				t.Error("returned an excluded four-on-the-floor kick")
			}
		}
		if (strings.Contains(prompt, "no toms") || strings.Contains(prompt, "without floor toms")) && slices.ContainsFunc(score.Events, func(event Event) bool {
			return strings.HasPrefix(event.Instrument, "tom_")
		}) {
			t.Error("returned excluded toms")
		}
	}
}

func TestGeneratedScoreMetadata(t *testing.T) {
	handler := newHandler(modelClient{})
	for _, tc := range []struct {
		prompt, expected string
	}{
		{"One bar of jazz waltz", "triplet"},
		{"One bar of dusty hip-hop", "Dilla-inspired"},
	} {
		body, _ := json.Marshal(map[string]string{"prompt": tc.prompt, "engine": "procedural"})
		response := generateRequest(handler, string(body))
		var score struct {
			Metadata struct {
				Description string `json:"description"`
				Feel        string `json:"feel"`
				Timing      string `json:"timing"`
			} `json:"metadata"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &score) != nil ||
			score.Metadata.Description == "" || score.Metadata.Feel == "" ||
			!strings.Contains(score.Metadata.Timing, tc.expected) {
			t.Errorf("%s missing useful metadata: %+v, HTTP %d", tc.prompt, score.Metadata, response.Code)
		}
	}
}

func TestScoreMetadataValidation(t *testing.T) {
	raw := strings.TrimSuffix(validScore, "}") + `,"metadata":{"description":"","feel":"swing"}}`
	if _, err := parseScore([]byte(raw)); err == nil {
		t.Fatal("accepted an empty beat description")
	}
	raw = strings.Replace(raw, `"description":""`, `"description":"A two-pulse blues loop."`, 1)
	if _, err := parseScore([]byte(raw)); err != nil {
		t.Fatalf("rejected score metadata: %v", err)
	}
}

func TestEmbeddedFormKitKit(t *testing.T) {
	handler := newHandler(modelClient{})
	get := func(path string) []byte {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned HTTP %d", path, response.Code)
		}
		return response.Body.Bytes()
	}
	for _, file := range []string{"formkit.js", "groove-worklet.js", "groove-clock.js", "theme.css", "BopkitMusic.woff"} {
		if len(get("/assets/formkit/"+file)) == 0 {
			t.Fatalf("empty FormKit asset: %s", file)
		}
	}
	var kit struct {
		Samples []struct{ File string }
	}
	if err := json.Unmarshal(get("/audio/manifest.json"), &kit); err != nil || len(kit.Samples) == 0 {
		t.Fatalf("missing FormKit sample manifest: %v", err)
	}
	for _, sample := range kit.Samples {
		body := get("/audio/" + sample.File)
		if len(body) <= 44 || string(body[:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
			t.Fatalf("missing WAV audio for %s", sample.File)
		}
	}
}

func TestProceduralFollowUps(t *testing.T) {
	handler := newHandler(modelClient{})
	base := generateRequest(handler, `{"engine":"procedural","prompt":"One bar of straight rock at 112 BPM"}`)
	if base.Code != 200 {
		t.Fatal(base.Body.String())
	}
	for _, tc := range []struct {
		prompt string
		bpm    int
		hits   map[string][]int
	}{
		{"Make it 200 bpm", 200, nil},
		{"Make sure there's a crash on the 3", 112, map[string][]int{"crash": {192}}},
		{"Make sure there's a bass drum on 1, 2, 3 and 4", 112, map[string][]int{"kick": {0, 96, 192, 288}}},
		{"Make it swung", 112, map[string][]int{"closed_hat": {64}}},
		{"Add a doublestroke on the downbeat", 112, map[string][]int{"snare": {0, 24}}},
	} {
		body, _ := json.Marshal(map[string]any{"engine": "procedural", "prompt": tc.prompt, "base_score": json.RawMessage(base.Body.Bytes())})
		result := generateRequest(handler, string(body))
		score, err := parseScore(result.Body.Bytes())
		if result.Code != 200 || err != nil || score.Style != "rock" || score.Tempo.BPM != tc.bpm {
			t.Errorf("%s: HTTP %d %s", tc.prompt, result.Code, result.Body)
			continue
		}
		for voice, ticks := range tc.hits {
			for _, tick := range ticks {
				if !slices.ContainsFunc(score.Events, func(event Event) bool {
					return event.Instrument == voice && *event.Tick == tick
				}) {
					t.Errorf("%s: missing %s at tick %d", tc.prompt, voice, tick)
				}
			}
		}
	}
}

func TestProceduralCompoundsAndNegation(t *testing.T) {
	handler := newHandler(modelClient{})
	rock := generateRequest(handler, `{"engine":"procedural","prompt":"One bar of straight rock at 112 BPM"}`)
	jazz := generateRequest(handler, `{"engine":"procedural","prompt":"One bar of small-group swing jazz with ride"}`)
	for _, tc := range []struct {
		prompt    string
		base      *httptest.ResponseRecorder
		required  map[string][]int
		forbidden map[string][]int
	}{
		{"Do not add a crash on beat three", rock, nil, map[string][]int{"crash": {-1}}},
		{"Avoid crash on beat 3", rock, nil, map[string][]int{"crash": {-1}}},
		{"Do not add any crash on 3", rock, nil, map[string][]int{"crash": {-1}}},
		{"Remove hi-hat on beat three", rock, map[string][]int{"closed_hat": {0}}, map[string][]int{"closed_hat": {192}}},
		{"Add an open hi-hat on three", rock, map[string][]int{"open_hat": {192}}, map[string][]int{"closed_hat": {192}}},
		{"Add a crash on the third beat", rock, map[string][]int{"crash": {192}}, nil},
		{"Do not swing the groove", rock, nil, map[string][]int{"closed_hat": {64}}},
		{"No double strokes on the downbeat", rock, nil, map[string][]int{"snare": {0, 24}}},
		{"Add a crash on beat three and remove closed hats", rock, map[string][]int{"crash": {192}}, map[string][]int{"closed_hat": {-1}}},
		{"Add a double stroke on beat two and remove the snare on beat two", rock, map[string][]int{"snare": {120}}, map[string][]int{"snare": {96}}},
		{"Add a crash on beat 3 and beat 4", rock, map[string][]int{"crash": {192, 288}}, nil},
		{"Make it lightly swung", jazz, map[string][]int{"ride": {152}}, map[string][]int{"ride": {160}}},
	} {
		t.Run(tc.prompt, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"engine": "procedural", "prompt": tc.prompt, "base_score": json.RawMessage(tc.base.Body.Bytes())})
			response := generateRequest(handler, string(body))
			score, err := parseScore(response.Body.Bytes())
			if response.Code != 200 || err != nil {
				t.Fatalf("HTTP %d %s", response.Code, response.Body)
			}
			has := func(voice string, tick int) bool {
				return slices.ContainsFunc(score.Events, func(event Event) bool {
					return event.Instrument == voice && (tick == -1 || *event.Tick == tick)
				})
			}
			for voice, ticks := range tc.required {
				for _, tick := range ticks {
					if !has(voice, tick) {
						t.Errorf("missing %s at %d", voice, tick)
					}
				}
			}
			for voice, ticks := range tc.forbidden {
				for _, tick := range ticks {
					if has(voice, tick) {
						t.Errorf("unexpected %s at %d", voice, tick)
					}
				}
			}
		})
	}
}

func TestProceduralComposesEditsOnNewGrooves(t *testing.T) {
	handler := newHandler(modelClient{})
	response := generateRequest(handler, `{"engine":"procedural","prompt":"Two bars of straight rock at 200 bpm, add a crash on beat 3 and a bass drum on 1, 2, 3 and 4, and make it swung"}`)
	score, err := parseScore(response.Body.Bytes())
	if response.Code != 200 || err != nil || score.Bars != 2 || score.Tempo.BPM != 200 {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body)
	}
	for bar := range score.Bars {
		for step := range 4 {
			tick := bar*384 + step*96
			if !slices.ContainsFunc(score.Events, func(event Event) bool {
				return event.Instrument == "kick" && *event.Tick == tick
			}) {
				t.Errorf("missing kick at %d", tick)
			}
		}
		tick := bar*384 + 192
		if !slices.ContainsFunc(score.Events, func(event Event) bool {
			return event.Instrument == "crash" && *event.Tick == tick
		}) {
			t.Errorf("missing crash at %d", tick)
		}
	}
	if score.Metadata == nil || score.Metadata.Feel != "Swung eighths" {
		t.Error("missing swing feel after compound edits")
	}
}

func TestProceduralExactHitCounts(t *testing.T) {
	handler := newHandler(modelClient{})
	base := generateRequest(handler, `{"engine":"procedural","prompt":"Two bars of straight rock at 112 BPM"}`)
	if base.Code != 200 {
		t.Fatal(base.Body.String())
	}
	type expectedCount struct {
		voice  string
		n      int
		perBar bool
	}
	for _, tc := range []struct {
		prompt string
		counts []expectedCount
		edit   bool
		bars   int
		crash  bool
	}{
		{prompt: "Give me a groove with 15 ghost notes", counts: []expectedCount{{"ghost", 15, false}}},
		{prompt: "Give me a groove with 20 bass hits", counts: []expectedCount{{"kick", 20, false}}},
		{prompt: "Four bars of funk at 110 BPM with fifteen snare ghost notes and twenty bass-drum hits", counts: []expectedCount{{"ghost", 15, false}, {"kick", 20, false}}, bars: 4},
		{prompt: "Two bars of straight rock with 6 kick hits in each bar and exactly 1 crash in the loop, with a crash on beat 3", counts: []expectedCount{{"kick", 6, true}, {"crash", 1, false}}},
		{prompt: "Set it to 3 ghost notes per bar and 7 snare hits total, with zero kicks", counts: []expectedCount{{"ghost", 3, true}, {"snare", 7, false}, {"kick", 0, false}}, edit: true, bars: 2},
		{prompt: "Use exactly 9 kicks and 2 crashes, with a crash on beat 3", counts: []expectedCount{{"kick", 9, false}, {"crash", 2, false}}, edit: true, bars: 2, crash: true},
	} {
		t.Run(tc.prompt, func(t *testing.T) {
			payload := map[string]any{"engine": "procedural", "prompt": tc.prompt}
			if tc.edit {
				payload["base_score"] = json.RawMessage(base.Body.Bytes())
			}
			body, _ := json.Marshal(payload)
			response := generateRequest(handler, string(body))
			if len(tc.counts) == 2 && tc.counts[1].voice == "crash" && tc.counts[1].n == 1 {
				if response.Code != 422 {
					t.Fatalf("a crash per bar conflicts with one crash total: %d %s", response.Code, response.Body)
				}
				return
			}
			if response.Code != 200 {
				t.Fatalf("HTTP %d: %s", response.Code, response.Body)
			}
			score, err := parseScore(response.Body.Bytes())
			if err != nil || tc.bars != 0 && score.Bars != tc.bars {
				t.Fatalf("invalid score or bar count: %v", err)
			}
			for _, want := range tc.counts {
				n := 1
				if want.perBar {
					n = score.Bars
				}
				found := make([]int, n)
				for _, event := range score.Events {
					if matchesVoice(event, want.voice) {
						found[*event.Tick/meters[score.Meter].ticks()%n]++
					}
				}
				for _, got := range found {
					if got != want.n {
						t.Errorf("got %v %s, want %d", found, want.voice, want.n)
						break
					}
				}
			}
			if tc.crash {
				for bar := range score.Bars {
					tick := bar*384 + 192
					if !slices.ContainsFunc(score.Events, func(e Event) bool { return *e.Tick == tick && e.Instrument == "crash" }) {
						t.Errorf("lost crash at tick %d", tick)
					}
				}
			}
		})
	}
	for _, prompt := range []string{
		"One bar of funk with 256 ghost notes",
		"One bar of funk with 3 kicks and 5 kick hits",
		"One bar of funk with 4 ghost notes and 3 snare hits",
		"One bar of rock with 5 kicks, no kick hits",
		"One bar of rock with -2 kick hits",
		"One bar of rock with 2.5 kick hits",
	} {
		body, _ := json.Marshal(map[string]string{"engine": "procedural", "prompt": prompt})
		response := generateRequest(handler, string(body))
		if response.Code != 422 {
			t.Errorf("%s: HTTP %d, want 422", prompt, response.Code)
		}
	}
}

func TestLocalModelRetriesWrongHitCount(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		content := validScore
		if calls.Add(1) > 1 {
			if !strings.Contains(request.Messages[1].Content, "expected 2 kick hits, got 1") {
				t.Errorf("missing count feedback: %s", request.Messages[1].Content)
			}
			content = strings.Replace(content, `"velocity":110}]`, `"velocity":110},{"tick":96,"instrument":"kick","velocity":90}]`, 1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]string{"content": content},
		}}})
	}))
	defer upstream.Close()
	handler := newHandler(modelClient{upstream.URL, "local-test", upstream.Client()})
	response := generateRequest(handler, `{"engine":"llm","prompt":"A blues groove with 2 bass hits"}`)
	if response.Code != 200 || calls.Load() != 2 {
		t.Fatalf("HTTP %d after %d calls: %s", response.Code, calls.Load(), response.Body)
	}
}

func TestCountedHitsRespectWiderPerBarLimit(t *testing.T) {
	base, err := parseScore([]byte(validScore))
	if err != nil {
		t.Fatal(err)
	}
	base.Style, base.Meter, base.Bars, base.Tempo = "rock", "4/4", 2, Tempo{112, "quarter"}
	base.Events = nil
	for tick := 0; tick < 768; tick += 48 {
		base.Events = append(base.Events, editedEvent(tick, "closed_hat", 70))
	}
	for _, tick := range []int{24, 72, 120} {
		base.Events = append(base.Events, editedEvent(tick, "snare", 40))
	}
	body, _ := json.Marshal(map[string]any{
		"engine": "procedural", "prompt": "3 ghost notes total and 2 snare hits per bar", "base_score": base,
	})
	response := generateRequest(newHandler(modelClient{}), string(body))
	if response.Code != 200 {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body)
	}
	score, err := parseScore(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		voice              string
		start, stop, count int
	}{
		{"snare", 0, 384, 2}, {"snare", 384, 768, 2}, {"ghost", 0, 768, 3},
	} {
		if got := countHits(score.Events, want.voice, want.start, want.stop); got != want.count {
			t.Errorf("got %d %s, want %d", got, want.voice, want.count)
		}
	}
}

func TestCountParserHandlesHundredsAndNegativeInstructions(t *testing.T) {
	for _, prompt := range []string{
		"Four bars of funk with anything but 5 kicks",
		"Two bars of blues. Do not add 4 kick hits.",
		"Two bars of blues. Don’t use exactly four kick hits.",
		"Four bars of funk with three hundred kicks",
		"A hip-hop groove with exactly 808 kicks",
	} {
		if _, _, err := parseHitCounts(prompt); err == nil {
			t.Errorf("parsed an exact count from %q", prompt)
		}
	}
	kitPrompt := "Give me a boom bap hip-hop groove with an 808 kick"
	if counts, clean, err := parseHitCounts(kitPrompt); len(counts) != 0 || clean != kitPrompt || err != nil {
		t.Errorf("treated the drum machine model as a count: %v, %q, %v", counts, clean, err)
	}
	counts, _, err := parseHitCounts("A loop with one hundred and twenty-three bass hits")
	if err != nil || !slices.Equal(counts, []hitCount{{voice: "kick", total: 123}}) {
		t.Errorf("wrong spelled count: %v, %v", counts, err)
	}
}

func TestCountedHitsPreserveExplicitPositionsAndMeter(t *testing.T) {
	handler := newHandler(modelClient{})
	for _, prompt := range []string{"Give me a 6/8 kick and rim groove", "Give me a 6 / 8 kick and rim groove"} {
		body, _ := json.Marshal(map[string]string{"engine": "procedural", "prompt": prompt})
		response := generateRequest(handler, string(body))
		score, err := parseScore(response.Body.Bytes())
		if response.Code != 200 || err != nil || score.Meter != "6/8" {
			t.Fatalf("lost meter in %q: HTTP %d %v", prompt, response.Code, err)
		}
	}
	base := generateRequest(handler, `{"engine":"procedural","prompt":"Two bars of straight rock at 112 BPM"}`)
	if base.Code != 200 {
		t.Fatal(base.Body.String())
	}
	body, _ := json.Marshal(map[string]any{
		"engine": "procedural", "base_score": json.RawMessage(base.Body.Bytes()),
		"prompt": "1 kick per bar on beat 3, 4 snare hits total, and 2 ghost notes total on beat 2, with a snare on beat 2",
	})
	response := generateRequest(handler, string(body))
	if response.Code != 200 {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body)
	}
	score, err := parseScore(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range score.Events {
		phase := *event.Tick % 384
		if event.Instrument == "kick" && phase != 192 || matchesVoice(event, "ghost") && phase != 96 {
			t.Errorf("misplaced event: phase %d, %+v", phase, event)
		}
	}
}

func TestCountedHitsFreeCapacityBeforeAdding(t *testing.T) {
	base := Score{
		SchemaVersion: "1.0", Style: "rock", Meter: "7/4", Bars: 4,
		TicksPerQuarter: 96, Tempo: Tempo{100, "quarter"},
	}
	for tick := 0; tick < 4*672; tick += 24 {
		base.Events = append(base.Events, editedEvent(tick, "closed_hat", 70), editedEvent(tick, "kick", 90))
		if tick/24 < 64 && tick/24%2 == 0 {
			base.Events = append(base.Events, editedEvent(tick, "snare", 100))
		}
	}
	if err := validateScore(base); err != nil || len(base.Events) != 256 {
		t.Fatalf("invalid test base: %v", err)
	}
	for _, prompt := range []string{"15 ghost notes and zero kicks", "zero kicks and 15 ghost notes"} {
		body, _ := json.Marshal(map[string]any{"engine": "procedural", "prompt": prompt, "base_score": base})
		response := generateRequest(newHandler(modelClient{}), string(body))
		if response.Code != 200 {
			t.Fatalf("%q: HTTP %d: %s", prompt, response.Code, response.Body)
		}
		score, err := parseScore(response.Body.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if countHits(score.Events, "ghost", 0, 4*672) != 15 || countHits(score.Events, "kick", 0, 4*672) != 0 {
			t.Fatalf("did not honor %q", prompt)
		}
	}
}

func TestProceduralMatchesBaiao(t *testing.T) {
	handler := newHandler(modelClient{})
	for _, name := range []string{"baiao", "baião"} {
		body, _ := json.Marshal(map[string]string{"engine": "procedural", "prompt": "Make a Brazilian " + name + " drum groove"})
		response := generateRequest(handler, string(body))
		score, err := parseScore(response.Body.Bytes())
		if response.Code != http.StatusOK || err != nil || score.Style != "latin" ||
			score.Metadata == nil || !strings.Contains(score.Metadata.Description, "baião") {
			t.Errorf("wrong groove for %s: %s", name, response.Body)
		}
	}
}
