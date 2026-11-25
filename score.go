package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"sort"
)

type Tempo struct {
	BPM      int    `json:"bpm"`
	BeatUnit string `json:"beat_unit"`
}

type Event struct {
	Tick       *int   `json:"tick"`
	Instrument string `json:"instrument"`
	Velocity   int    `json:"velocity"`
}

type Score struct {
	SchemaVersion   string  `json:"schema_version"`
	Style           string  `json:"style"`
	Meter           string  `json:"meter"`
	Tempo           Tempo   `json:"tempo"`
	Bars            int     `json:"bars"`
	TicksPerQuarter int     `json:"ticks_per_quarter"`
	Events          []Event `json:"events"`
}

func parseScore(raw []byte) (Score, error) {
	var score Score
	if err := json.Unmarshal(raw, &score, json.RejectUnknownMembers(true)); err != nil {
		return Score{}, errors.New("score must be one JSON object with the required fields")
	}
	if err := validateScore(score); err != nil {
		return Score{}, err
	}
	sort.Slice(score.Events, func(i, j int) bool {
		a, b := score.Events[i], score.Events[j]
		if *a.Tick == *b.Tick {
			return a.Instrument < b.Instrument
		}
		return *a.Tick < *b.Tick
	})
	return score, nil
}

var allowedInstruments = map[string]bool{
	"kick": true, "snare": true,
	"closed_hat": true, "open_hat": true,
	"ride": true, "crash": true,
	"tom_high": true, "tom_mid": true,
	"tom_low": true, "rim": true,
}

func validateScore(score Score) error {
	if score.SchemaVersion != "1.0" || score.TicksPerQuarter != 96 {
		return errors.New("use schema_version 1.0 and ticks_per_quarter 96")
	}

	switch score.Style {
	case "rock", "funk", "blues", "jazz":
	default:
		return errors.New("style must be rock, funk, blues, or jazz")
	}

	ticksPerBar := map[string]int{
		"4/4": 384, "6/8": 288,
		"5/4": 480, "7/4": 672,
	}[score.Meter]

	if ticksPerBar == 0 {
		return errors.New("meter must be 4/4, 6/8, 5/4, or 7/4")
	}

	if score.Bars < 1 || score.Bars > 4 {
		return errors.New("bars must be 1-4")
	}

	if score.Tempo.BPM < 40 || score.Tempo.BPM > 220 {
		return errors.New("BPM must be 40-220")
	}

	if score.Tempo.BeatUnit != "quarter" &&
		score.Tempo.BeatUnit != "dotted_quarter" {
		return errors.New("tempo beat_unit must be quarter or dotted_quarter")
	}

	if len(score.Events) < 1 || len(score.Events) > 256 {
		return errors.New("events must contain 1-256 notes")
	}

	type hit struct {
		tick       int
		instrument string
	}
	seen := make(map[hit]bool)
	hats := make(map[int]string)

	for _, e := range score.Events {
		if e.Tick == nil {
			return errors.New("every event requires an integer tick")
		}
		tick := *e.Tick
		if tick < 0 || tick >= ticksPerBar*score.Bars {
			return fmt.Errorf("event tick %d outside score; valid ticks are 0 through %d", tick, ticksPerBar*score.Bars-1)
		}

		if !allowedInstruments[e.Instrument] {
			return fmt.Errorf("event at tick %d uses an unknown instrument", tick)
		}

		if e.Velocity < 1 || e.Velocity > 127 {
			return fmt.Errorf("invalid velocity %d", e.Velocity)
		}

		key := hit{tick, e.Instrument}
		if seen[key] {
			return fmt.Errorf("duplicate %s at tick %d", e.Instrument, tick)
		}
		seen[key] = true

		if e.Instrument == "open_hat" || e.Instrument == "closed_hat" {
			if previous, ok := hats[tick]; ok && previous != e.Instrument {
				return fmt.Errorf("hi-hat conflict at tick %d", tick)
			}
			hats[tick] = e.Instrument
		}
	}

	return nil
}
