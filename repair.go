package main

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"slices"
	"strings"
)

// Try the canonical parser first so clean scores are returned as authored.
func parseGeneratedScore(raw []byte) (Score, string, error) {
	score, strictErr := parseScore(raw)
	if strictErr == nil {
		return score, "", nil
	}

	var model struct {
		Score
		Events []jsontext.Value `json:"events"`
	}
	if err := json.Unmarshal(raw, &model); err != nil {
		return Score{}, "", strictErr
	}
	meter, ok := meters[model.Meter]
	if !ok || model.Bars < 1 || model.Bars > 4 || len(model.Events) == 0 {
		return Score{}, "", strictErr
	}
	length := model.Bars * meter.ticks()
	if len(model.Grouping) == 0 {
		model.Grouping = nil
	}

	type hit struct {
		Event
		adjusted, atEnd bool
	}
	hits := make([]hit, 0, len(model.Events))
	for _, rawEvent := range model.Events {
		var event struct {
			Tick       *float64 `json:"tick"`
			Instrument string   `json:"instrument"`
			Velocity   *float64 `json:"velocity"`
			Offset     float64  `json:"offset"`
		}
		if err := json.Unmarshal(rawEvent, &event); err != nil || event.Tick == nil {
			continue
		}
		voice := canonicalInstrument(event.Instrument)
		if voice == "" {
			continue
		}
		rounded := math.Round(*event.Tick)
		if rounded < -1 || rounded > float64(length) {
			continue
		}
		tick := min(length-1, max(0, int(rounded)))
		offset := 0
		roundedOffset := math.Round(event.Offset)
		if roundedOffset >= -6 && roundedOffset <= 6 {
			offset = min(length-1-tick, max(-tick, int(roundedOffset)))
		}
		velocity := 80
		if event.Velocity != nil {
			velocity = int(min(127, max(1, math.Round(*event.Velocity))))
		}
		adjusted := float64(tick) != *event.Tick || voice != event.Instrument ||
			float64(offset) != event.Offset || event.Velocity == nil || float64(velocity) != *event.Velocity
		hits = append(hits, hit{Event{&tick, voice, velocity, offset}, adjusted, rounded == float64(length) && event.Offset >= 0})
	}
	type onset struct {
		tick  int
		voice string
	}
	written, sounding := make(map[onset]bool), make(map[onset]bool)
	// Prefer untouched hits and settle the downbeat before the loop edge.
	priority := func(h hit) int {
		switch {
		case h.atEnd:
			return 2
		case h.adjusted:
			return 1
		default:
			return 0
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int { return cmp.Compare(priority(a), priority(b)) })
	adjusted := 0
	for _, hit := range hits {
		if hit.atEnd && sounding[onset{0, hit.Instrument}] {
			continue
		}
		note := onset{*hit.Tick, hit.Instrument}
		sound := onset{*hit.Tick + hit.Offset, hit.Instrument}
		if written[note] || sounding[sound] {
			continue
		}
		model.Score.Events = append(model.Score.Events, hit.Event)
		if hit.adjusted {
			if err := validateScore(model.Score); err != nil {
				model.Score.Events = model.Score.Events[:len(model.Score.Events)-1]
				continue
			}
			adjusted++
		}
		written[note], sounding[sound] = true, true
	}
	if err := validateScore(model.Score); err != nil {
		return Score{}, "", err
	}
	sortScoreEvents(model.Score.Events)

	removed := len(model.Events) - len(model.Score.Events)
	var details []string
	if adjusted > 0 {
		details = append(details, fmt.Sprintf("%d adjusted", adjusted))
	}
	if removed > 0 {
		details = append(details, fmt.Sprintf("%d removed", removed))
	}
	if len(details) == 0 {
		return model.Score, "Cleaned up score fields", nil
	}
	return model.Score, "Hits " + strings.Join(details, ", "), nil
}

var modelVoiceAliases = map[string]string{
	"hi_hat": "closed_hat", "hihat": "closed_hat", "closed_hi_hat": "closed_hat",
	"closed_hihat": "closed_hat", "open_hi_hat": "open_hat", "open_hihat": "open_hat",
	"pedal_hi_hat": "foot_hat", "bass_drum": "kick", "kick_drum": "kick", "snare_drum": "snare",
	"ride_cymbal": "ride", "crash_cymbal": "crash", "floor_tom": "tom_low",
	"high_tom": "tom_high", "mid_tom": "tom_mid", "low_tom": "tom_low", "rim_click": "rim",
	"cross_stick": "rim", "side_stick": "rim",
}

func canonicalInstrument(value string) string {
	voice := strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	}), "_")
	if allowedInstruments[voice] {
		return voice
	}
	return modelVoiceAliases[voice]
}
