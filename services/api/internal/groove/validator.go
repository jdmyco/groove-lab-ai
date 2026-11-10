package groove

import "fmt"

var allowedInstruments = map[string]bool{
	"kick": true, "snare": true,
	"closed_hat": true, "open_hat": true,
	"ride": true, "crash": true,
	"tom_high": true, "tom_mid": true,
	"tom_low": true, "rim": true,
}

func Validate(score Score) error {
	if score.SchemaVersion != "1.0" || score.TicksPerQuarter != 96 {
		return fmt.Errorf("unsupported score version or tick resolution")
	}

	switch score.Style {
	case "rock", "funk", "blues", "jazz":
	default:
		return fmt.Errorf("unsupported style %q", score.Style)
	}

	ticksPerBar := map[string]int{
		"4/4": 384, "6/8": 288,
		"5/4": 480, "7/4": 672,
	}[score.Meter]

	if ticksPerBar == 0 {
		return fmt.Errorf("unsupported meter %q", score.Meter)
	}

	if score.Bars < 1 || score.Bars > 4 {
		return fmt.Errorf("bars must be 1-4")
	}

	if score.Tempo.BPM < 40 || score.Tempo.BPM > 220 {
		return fmt.Errorf("BPM must be 40-220")
	}

	if score.Tempo.BeatUnit != "quarter" &&
		score.Tempo.BeatUnit != "dotted_quarter" {
		return fmt.Errorf("invalid beat unit")
	}

	if len(score.Events) < 1 || len(score.Events) > 256 {
		return fmt.Errorf("events must contain 1-256 notes")
	}

	seen := make(map[string]bool)
	hats := make(map[int]string)

	for _, e := range score.Events {
		if e.Tick < 0 || e.Tick >= ticksPerBar*score.Bars {
			return fmt.Errorf("event tick %d outside score", e.Tick)
		}

		if !allowedInstruments[e.Instrument] {
			return fmt.Errorf("unknown instrument %q", e.Instrument)
		}

		if e.Velocity < 1 || e.Velocity > 127 {
			return fmt.Errorf("invalid velocity %d", e.Velocity)
		}

		key := fmt.Sprintf("%d/%s", e.Tick, e.Instrument)
		if seen[key] {
			return fmt.Errorf("duplicate %s at tick %d", e.Instrument, e.Tick)
		}
		seen[key] = true

		if e.Instrument == "open_hat" || e.Instrument == "closed_hat" {
			if previous, ok := hats[e.Tick]; ok && previous != e.Instrument {
				return fmt.Errorf("hi-hat conflict at tick %d", e.Tick)
			}
			hats[e.Tick] = e.Instrument
		}
	}

	return nil
}
