package main

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type hitCount struct {
	voice  string
	total  int
	perBar bool
}

const countOnes = `one|two|three|four|five|six|seven|eight|nine`
const countTens = `twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety`
const smallCount = `zero|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|(?:` + countTens + `)(?:[- ]+(?:` + countOnes + `))?|` + countOnes
const numberOfHits = `[+-]?[0-9]+(?:\.[0-9]+)?|(?:` + countOnes + `) hundred(?:[- ]+(?:and )?(?:` + smallCount + `))?|` + smallCount

var hitCountPattern = regexp.MustCompile(`(?i)(?:^|[^[:alnum:].+/–—-])(?:(at least|at most|up to|around|about|roughly|approximately|under|over|fewer than|more than)\s+)?(` + numberOfHits + `)\s+(` + drumVoiceNames() + `)\b(?:[- ]+(?:hits?|strokes?|notes?))?(?:\s+((?:per|in each|in every|each|every) (?:bar|measure))\b|\s+(?:total|overall|(?:across|in) the (?:whole )?loop)\b)?`)
var drumMachineNumber = regexp.MustCompile(`^(?:606|707|808|909)$`)
var drumMachineArticle = regexp.MustCompile(`(?i)\b(?:a|an|the)\s+$`)
var continuedNumber = regexp.MustCompile(`[0-9]\s*[/–—-]\s*$`)
var negatedCount = regexp.MustCompile(`(?i)\b(?:anything but|except|not|don[']t|never|no|without|avoid|omit|skip|remove|drop)(?: +(?:add|adding|use|using|include|including))? +(?:exactly +)?$`)

var nonCountContext = regexp.MustCompile(`(?i)^\s+(?:accents?|patterns?|figures?|(?:per|in each|in every) group)\b`)
var beginsPlacement = regexp.MustCompile(`(?i)^\s+on\s+`)

func parseHitCounts(prompt string) ([]hitCount, string, error) {
	prompt = strings.ReplaceAll(prompt, "−", "-")
	clean := []byte(prompt)
	var counts []hitCount
	for _, span := range hitCountPattern.FindAllStringSubmatchIndex(prompt, -1) {
		if span[2] >= 0 || nonCountContext.MatchString(prompt[span[1]:]) {
			continue // An approximate request can still guide the normal style matcher.
		}
		prefix := strings.ReplaceAll(prompt[:span[4]], "’", "'")
		number := strings.ToLower(prompt[span[4]:span[5]])
		if continuedNumber.MatchString(prefix) || drumMachineNumber.MatchString(number) && drumMachineArticle.MatchString(prefix) {
			continue
		}
		if negatedCount.MatchString(prefix) {
			return nil, "", fmt.Errorf("specify the exact final hit count")
		}
		n, err := strconv.Atoi(number)
		if err != nil {
			n = spelledCount(number)
		}
		if n < 0 || n > 256 {
			return nil, "", fmt.Errorf("hit counts must be whole numbers between 0 and 256")
		}
		voice := resolveVoice(prompt[span[6]:span[7]])
		count := hitCount{voice: voice, total: n, perBar: span[8] >= 0}
		for _, seen := range counts {
			if seen.voice == voice && seen.perBar == count.perBar && seen.total != count.total {
				return nil, "", fmt.Errorf("conflicting counts for %s", hitName(voice))
			}
		}
		if !slices.Contains(counts, count) {
			counts = append(counts, count)
		}
		// Leave a following positional directive for the beat editor to read.
		end := span[1]
		if beginsPlacement.MatchString(prompt[end:]) {
			for i := span[7]; i < end; i++ {
				clean[i] = ' '
			}
			end = span[6]
		}
		for i := span[4]; i < end; i++ {
			clean[i] = ' '
		}
	}
	return counts, string(clean), nil
}

func spelledCount(number string) int {
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	if n := slices.Index(words, number); n >= 0 {
		return n
	}
	number = strings.ReplaceAll(number, "-", " ")
	if hundreds, rest, found := strings.Cut(number, " hundred"); found {
		n := 100 * slices.Index(words, hundreds)
		rest = strings.TrimPrefix(strings.TrimSpace(rest), "and ")
		if rest != "" {
			n += spelledCount(rest)
		}
		return n
	}
	tens, ones, _ := strings.Cut(number, " ")
	n := slices.Index(strings.Split(countTens, "|"), tens)
	if n < 0 {
		return -1
	}
	return (n+2)*10 + max(0, slices.Index(words, strings.TrimSpace(ones)))
}

func hitName(voice string) string {
	switch voice {
	case "ghost":
		return "ghost note"
	case "hat":
		return "hi-hat hit"
	default:
		return strings.ReplaceAll(voice, "_", " ") + " hit"
	}
}

func countText(voice string, n int) string {
	name := hitName(voice)
	if n != 1 {
		name += "s"
	}
	return fmt.Sprintf("%d %s", n, name)
}

func countHits(events []Event, voice string, start, stop int) int {
	n := 0
	for _, e := range events {
		if *e.Tick >= start && *e.Tick < stop && matchesVoice(e, voice) {
			n++
		}
	}
	return n
}

func (c hitCount) span(score Score) int {
	bars := score.Bars
	if c.perBar {
		bars = 1
	}
	return meters[score.Meter].ticks() * bars
}

func verifyHitCounts(score Score, counts []hitCount) error {
	length := meters[score.Meter].ticks() * score.Bars
	for _, c := range counts {
		span := c.span(score)
		for start := 0; start < length; start += span {
			if actual := countHits(score.Events, c.voice, start, start+span); actual != c.total {
				where := ""
				if c.perBar {
					where = fmt.Sprintf(" in bar %d", start/span+1)
				}
				return fmt.Errorf("expected %s%s, got %d", countText(c.voice, c.total), where, actual)
			}
		}
	}
	return nil
}

func countEvent(voice string, tick, quarter, bar, grid int, score Score) Event {
	instrument := voice
	velocity := 88
	switch voice {
	case "ghost":
		instrument, velocity = "snare", 36+(quarter*5+tick%96+bar*3)%13
	case "snare":
		velocity = 96
	case "kick":
		velocity = 96
		if tick%96 != 0 {
			velocity = 80 + (quarter*7+bar*3)%11
		}
		if score.Style == "jazz" {
			velocity -= 30
		}
	case "hat", "closed_hat", "open_hat", "ride", "shaker", "clave", "cowbell", "foot_hat":
		if voice == "hat" {
			instrument = "closed_hat"
		}
		velocity = 64
		if tick%96 == 0 {
			velocity += 10
		}
	case "tom":
		instrument = "tom_low"
	case "crash":
		velocity = 100
	}
	event := editedEvent(tick, instrument, velocity)
	if voice == "ghost" && score.Style == "hip_hop" && grid == 48 {
		event.Offset = 2
	}
	return event
}

type hitPlacement struct {
	tick  int
	voice string
}

// Exact counts reserve positional hits and keep already-satisfied overlapping
// constraints, such as ghost notes within a larger snare total.
func applyHitCounts(score *Score, counts []hitCount, required []Event, blocked []hitPlacement, excluded []string, voices [][]string) error {
	if err := verifyHitCounts(*score, counts); err == nil {
		return nil
	}
	barTicks := meters[score.Meter].ticks()
	length := barTicks * score.Bars
	grid := offbeatGrid(score.Events)
	ordered := orderHitCounts(*score, counts)
	for i, c := range ordered {
		protected := func(e Event) bool {
			return slices.ContainsFunc(required, func(hit Event) bool {
				return *e.Tick == *hit.Tick && e.Instrument == hit.Instrument
			}) || slices.ContainsFunc(ordered[:i], func(previous hitCount) bool {
				return matchesVoice(e, previous.voice)
			})
		}
		allowed := func(e Event) bool {
			return !protected(e) && !slices.ContainsFunc(excluded, func(voice string) bool {
				return matchesVoice(e, voice)
			}) && !slices.ContainsFunc(blocked, func(hit hitPlacement) bool {
				return *e.Tick == hit.tick && matchesVoice(e, hit.voice)
			})
		}
		trim := func(start, stop, target int) int {
			have := countHits(score.Events, c.voice, start, stop)
			for have > target {
				index := leastEssentialHit(*score, c.voice, start, stop, barTicks, protected)
				if index < 0 {
					break
				}
				event := score.Events[index]
				score.Events = slices.Delete(score.Events, index, index+1)
				// Keep the last occurrence of an explicitly requested instrument.
				if slices.ContainsFunc(voices, func(options []string) bool {
					matches := func(e Event) bool {
						return slices.ContainsFunc(options, func(voice string) bool { return matchesVoice(e, voice) })
					}
					return matches(event) && !slices.ContainsFunc(score.Events, matches)
				}) {
					required = append(required, event)
					score.Events = append(score.Events, event)
					continue
				}
				have--
			}
			return have
		}
		caps := make([]int, score.Bars)
		for bar := range caps {
			caps[bar] = 256
			start := bar * barTicks
			for _, wider := range ordered[i+1:] {
				if !wider.perBar || groupVoice(c.voice) != wider.voice {
					continue
				}
				cap := wider.total
				for _, e := range score.Events {
					if *e.Tick >= start && *e.Tick < start+barTicks && matchesVoice(e, wider.voice) &&
						!matchesVoice(e, c.voice) && protected(e) {
						cap--
					}
				}
				caps[bar] = min(caps[bar], cap)
			}
			if caps[bar] != 256 && trim(start, start+barTicks, caps[bar]) > caps[bar] {
				return fmt.Errorf("requested %s cannot fit the other required hits in bar %d", hitName(c.voice), bar+1)
			}
		}
		span := c.span(*score)
		for start := 0; start < length; start += span {
			stop := start + span
			have := trim(start, stop, c.total)
			if have < c.total {
				have += placeCountedHits(score, c, start, stop, barTicks, grid, c.total-have, caps, allowed)
			}
			if have != c.total {
				return fmt.Errorf("could not fit %s: %w", countText(c.voice, c.total), verifyHitCounts(*score, []hitCount{c}))
			}
		}
	}
	return verifyHitCounts(*score, counts)
}

func orderHitCounts(score Score, counts []hitCount) []hitCount {
	length := meters[score.Meter].ticks() * score.Bars
	pending := slices.Clone(counts)
	// Free capacity on independent voices before adding notes. Within a group,
	// satisfy the specific voice first so the group total includes those hits.
	delta := func(c hitCount) int {
		target := c.total
		if c.perBar {
			target *= score.Bars
		}
		return target - countHits(score.Events, c.voice, 0, length)
	}
	slices.SortStableFunc(pending, func(a, b hitCount) int { return cmp.Compare(delta(a), delta(b)) })
	ordered := make([]hitCount, 0, len(counts))
	for len(pending) > 0 {
		i := slices.IndexFunc(pending, func(c hitCount) bool {
			return !slices.ContainsFunc(pending, func(other hitCount) bool {
				return groupVoice(other.voice) == c.voice || other.voice == c.voice && other.perBar && !c.perBar
			})
		})
		ordered = append(ordered, pending[i])
		pending = slices.Delete(pending, i, i+1)
	}
	return ordered
}

func groupVoice(voice string) string {
	switch voice {
	case "ghost":
		return "snare"
	case "closed_hat", "open_hat", "foot_hat":
		return "hat"
	case "tom_low", "tom_mid", "tom_high":
		return "tom"
	default:
		return ""
	}
}

func leastEssentialHit(score Score, voice string, start, stop, barTicks int, protected func(Event) bool) int {
	density := make([]int, score.Bars)
	for _, e := range score.Events {
		if matchesVoice(e, voice) {
			density[*e.Tick/barTicks]++
		}
	}
	index, least := -1, 0
	for i, e := range score.Events {
		tick := *e.Tick
		if tick < start || tick >= stop || !matchesVoice(e, voice) || protected(e) {
			continue
		}
		priority := e.Velocity - 10*density[tick/barTicks]
		if tick%96 == 0 && e.Instrument == "kick" {
			priority += 25
		}
		if tick%barTicks == 0 && e.Instrument == "kick" {
			priority += 32
		}
		if e.Instrument == "snare" && !matchesVoice(e, "ghost") {
			priority += 48
		}
		if index < 0 || priority < least {
			index, least = i, priority
		}
	}
	return index
}

func placeCountedHits(score *Score, c hitCount, start, stop, barTicks, grid, needed int, caps []int, allowed func(Event) bool) int {
	length := score.Bars * barTicks
	subdivisions := []int{0, grid / 2, grid, 48 + grid/2}
	if grid == 64 {
		subdivisions = []int{0, 32, 64}
	}
	if meters[score.Meter].denominator == 8 {
		subdivisions = []int{0, 24, 48, 72}
		grid = 48
	}
	byBar := make([]int, score.Bars)
	motif := make(map[int]int)
	for _, e := range score.Events {
		if matchesVoice(e, c.voice) {
			byBar[*e.Tick/barTicks]++
			motif[*e.Tick%barTicks]++
		}
	}
	var slots []Event
	for bar := start / barTicks; bar < stop/barTicks; bar++ {
		for phase := 0; phase < barTicks; phase += 96 {
			for _, subdivision := range subdivisions {
				if phase+subdivision >= barTicks {
					continue
				}
				tick := bar*barTicks + phase + subdivision
				voices := []string{c.voice}
				switch c.voice {
				case "hat":
					voices = []string{"closed_hat", "open_hat", "foot_hat"}
				case "tom":
					voices = []string{"tom_low", "tom_mid", "tom_high"}
				}
				for _, voice := range voices {
					e := countEvent(voice, tick, phase/96, bar, grid, *score)
					if allowed(e) {
						slots = append(slots, e)
					}
				}
			}
		}
	}
	placed := 0
	for len(slots) > 0 && placed < needed && len(score.Events) < 256 {
		slices.SortStableFunc(slots, func(a, b Event) int {
			rank := func(e Event) int {
				tick := *e.Tick
				bar, phase := tick/barTicks, tick%barTicks
				rating := 3*motif[phase] - 16*byBar[bar]
				switch c.voice {
				case "hat":
					if e.Instrument != "closed_hat" {
						rating -= 12
					}
				case "tom":
					if e.Instrument != "tom_low" {
						rating -= 12
					}
				case "ghost":
					if phase%96 == 0 {
						rating -= 20
					}
				case "kick":
					if phase == 0 || phase%96 == grid {
						rating += 8
					}
				case "crash":
					if phase == 0 {
						rating += 16
					}
				}
				return rating
			}
			return cmp.Compare(rank(b), rank(a))
		})
		e := slots[0]
		slots = slots[1:]
		if byBar[*e.Tick/barTicks] >= caps[*e.Tick/barTicks] {
			continue
		}
		if (e.Instrument == "kick" || e.Instrument == "foot_hat") && !pedalSpacing(*score, e, length) {
			continue
		}
		score.Events = append(score.Events, e)
		if validateScore(*score) != nil {
			score.Events = score.Events[:len(score.Events)-1]
			continue
		}
		byBar[*e.Tick/barTicks]++
		motif[*e.Tick%barTicks]++
		placed++
	}
	return placed
}

func pedalSpacing(score Score, added Event, length int) bool {
	tempoTicks := 96
	if score.Tempo.BeatUnit == "dotted_quarter" {
		tempoTicks = 144
	}
	for _, e := range score.Events {
		if e.Instrument != added.Instrument {
			continue
		}
		gap := *e.Tick + e.Offset - *added.Tick - added.Offset
		gap = max(gap, -gap)
		if min(gap, length-gap)*60000 < 48*score.Tempo.BPM*tempoTicks {
			return false
		}
	}
	return true
}
