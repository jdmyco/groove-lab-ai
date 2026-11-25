package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
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
	Offset     int    `json:"offset,omitzero"`
}

type Score struct {
	SchemaVersion   string  `json:"schema_version"`
	Style           string  `json:"style"`
	Meter           string  `json:"meter"`
	Grouping        []int   `json:"grouping,omitempty"`
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

var styles = []string{"rock", "funk", "blues", "jazz", "latin", "soul", "disco", "reggae", "hip_hop", "fusion"}

type meterSpec struct{ numerator, denominator int }

var meters = map[string]meterSpec{
	"3/4": {3, 4}, "4/4": {4, 4}, "5/4": {5, 4}, "7/4": {7, 4},
	"6/8": {6, 8}, "7/8": {7, 8}, "12/8": {12, 8},
}

func (m meterSpec) ticks() int { return m.numerator * 384 / m.denominator }

var allowedInstruments = map[string]bool{
	"kick": true, "snare": true, "closed_hat": true, "open_hat": true,
	"ride": true, "crash": true, "tom_high": true, "tom_mid": true,
	"tom_low": true, "rim": true, "foot_hat": true, "cowbell": true,
	"clave": true, "shaker": true,
}

func validateScore(score Score) error {
	if score.SchemaVersion != "1.0" || score.TicksPerQuarter != 96 {
		return errors.New("use schema_version 1.0 and ticks_per_quarter 96")
	}
	if !slices.Contains(styles, score.Style) {
		return errors.New("style must be rock, funk, blues, jazz, latin, soul, disco, reggae, hip_hop, or fusion")
	}
	meter, ok := meters[score.Meter]
	if !ok {
		return errors.New("meter must be 3/4, 4/4, 5/4, 7/4, 6/8, 7/8, or 12/8")
	}
	if score.Grouping != nil {
		sum := 0
		for _, group := range score.Grouping {
			if group < 1 || group > meter.numerator {
				return errors.New("grouping counts must be positive and fit the meter")
			}
			sum += group
		}
		if len(score.Grouping) < 2 || sum != meter.numerator {
			return errors.New("grouping must have at least two counts that add up to the meter numerator")
		}
	}
	ticksPerBar := meter.ticks()

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
	type strokes struct {
		hands, keepers   int
		openHat, footHat bool
	}
	total := ticksPerBar * score.Bars
	written := make(map[hit]bool)
	sounding := make(map[hit]bool)
	at := make(map[int]strokes)
	for _, event := range score.Events {
		if event.Tick == nil {
			return errors.New("every event requires an integer tick")
		}
		tick := *event.Tick
		if tick < 0 || tick >= total {
			return fmt.Errorf("event tick %d outside score; valid ticks are 0 through %d", tick, total-1)
		}
		actual := tick + event.Offset
		if event.Offset < -6 || event.Offset > 6 || actual < 0 || actual >= total {
			return fmt.Errorf("offset at tick %d must be within ±6 and keep the hit inside the loop", tick)
		}
		if !allowedInstruments[event.Instrument] {
			return fmt.Errorf("event at tick %d uses an unknown instrument", tick)
		}
		if event.Velocity < 1 || event.Velocity > 127 {
			return fmt.Errorf("invalid velocity %d", event.Velocity)
		}
		notated := hit{tick, event.Instrument}
		sounded := hit{actual, event.Instrument}
		if written[notated] || sounding[sounded] {
			return fmt.Errorf("duplicate %s at tick %d", event.Instrument, tick)
		}
		written[notated], sounding[sounded] = true, true

		counts := at[actual]
		switch event.Instrument {
		case "kick":
		case "foot_hat":
			counts.footHat = true
		case "closed_hat", "open_hat", "ride", "crash", "cowbell", "clave", "shaker":
			counts.keepers++
			counts.hands++
			counts.openHat = counts.openHat || event.Instrument == "open_hat"
		default:
			counts.hands++
		}
		if counts.hands > 2 || counts.keepers > 1 || counts.openHat && counts.footHat {
			return fmt.Errorf("conflicting drum strokes at sounding tick %d", actual)
		}
		at[actual] = counts
	}
	if !tempoAllowsHands(score.Tempo, minimumHandGap(score.Events, total)) {
		return errors.New("the hand strokes are too close together at this tempo")
	}
	return nil
}

// Check a second pass so the measured gap includes the loop boundary.
func minimumHandGap(events []Event, loopTicks int) int {
	strokes := make([]int, 0, len(events))
	for _, event := range events {
		if event.Instrument != "kick" && event.Instrument != "foot_hat" {
			strokes = append(strokes, *event.Tick+event.Offset)
		}
	}
	slices.Sort(strokes)
	older, newer := -loopTicks, -loopTicks
	gap := loopTicks
	for cycle := 0; cycle < 2; cycle++ {
		for _, stroke := range strokes {
			now := stroke + cycle*loopTicks
			gap = min(gap, now-older)
			older, newer = newer, now
		}
	}
	return gap
}

func tempoAllowsHands(tempo Tempo, tickGap int) bool {
	beat := 96
	if tempo.BeatUnit == "dotted_quarter" {
		beat = 144
	}
	return tickGap*60000 >= 48*tempo.BPM*beat
}
