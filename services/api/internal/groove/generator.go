package groove

import (
	"fmt"
	"sort"
)

func Generate(r Request) (Score, error) {
	if r.Style != "rock" || r.Meter != "4/4" {
		return Score{}, fmt.Errorf("only rock in 4/4 is supported at this time")
	}

	if r.Bars < 1 || r.Bars > 2 {
		return Score{}, fmt.Errorf("bars must be 1 or 2")
	}

	if r.Tempo.BPM < 40 || r.Tempo.BPM > 220 ||
		r.Tempo.BeatUnit != "quarter" {
		return Score{}, fmt.Errorf("invalid tempo")
	}

	events := make([]Event, 0, 12*r.Bars)

	for bar := 0; bar < r.Bars; bar++ {
		offset := bar * 384

		// Kick on beats 1 and 3.
		for _, tick := range []int{0, 192} {
			events = append(events, Event{
				Tick: offset + tick, Instrument: "kick", Velocity: 110,
			})
		}

		// Snare on beats 2 and 4.
		for _, tick := range []int{96, 288} {
			events = append(events, Event{
				Tick: offset + tick, Instrument: "snare", Velocity: 112,
			})
		}

		// Closed hi-hat every eighth note.
		for tick := 0; tick < 384; tick += 48 {
			velocity := 65
			if tick%96 == 0 {
				velocity = 73
			}

			events = append(events, Event{
				Tick:       offset + tick,
				Instrument: "closed_hat",
				Velocity:   velocity,
			})
		}
	}

	// Sort by tick, then instrument, for reproducible output.
	sort.Slice(events, func(i, j int) bool {
		if events[i].Tick == events[j].Tick {
			return events[i].Instrument < events[j].Instrument
		}
		return events[i].Tick < events[j].Tick
	})

	return Score{
		SchemaVersion:   "1.0",
		Style:           r.Style,
		Meter:           r.Meter,
		Tempo:           r.Tempo,
		Bars:            r.Bars,
		TicksPerQuarter: 96,
		Events:          events,
	}, nil
}
