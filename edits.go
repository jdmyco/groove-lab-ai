package main

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type hitEdit struct {
	voice                    string
	beats                    []int
	remove, double, preserve bool
}

type grooveEdits struct {
	hits  []hitEdit
	swing int
}

var countedBeat = `(?:downbeats?|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|first|second|third|fourth|fifth|sixth|seventh|eighth|ninth|tenth|eleventh|twelfth|[0-9]+(?:st|nd|rd|th)?)`
var beatList = `(\b` + countedBeat + `\b(?:\s*(?:,\s*(?:and\s+)?|\band\s+)(?:the\s+)?(?:beats?\s+)?\b` + countedBeat + `\b)*)`

const negative = `do not|don't|never|avoid|omit|skip|remove|drop|no|without`
const negativeLead = `(?:(` + negative + `)\s+(?:(?:add|put|place|use)\s+)?(?:(?:the|a|an|any)\s+)?)?`

var drumPosition = regexp.MustCompile(`\b` + negativeLead + `(` + drumVoiceNames() + `)\s+(?:hits?\s+)?on\s+(?:the\s+)?(?:beats?\s+)?` + beatList)
var strokePosition = regexp.MustCompile(`\b` + negativeLead + `(?:(` + drumVoiceNames() + `)\s+)?double[- ]?strokes?(?:\s+on\s+(?:the\s+)?(?:beats?\s+)?` + beatList + `)?`)
var beatToken = regexp.MustCompile(`\b` + countedBeat + `\b`)
var swingCommand = regexp.MustCompile(`\b` + negativeLead + `(?:make\s+(?:it|this|the groove)\s+(?:(?:more|heavily|lightly)\s+)?swung|swing\s+(?:it|this|the groove)|add\s+(?:(?:a little|some|heavy|light)\s+)?swing)\b`)

func drumVoiceNames() string {
	var names []string
	for _, group := range voiceTerms {
		for _, name := range group[1:] {
			names = append(names, strings.ReplaceAll(regexp.QuoteMeta(name), " ", "[- ]+"))
		}
	}
	return strings.Join(names, "|")
}

func parseEdits(prompt string) (grooveEdits, string) {
	var edits grooveEdits
	prompt = strings.ReplaceAll(strings.ToLower(prompt), "’", "'")
	type positioned struct {
		at   int
		edit hitEdit
	}
	var placed []positioned
	clean := []byte(prompt)
	for _, re := range []*regexp.Regexp{drumPosition, strokePosition} {
		for _, span := range re.FindAllStringIndex(prompt, -1) {
			match := re.FindStringSubmatch(prompt[span[0]:span[1]])
			voice := "snare"
			name := strings.Join(strings.Fields(strings.ReplaceAll(match[2], "-", " ")), " ")
			for _, group := range voiceTerms {
				if slices.Contains(group[1:], name) {
					voice = group[0]
					break
				}
			}
			placed = append(placed, positioned{span[0], hitEdit{
				voice: voice, beats: editBeats(match[3]), remove: match[1] != "", double: re == strokePosition,
				preserve: match[1] != "" && phraseIn(normalized(match[0]), match[1]+" add"),
			}})
			for i := span[0]; i < span[1]; i++ {
				clean[i] = ' '
			}
		}
	}
	slices.SortStableFunc(placed, func(a, b positioned) int { return cmp.Compare(a.at, b.at) })
	for _, found := range placed {
		edits.hits = append(edits.hits, found.edit)
	}
	prompt = string(clean)
	for _, match := range swingCommand.FindAllStringSubmatch(prompt, -1) {
		edits.swing = 64
		switch {
		case match[1] != "":
			edits.swing = 48
		case strings.Contains(match[0], "heavy") || strings.Contains(match[0], "heavily") || strings.Contains(match[0], "more"):
			edits.swing = 68
		case strings.Contains(match[0], "light"):
			edits.swing = 56
		}
	}
	return edits, swingCommand.ReplaceAllString(prompt, " ")
}

func editBeats(text string) []int {
	words := []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
	ordinals := []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth"}
	var beats []int
	for _, token := range beatToken.FindAllString(text, -1) {
		n, err := strconv.Atoi(strings.TrimRight(token, "stndrh"))
		switch {
		case strings.HasPrefix(token, "downbeat"):
			n = 1
		case err != nil:
			n = max(slices.Index(words, token), slices.Index(ordinals, token))
		}
		if !slices.Contains(beats, n) {
			beats = append(beats, n)
		}
	}
	if len(beats) == 0 {
		return []int{1}
	}
	return beats
}

func editedEvent(tick int, voice string, velocity int) Event {
	return Event{Tick: &tick, Instrument: voice, Velocity: velocity}
}

// Reserve requested onsets before restoring the original timekeeping and ornaments.
func editGroove(score Score, brief grooveBrief, edits grooveEdits) (Score, error) {
	meter := meters[score.Meter]
	if brief.style != "" && brief.style != score.Style || brief.meter != "" && brief.meter != score.Meter ||
		brief.bars != 0 && brief.bars != score.Bars || len(brief.grouping) > 0 && !slices.Equal(brief.grouping, score.Grouping) {
		return Score{}, errors.New("start a new groove to change the selected groove's style, meter, grouping, or bar count")
	}
	if brief.fourOnFloor {
		if score.Meter != "4/4" {
			return Score{}, errors.New("four-on-the-floor requires a 4/4 groove")
		}
		edits.hits = append(edits.hits, hitEdit{voice: "kick", beats: []int{1, 2, 3, 4}})
	}
	if brief.bpm == 0 && len(edits.hits) == 0 && edits.swing == 0 && len(brief.excluded) == 0 {
		return Score{}, errors.New("describe a tempo, swing change, or instrument hit on a beat to edit the selected groove")
	}
	score.Tempo = retime(brief, score.Tempo)
	original := slices.Clone(score.Events)
	var notes []string
	var required []Event
	for _, voice := range brief.excluded {
		original = slices.DeleteFunc(original, func(event Event) bool { return matchesVoice(event, voice) })
		notes = append(notes, "Removed "+strings.ReplaceAll(voice, "_", " "))
	}
	if edits.swing != 0 {
		if meter.denominator != 4 {
			return Score{}, errors.New("the procedural swing edit currently uses quarter-note meters")
		}
		from := offbeatGrid(original)
		for i, event := range original {
			tick := *event.Tick
			switch tick % 96 {
			case from / 2, 48 + from/2:
				tick += (edits.swing - from) / 2
			case from:
				tick += edits.swing - from
			}
			original[i].Tick = &tick
		}
		notes = append(notes, fmt.Sprintf("Eighth offbeats at tick %d of each quarter", edits.swing))
		if edits.swing != 48 && !slices.ContainsFunc(original, func(event Event) bool { return *event.Tick%96 == edits.swing }) {
			voice := "closed_hat"
			for _, event := range original {
				switch event.Instrument {
				case "closed_hat", "ride", "shaker", "cowbell":
					voice = event.Instrument
				}
			}
			if slices.Contains(brief.excluded, voice) || slices.Contains(brief.excluded, "hat") && voice == "closed_hat" {
				return Score{}, errors.New("an offbeat timekeeper is needed to swing this groove")
			}
			for bar := range score.Bars {
				for tick := edits.swing; tick < meter.ticks(); tick += 96 {
					required = append(required, editedEvent(bar*meter.ticks()+tick, voice, 58))
				}
			}
		}
	}
	for _, edit := range edits.hits {
		if edit.preserve {
			continue
		}
		steps := make([]string, 0, len(edit.beats))
		for _, beat := range edit.beats {
			if beat < 1 || beat > meter.numerator {
				return Score{}, fmt.Errorf("beat %d is outside %s", beat, score.Meter)
			}
			steps = append(steps, strconv.Itoa(beat))
			for bar := range score.Bars {
				tick := bar*meter.ticks() + (beat-1)*384/meter.denominator
				spacing := 24
				if edits.swing != 0 {
					spacing = edits.swing / 2
				} else {
					spacing = offbeatGrid(original) / 2
				}
				if edit.remove {
					if edit.double {
						tick += spacing
					}
					clear := func(e Event) bool { return *e.Tick == tick && matchesVoice(e, edit.voice) }
					original = slices.DeleteFunc(original, clear)
					required = slices.DeleteFunc(required, clear)
					continue
				}
				voice, velocity := edit.voice, 106
				switch voice {
				case "hat":
					voice, velocity = "closed_hat", 72
				case "tom":
					voice = "tom_low"
				case "ghost":
					voice, velocity = "snare", 42
				}
				for _, e := range original {
					if *e.Tick == tick && matchesVoice(e, edit.voice) {
						voice, velocity = e.Instrument, e.Velocity
						break
					}
				}
				required = append(required, editedEvent(tick, voice, velocity))
				if edit.double {
					required = append(required, editedEvent(tick+spacing, voice, max(1, velocity*3/4)))
				}
			}
		}
		verb := "Placed"
		if edit.remove {
			verb = "Removed"
		}
		voice := strings.ReplaceAll(edit.voice, "_", " ")
		if edit.double {
			voice += " double stroke"
		}
		notes = append(notes, fmt.Sprintf("%s %s on beat %s", verb, voice, strings.Join(steps, ", ")))
	}
	key := func(event Event) string { return fmt.Sprintf("%d/%s", *event.Tick, event.Instrument) }
	occupied := make(map[string]bool)
	score.Events = make([]Event, 0, min(256, len(original)+len(required)))
	for _, event := range required {
		if !occupied[key(event)] {
			score.Events = append(score.Events, event)
			occupied[key(event)] = true
		}
	}
	if len(required) > 0 {
		if err := validateScore(score); err != nil {
			return Score{}, fmt.Errorf("the requested hits are not playable together: %w", err)
		}
	}
	// Keep feet and stronger anchor hits before light ornamentation when a
	// crash or faster tempo conflicts with an existing hand part.
	slices.SortStableFunc(original, func(a, b Event) int {
		priority := func(e Event) int {
			if e.Instrument == "kick" || e.Instrument == "foot_hat" {
				return 128 + e.Velocity
			}
			return e.Velocity
		}
		return cmp.Compare(priority(b), priority(a))
	})
	trimmed := 0
	for _, event := range original {
		if occupied[key(event)] {
			continue
		}
		score.Events = append(score.Events, event)
		if err := validateScore(score); err != nil {
			score.Events = score.Events[:len(score.Events)-1]
			trimmed++
		} else {
			occupied[key(event)] = true
		}
	}
	slices.SortFunc(score.Events, func(a, b Event) int {
		if order := cmp.Compare(*a.Tick, *b.Tick); order != 0 {
			return order
		}
		return strings.Compare(a.Instrument, b.Instrument)
	})
	if err := validateScore(score); err != nil {
		return Score{}, err
	}
	for _, options := range brief.required {
		if !slices.ContainsFunc(score.Events, func(event Event) bool {
			return slices.ContainsFunc(options, func(voice string) bool { return matchesVoice(event, voice) })
		}) {
			return Score{}, fmt.Errorf("specify a beat for %s", strings.Join(options, " or "))
		}
	}
	for _, voice := range brief.excluded {
		if slices.ContainsFunc(score.Events, func(event Event) bool { return matchesVoice(event, voice) }) {
			return Score{}, errors.New("an instrument cannot be both added and excluded")
		}
	}
	if len(notes) > 0 || trimmed > 0 {
		score.Metadata = nil
		describeScore(&score)
		if len(notes) > 0 {
			score.Metadata.Description = strings.Join(notes, ". ") + "."
		}
		if trimmed > 0 {
			score.Metadata.Description += fmt.Sprintf(" Simplified %d conflicting hits to keep the groove playable.", trimmed)
		}
		if len([]rune(score.Metadata.Description)) > 360 {
			return Score{}, errors.New("split those edits into smaller steps")
		}
		if edits.swing != 0 {
			score.Metadata.Feel = "Swung eighths"
			if edits.swing == 48 {
				score.Metadata.Feel = "Straight eighths"
			}
			score.Metadata.Timing = fmt.Sprintf("The eighth-note offbeat lands %d ticks after the quarter-note beat.", edits.swing)
		}
	}
	return score, nil
}

// Find the dominant written eighth subdivision in the timekeeping part.
func offbeatGrid(events []Event) int {
	var counts [96]int
	for _, event := range events {
		switch event.Instrument {
		case "closed_hat", "open_hat", "ride", "cowbell", "shaker", "clave":
			counts[*event.Tick%96]++
		}
	}
	grid := 48
	for _, tick := range []int{56, 64, 68} {
		if counts[tick] > counts[grid] {
			grid = tick
		}
	}
	return grid
}

func matchesVoice(event Event, voice string) bool {
	switch voice {
	case "ghost":
		return event.Instrument == "snare" && event.Velocity <= 55
	case "hat":
		return strings.HasSuffix(event.Instrument, "_hat")
	case "tom":
		return strings.HasPrefix(event.Instrument, "tom_")
	default:
		return event.Instrument == voice
	}
}
