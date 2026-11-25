package main

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type grooveTags struct {
	Family     string   `json:"family"`
	Feel       string   `json:"feel"`
	Color      string   `json:"color"`
	Influences []string `json:"influences"`
	CycleBars  int      `json:"cycle_bars"`
}

type catalogRecord struct {
	Messages []message  `json:"messages"`
	Groove   grooveTags `json:"groove"`
}

type catalogGroove struct {
	score          Score
	tags           grooveTags
	family, search string
	voices         map[string]bool
	fourOnFloor    bool
	handGap        int
}

var loadCatalog = sync.OnceValues(func() ([]catalogGroove, error) {
	catalog := make([]catalogGroove, 0, bytes.Count(corpus, []byte{'\n'}))
	for index, line := range bytes.Split(bytes.TrimSpace(corpus), []byte("\n")) {
		var record catalogRecord
		if err := json.Unmarshal(line, &record, json.RejectUnknownMembers(true)); err != nil {
			return nil, fmt.Errorf("catalog row %d: %w", index+1, err)
		}
		if len(record.Messages) != 3 || record.Groove.Family == "" || record.Groove.CycleBars < 1 {
			return nil, fmt.Errorf("catalog row %d is missing messages or groove metadata", index+1)
		}
		score, err := parseScore([]byte(record.Messages[2].Content))
		if err != nil {
			return nil, fmt.Errorf("catalog row %d: %w", index+1, err)
		}
		tags := record.Groove
		if score.Bars%tags.CycleBars != 0 {
			return nil, fmt.Errorf("catalog row %d cuts off its repeating cycle", index+1)
		}
		voices := make(map[string]bool)
		kicksOnQuarters := 0
		for _, event := range score.Events {
			if event.Instrument == "kick" && *event.Tick%96 == 0 {
				kicksOnQuarters++
			}
			voices[event.Instrument] = true
			if event.Instrument == "snare" && event.Velocity <= 55 {
				voices["ghost"] = true
			}
			if strings.HasPrefix(event.Instrument, "tom_") {
				voices["tom"] = true
			}
			if strings.HasSuffix(event.Instrument, "_hat") {
				voices["hat"] = true
			}
		}
		catalog = append(catalog, catalogGroove{
			score: score, tags: tags, family: normalized(tags.Family),
			search: normalized(tags.Family + " " + tags.Feel + " " + tags.Color + " " + record.Messages[1].Content),
			voices: voices, fourOnFloor: score.Meter == "4/4" && kicksOnQuarters == score.Bars*4,
			handGap: minimumHandGap(score.Events, score.Bars*meters[score.Meter].ticks()),
		})
	}
	return catalog, nil
})

type grooveBrief struct {
	text, style, meter, beatUnit string
	bpm, bars                    int
	grouping                     []int
	required                     [][]string
	excluded                     []string
	influences                   map[string]bool
	fourOnFloor, omitFourOnFloor bool
}

const tempoUnit = `(?:dotted[- ]quarter(?:[- ]note)?|quarter(?:[- ]note)?)`

var (
	meterPattern = regexp.MustCompile(`\b([0-9]+)\s*/\s*([0-9]+)\b`)
	bpmPrefix    = regexp.MustCompile(`\bbpm\s*[:=]?\s*([+-]?[0-9]+(?:\.[0-9]+)?)\b(?:\s+` + tempoUnit + `)?`)
	bpmPattern   = regexp.MustCompile(`(?:^|[^[:alnum:]])([+-]?[0-9]+(?:\.[0-9]+)?)(?:\s+` + tempoUnit + `)?\s*bpm\b`)
	barsPattern  = regexp.MustCompile(`\b([0-9]+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve)[ -]+bars?\b`)
	groupPattern = regexp.MustCompile(`\b[0-9]+(?:\s*\+\s*[0-9]+)+\b`)
	styleTerms   = [][]string{
		{"rock", "rock", "garage rock", "prog rock", "progressive rock"},
		{"funk", "funk", "funky", "second line", "new orleans funk", "disco funk"},
		{"blues", "blues", "bluesy", "chicago shuffle"},
		{"jazz", "jazz", "jazzy", "bebop", "jazz waltz"},
		{"latin", "latin", "bossa", "bossa nova", "samba", "samba jazz", "mambo", "songo", "afro cuban"},
		{"soul", "soul", "neo soul", "r b", "r and b"},
		{"disco", "disco"},
		{"reggae", "reggae", "one drop", "rockers"},
		{"hip_hop", "hip hop", "boom bap", "lo fi", "lofi"},
		{"fusion", "fusion", "odd time fusion", "jazz fusion"},
	}
	voiceTerms = [][]string{
		{"ghost", "snare ghost notes", "snare ghost note", "snare ghosts", "ghost notes", "ghost note"},
		{"open_hat", "open hi hats", "open hi hat", "open hats", "open hat"},
		{"closed_hat", "closed hi hats", "closed hi hat", "closed hats", "closed hat"},
		{"foot_hat", "foot hi hats", "foot hi hat", "pedal hi hats", "pedal hi hat", "foot hats", "foot hat"},
		{"hat", "hi hats", "hi hat", "hats", "hat"},
		{"ride", "ride cymbal", "ride"}, {"rim", "rim clicks", "rim click", "cross stick", "side stick", "rim"},
		{"kick", "bass drum", "kicks", "kick"}, {"snare", "snares", "snare"},
		{"tom_low", "floor toms", "floor tom", "low toms", "low tom"},
		{"tom_mid", "mid toms", "mid tom"}, {"tom_high", "high toms", "high tom"},
		{"tom", "toms", "tom"}, {"cowbell", "cowbells", "cowbell"},
		{"clave", "clave"}, {"shaker", "shakers", "shaker"}, {"crash", "crash cymbals", "crash cymbal", "crash"},
	}
	accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n")
)

func normalized(text string) string {
	return " " + strings.Join(strings.FieldsFunc(accents.Replace(strings.ToLower(text)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ") + " "
}

func phraseIn(text, phrase string) bool {
	return strings.Contains(text, normalized(phrase))
}

func parseBrief(prompt string) (grooveBrief, error) {
	lower := strings.ReplaceAll(strings.ToLower(prompt), "−", "-")
	brief := grooveBrief{text: normalized(lower)}
	if match := meterPattern.FindStringSubmatch(lower); match != nil {
		brief.meter = match[1] + "/" + match[2]
		if _, ok := meters[brief.meter]; !ok {
			return brief, fmt.Errorf("the catalog does not support %s meter", brief.meter)
		}
	}
	tempo := bpmPattern.FindStringSubmatch(lower)
	if tempo == nil {
		tempo = bpmPrefix.FindStringSubmatch(lower)
	}
	if match := tempo; match != nil {
		bpm, _ := strconv.Atoi(match[1])
		if bpm < 40 || bpm > 220 {
			return brief, errors.New("tempo must be between 40 and 220 BPM")
		}
		brief.bpm = bpm
		switch {
		case strings.Contains(match[0], "dotted"):
			brief.beatUnit = "dotted_quarter"
		case strings.Contains(match[0], "quarter"):
			brief.beatUnit = "quarter"
		}
	}
	if match := barsPattern.FindStringSubmatch(lower); match != nil {
		bars, err := strconv.Atoi(match[1])
		if err != nil {
			bars = slices.Index([]string{"", "one", "two", "three", "four"}, match[1])
		}
		if bars < 1 || bars > 4 {
			return brief, errors.New("request one to four bars")
		}
		brief.bars = bars
	}
	if match := groupPattern.FindString(lower); match != "" {
		for _, count := range strings.Split(match, "+") {
			group, err := strconv.Atoi(strings.TrimSpace(count))
			if err != nil || group < 1 || group > 12 {
				return brief, errors.New("group counts must be between 1 and 12")
			}
			brief.grouping = append(brief.grouping, group)
		}
	}

	bestLength := 0
	for _, terms := range styleTerms {
		for _, label := range terms[1:] {
			if phraseIn(brief.text, label) && len(label) > bestLength {
				brief.style, bestLength = terms[0], len(label)
			}
		}
	}
	for _, term := range []string{"four on the floor", "4 on the floor"} {
		if phraseIn(brief.text, term) {
			brief.fourOnFloor = true
			for _, negative := range []string{"no ", "without ", "avoid ", "omit ", "skip "} {
				brief.omitFourOnFloor = brief.omitFourOnFloor || phraseIn(brief.text, negative+term)
			}
		}
	}
	brief.fourOnFloor = brief.fourOnFloor && !brief.omitFourOnFloor
	if brief.style == "" && brief.fourOnFloor {
		brief.style = "disco"
	}
	brief.required, brief.excluded = voiceConstraints(prompt)
	return brief, nil
}

func voiceConstraints(prompt string) (required [][]string, excluded []string) {
	clauses := strings.FieldsFunc(prompt, func(r rune) bool {
		return strings.ContainsRune(",.;!?\n", r)
	})
	for _, clause := range clauses {
		remaining := strings.TrimLeft(normalized(clause), " ")
		negative, alternate := false, false
	nextWord:
		for remaining != "" {
			for _, voice := range voiceTerms {
				for _, name := range voice[1:] {
					if rest, found := strings.CutPrefix(remaining, name+" "); found {
						switch {
						case negative:
							excluded = append(excluded, voice[0])
						case alternate && len(required) > 0:
							last := len(required) - 1
							required[last] = append(required[last], voice[0])
						default:
							required = append(required, []string{voice[0]})
						}
						remaining, alternate = rest, false
						continue nextWord
					}
				}
			}
			word, rest, _ := strings.Cut(remaining, " ")
			switch word {
			case "no", "without", "omit", "skip", "avoid", "remove", "drop":
				negative = true
			case "or":
				alternate = true
			case "a", "any", "the", "and":
			default:
				negative, alternate = false, false
			}
			remaining = rest
		}
	}
	return required, excluded
}

func retime(brief grooveBrief, tempo Tempo) Tempo {
	if brief.bpm != 0 {
		tempo.BPM = brief.bpm
	}
	if brief.beatUnit != "" {
		tempo.BeatUnit = brief.beatUnit
	}
	return tempo
}

func ranking(brief grooveBrief, candidate catalogGroove) (int, bool) {
	score := candidate.score
	if brief.style != "" && score.Style != brief.style ||
		brief.meter != "" && score.Meter != brief.meter ||
		len(brief.grouping) > 0 && !slices.Equal(brief.grouping, score.Grouping) ||
		brief.bars != 0 && brief.bars%candidate.tags.CycleBars != 0 ||
		brief.fourOnFloor && !candidate.fourOnFloor ||
		brief.omitFourOnFloor && candidate.fourOnFloor ||
		!tempoAllowsHands(retime(brief, score.Tempo), candidate.handGap) {
		return 0, false
	}
	for _, options := range brief.required {
		if !slices.ContainsFunc(options, func(voice string) bool { return candidate.voices[voice] }) {
			return 0, false
		}
	}
	for _, excluded := range brief.excluded {
		if candidate.voices[excluded] {
			return 0, false
		}
	}
	if len(brief.influences) > 0 && !slices.ContainsFunc(candidate.tags.Influences, func(artist string) bool {
		return brief.influences[artist]
	}) {
		return 0, false
	}
	rank := 0
	if strings.Contains(brief.text, candidate.family) {
		rank += 18
	}
	for _, term := range []string{"swing", "swung", "shuffle", "half time", "laid back", "late pocket", "straight", "sparse", "second line", "displaced", "triplet", "sixteenth", "eighth", "diddles", "offbeat", "ostinato"} {
		if phraseIn(brief.text, term) && phraseIn(candidate.search, term) {
			rank += 6
		}
	}
	if brief.meter == "" && score.Meter == "4/4" {
		rank += 3
	}
	if score.Bars == brief.bars {
		rank += 3
	}
	target := brief.bpm
	if target == 0 {
		switch {
		case phraseIn(brief.text, "upbeat") || phraseIn(brief.text, "fast"):
			target = 122
		case phraseIn(brief.text, "slow"):
			target = 72
		default:
			target = score.Tempo.BPM
		}
	}
	// Compare tempos in quarter notes if the request specifies a different beat unit.
	comparable := score.Tempo.BPM
	if brief.beatUnit == "quarter" && score.Tempo.BeatUnit == "dotted_quarter" {
		comparable = comparable * 3 / 2
	} else if brief.beatUnit == "dotted_quarter" && score.Tempo.BeatUnit == "quarter" {
		comparable = comparable * 2 / 3
	}
	distance := target - comparable
	return rank - max(distance, -distance)/5, true
}

func chooseGroove(brief grooveBrief, catalog []catalogGroove) (Score, error) {
	brief.influences = make(map[string]bool)
	for _, candidate := range catalog {
		for _, artist := range candidate.tags.Influences {
			parts := strings.Fields(artist)
			surname := parts[len(parts)-1]
			shorthand := surname != "Lay" && surname != "Below"
			if phraseIn(brief.text, artist) || shorthand && phraseIn(brief.text, surname) ||
				artist == "José Luis Changuito Quintana" && phraseIn(brief.text, "Changuito") {
				brief.influences[artist] = true
			}
		}
	}
	var chosen *catalogGroove
	best := 0
	for i := range catalog {
		candidate := &catalog[i]
		if rank, matches := ranking(brief, *candidate); matches && (chosen == nil || rank > best) {
			chosen, best = candidate, rank
		}
	}
	if chosen == nil {
		return Score{}, errors.New("no catalog groove matches those style, meter, grouping, length, or instrument requirements")
	}
	score := chosen.score
	score.Tempo = retime(brief, score.Tempo)
	if brief.bars != 0 && brief.bars != score.Bars {
		cycleTicks := meters[score.Meter].ticks() * chosen.tags.CycleBars
		events := make([]Event, 0, len(score.Events))
		for bar := 0; bar < brief.bars; bar += chosen.tags.CycleBars {
			start := bar * meters[score.Meter].ticks()
			for _, event := range score.Events {
				if *event.Tick >= cycleTicks {
					break
				}
				tick := start + *event.Tick
				event.Tick = &tick
				events = append(events, event)
			}
		}
		score.Bars, score.Events = brief.bars, events
	}
	return score, validateScore(score)
}

func serveProcedural(w http.ResponseWriter, prompt string, base *Score) {
	edits, seed := parseEdits(prompt)
	brief, err := parseBrief(seed)
	var score Score
	if err == nil {
		if base != nil {
			if brief.fourOnFloor && !phraseIn(brief.text, "disco") {
				if brief.style == "disco" {
					brief.style = ""
				}
			}
			err = validateScore(*base)
			if err == nil {
				score, err = editGroove(*base, brief, edits)
			}
		} else {
			catalog, loadErr := loadCatalog()
			if loadErr != nil {
				writeError(w, http.StatusInternalServerError, "The groove catalog could not be read.")
				return
			}
			score, err = chooseGroove(brief, catalog)
			if err == nil && (len(edits.hits) > 0 || edits.swing != 0) {
				score, err = editGroove(score, brief, edits)
			}
		}
	}
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, score)
}
