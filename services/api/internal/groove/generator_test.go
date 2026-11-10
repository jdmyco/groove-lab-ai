package groove

import (
	"reflect"
	"testing"
)

func testRequest() Request {
	return Request{
		Style: "rock",
		Meter: "4/4",
		Tempo: Tempo{BPM: 100, BeatUnit: "quarter"},
		Bars:  1,
	}
}

func TestGenerate(t *testing.T) {
	request := testRequest()

	score, err := Generate(request)
	if err != nil {
		t.Fatal(err)
	}

	if err := Validate(score); err != nil {
		t.Fatal(err)
	}

	if got := len(score.Events); got != 12 {
		t.Fatalf("got %d events, want 12", got)
	}

	// Identical input should produce identical output.
	again, err := Generate(request)
	if err != nil || !reflect.DeepEqual(score, again) {
		t.Fatalf("generator is not deterministic: %v", err)
	}

	// Two bars should produce twice as many events.
	request.Bars = 2
	two, err := Generate(request)
	if err != nil {
		t.Fatal(err)
	}

	if len(two.Events) != 24 {
		t.Fatalf("got %d events, want 24", len(two.Events))
	}

	if err := Validate(two); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Score)
	}{
		{"tick past bar", func(s *Score) {
			s.Events[0].Tick = 384
		}},
		{"velocity zero", func(s *Score) {
			s.Events[0].Velocity = 0
		}},
		{"unknown instrument", func(s *Score) {
			s.Events[0].Instrument = "piano"
		}},
		{"duplicate hit", func(s *Score) {
			s.Events = append(s.Events, s.Events[0])
		}},
		{"hi-hat conflict", func(s *Score) {
			s.Events = append(s.Events, Event{
				Tick: 0, Instrument: "open_hat", Velocity: 80,
			})
		}},
		{"empty score", func(s *Score) {
			s.Events = nil
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := Generate(testRequest())
			if err != nil {
				t.Fatal(err)
			}

			tt.modify(&score)

			if err := Validate(score); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
