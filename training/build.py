"""Rebuild train.jsonl from original drum ostinatos and their accompaniment variants."""
from collections import Counter
import json
from pathlib import Path

from counted import add_counted_examples, count_line, counted_events
from expansion import add_families
from repertoire import add_repertoire

ROOT = Path(__file__).resolve().parent
SYSTEM = (ROOT / "system-prompt.txt").read_text().strip()
VOICES = {
    "k": ("kick", 101), "s": ("snare", 109), "g": ("snare", 39),
    "x": ("rim", 83), "h": ("closed_hat", 64), "o": ("open_hat", 76),
    "p": ("foot_hat", 48), "r": ("ride", 68), "c": ("cowbell", 67),
    "v": ("clave", 70), "z": ("shaker", 57), "t": ("tom_high", 82),
    "m": ("tom_mid", 81), "l": ("tom_low", 86), "a": ("crash", 103),
}
KEEPERS = {'ride', 'closed_hat', 'open_hat', 'cowbell', 'clave', 'shaker', 'crash'}
rows = []
patterns = set()


def read_cell(cell, bar, unit, index, accents, lag, style, light_offbeat, soft_kicks):
    for part in cell.split(";"):
        if not part.strip():
            continue
        symbol, steps = part.split(":")
        voice, base = VOICES[symbol]
        for position, mark in enumerate(steps.split()):
            tick = int(mark.rstrip("*")) * unit
            accent = mark.endswith("*") or symbol in "horcvz" and tick in accents
            displacement = lag if symbol == "s" else 0
            if style == "hip_hop" and symbol in "ho" and tick % 96:
                displacement = (-2, 3, 1, -3, 2)[(int(mark.rstrip("*")) + index) % 5]
            level = base + (12 if accent else 0) + (2, -6, 1, -9, -2)[(position + index) % 5]
            if symbol == "k" and style == "jazz":
                level -= 48
            elif symbol == "k" and soft_kicks:
                level -= 22
            if voice in KEEPERS and tick % 96 == light_offbeat:
                level -= 24
            event = dict(tick=bar + tick, instrument=voice, velocity=level)
            if displacement:
                event["offset"] = displacement
            yield event


def phrase(cell, bar_index):
    bars = cell.split("|")
    return bars[bar_index % len(bars)]


def family(name, style, meter, base_bpm, feel, hook, top, kick, *,
           hands="", artists=(), grouping=(), unit=24, lag=0, phrase_bars=1, variants=4,
           light_offbeat=None, soft_kicks=False):
    """Combine four authored timekeepers and five authored kick/hand responses."""
    num, denom = map(int, meter.split("/"))
    bar_ticks = num * 384 // denom
    count = []
    pulse = 0
    for group in grouping:
        count.append(pulse)
        pulse += group * 384 // denom
    assert not grouping or pulse == bar_ticks
    assert 1 <= variants <= len(top) and len(kick) == 5
    beat_unit = "dotted_quarter" if meter in ("6/8", "12/8") else "quarter"
    timing = "dotted-quarter BPM" if beat_unit == "dotted_quarter" else "BPM"
    for pattern_index, (color, timekeeper) in enumerate(top[:variants]):
        for response_index, response in enumerate(kick):
            serial = pattern_index * 5 + response_index
            bpm = base_bpm + (0, -4, 5, -2, 8)[response_index] + (0, 3, -3, 1)[pattern_index]
            bars = phrase_bars
            if (serial % 5 == 3 or serial % 5 == 0) and phrase_bars == 1:
                bars = 2
            if serial % 10 == 8 and style in ("blues", "reggae", "soul"):
                bars = 4
            if serial == 16 and name in ("straight rock", "jazz waltz", "dusty hip-hop"):
                bars = 3

            events = []
            for bar in range(bars):
                for cell in (phrase(timekeeper, bar), phrase(response, bar), phrase(hands, bar)):
                    events.extend(read_cell(cell, bar * bar_ticks, unit, response_index,
                                            count, lag, style, light_offbeat, soft_kicks))
            events.sort(key=lambda e: (e["tick"], e["instrument"]))
            audit(events, name, bar_ticks, bars, bpm, beat_unit, count)
            if light_offbeat is not None:
                top_velocities = {e["tick"]: e["velocity"] for e in events if e["instrument"] in KEEPERS}
                for tick, velocity in top_velocities.items():
                    if tick % 96 == light_offbeat:
                        assert velocity < top_velocities.get(tick - light_offbeat, 127), (name, tick)
            if soft_kicks:
                kicks = [e["velocity"] for e in events if e["instrument"] == "kick"]
                backbeats = [e["velocity"] for e in events if e["instrument"] == "snare" and e["velocity"] > 55]
                assert max(kicks) < min(backbeats), name

            # Compare the underlying pulse, ignoring dynamics, tempo and exact-repeat padding.
            keys = tuple((e["tick"], e["instrument"]) for e in events if e["tick"] < bar_ticks * phrase_bars)
            fingerprint = (meter, keys)
            assert fingerprint not in patterns, (name, serial)
            patterns.add(fingerprint)

            length = {1: "one bar", 2: "two bars", 3: "three bars", 4: "four bars"}[bars]
            groups = (" grouped " + "+".join(map(str, grouping))) if grouping else ""
            stats = f"{meter}{groups} at {bpm} {timing}"
            descriptions = (
                f"Give me {length} of {name} in {stats}, with {color}. {hook}",
                f"Make a {name} groove in {stats}. Keep it to {length}, use {color}, and let it loop. {hook}",
                f"I need {length} of {name} for a practice loop, {stats}. Use {color}. {hook}",
                f"Play a repeating {name} pocket with {color}, {stats}, for {length}. {hook}",
                f"Write {length} of {name} with {color} in {stats}. {hook}",
            )
            prompt = descriptions[response_index]
            if not grouping and response_index == 1 and pattern_index in (1, 3):
                prompt = f"Can you give me a {name} loop with {color}? {hook}"
            if artists and serial % 4 == 0:
                prompt += f" Take inspiration from {artists[0]}'s feel."
            score = {"schema_version": "1.0", "style": style, "meter": meter}
            if grouping:
                score["grouping"] = grouping
            score.update(tempo={"bpm": bpm, "beat_unit": beat_unit}, bars=bars,
                         ticks_per_quarter=96, events=events)
            score["metadata"] = {
                "description": f"A looping {name} groove with {color}. {hook}",
                "feel": feel,
                "timing": timing_description(style, meter, unit, grouping, lag, events),
            }
            rows.append({
                "messages": [
                    {"role": "system", "content": SYSTEM},
                    {"role": "user", "content": prompt},
                    {"role": "assistant", "content": json.dumps(score, ensure_ascii=False, separators=(",", ":"))},
                ],
                "groove": {
                    "family": name, "feel": feel, "color": color,
                    "influences": artists, "cycle_bars": phrase_bars,
                },
            })


def timing_description(style, meter, unit, grouping, lag, events):
    if style == "hip_hop":
        return "Dilla-inspired pocket with uneven hat offsets and a late snare backbeat."
    if meter in ("6/8", "12/8"):
        return "Compound eighths grouped in threes, with an accent at each group start."
    group_text = "+".join(map(str, grouping))
    if unit == 32:
        pulse = f" Accent the {group_text} groups." if grouping else ""
        top = [e["tick"] for e in events if e["instrument"] in KEEPERS]
        if any(tick % 96 == 64 for tick in top):
            return "Written triplet subdivisions give the offbeats a long-short swing." + pulse
        if all(tick % 96 == 0 for tick in top):
            return "Quarter-note timekeeping over a triplet grid for the drum replies." + pulse
        return "Triplet-grid timekeeping and drum replies." + pulse
    if grouping:
        return f"Straight subdivisions with accents grouped {group_text}."
    if lag:
        return f"Straight timekeeping with the written snare delayed by {lag} ticks."
    return "Straight written subdivisions with accents set by hit velocity."


def audit(events, name, bar_ticks, bars, bpm, beat_unit, accents):
    assert 1 <= len(events) <= 256, name
    written = set()
    sounding = {}
    for e in events:
        tick = e["tick"] + e.get("offset", 0)
        voice = e["instrument"]
        assert 0 <= tick < bar_ticks * bars, (name, e)
        assert 1 <= e["velocity"] <= 127, e
        assert (e["tick"], voice) not in written, (name, e)
        written.add((e["tick"], voice))
        bucket = sounding.setdefault(tick, [])
        assert voice not in bucket, (name, e)
        bucket.append(voice)

    for tick, voices in sounding.items():
        assert len(set(voices) - {"kick", "foot_hat"}) <= 2, (name, tick, voices)
        assert len(set(voices) & KEEPERS) <= 1, (name, tick, voices)
        assert not {"open_hat", "foot_hat"} <= set(voices), (name, tick, voices)
    for bar in range(bars):
        hits = [e for e in events if bar * bar_ticks <= e["tick"] < (bar + 1) * bar_ticks]
        pulse = [e for e in hits if e["instrument"] in KEEPERS]
        assert pulse and pulse[-1]["tick"] >= (bar + 1) * bar_ticks - 144, name
        toms = [e["tick"] for e in hits if e["instrument"].startswith("tom_")]
        if toms:
            assert toms[0] < (bar + 1) * bar_ticks - 96, (name, toms)
        for accent in accents:
            assert any((bar * bar_ticks + accent, v) in written for v in
                       ("kick", "ride", "closed_hat", "open_hat", "cowbell", "clave", "shaker", "snare", "rim")), (name, accent)

    # Greedy two-hand assignment over the seam; allow fast doubles and count feet separately.
    hand_clock = [-1000, -1000]
    ticks_per_beat = 144 if beat_unit == "dotted_quarter" else 96
    for cycle in range(2):
        for tick, voices in sorted(sounding.items()):
            ms = (tick + cycle * bar_ticks * bars) * 60000 / (bpm * ticks_per_beat)
            for _ in set(voices) - {"kick", "foot_hat"}:
                hand = hand_clock.index(min(hand_clock))
                assert ms - hand_clock[hand] >= 48, (name, bpm, tick, voices)
                hand_clock[hand] = ms


def counted_family(name, style, meter, bpm, feel, unit, grouping, lag, character, cells, metrics, variant):
    num, denom = map(int, meter.split("/"))
    bar_ticks = num * 384 // denom
    bars = len(cells)
    accents = [sum(grouping[:i]) * 384 // denom for i in range(len(grouping))]
    beat_unit = "dotted_quarter" if meter in ("6/8", "12/8") else "quarter"
    events = sorted(
        (e for bar, cell in enumerate(cells) for e in
         read_cell(cell, bar * bar_ticks, unit, variant, accents, lag, style, None, False)),
        key=lambda e: (e["tick"], e["instrument"]),
    )
    audit(events, name, bar_ticks, bars, bpm, beat_unit, accents)
    cycle = bars
    for size in range(1, bars):
        if bars % size == 0 and all(cells[i] == cells[i % size] for i in range(bars)):
            cycle = size
            break
    keys = tuple((e["tick"], e["instrument"]) for e in events if e["tick"] < bar_ticks * cycle)
    assert (meter, keys) not in patterns, (name, variant)
    patterns.add((meter, keys))
    counts = count_line(counted_events(events, metrics, bar_ticks, bars), variant)
    groups = " grouped " + "+".join(map(str, grouping)) if grouping else ""
    pulse = " dotted-quarter" if beat_unit == "dotted_quarter" else ""
    length = ("", "one bar", "two bars", "three bars", "four bars")[bars]
    specification = f"{meter}{groups} at {bpm}{pulse} BPM"
    prompts = (
        f"Give me {length} of {name} in {specification} with exactly {counts}. Keep it looping.",
        f"Make a {name} practice groove with {counts}. Use {length}, {specification}.",
        f"I need {length} of {name}, {specification}. Fit {counts} into the loop.",
        f"Write a {name} loop in {specification}. Over {length}, use {counts}.",
        f"Can you give me {length} of {name} in {specification}? Use {counts} and keep a steady pulse.",
    )
    score = {"schema_version": "1.0", "style": style, "meter": meter}
    if grouping:
        score["grouping"] = grouping
    score.update(tempo={"bpm": bpm, "beat_unit": beat_unit}, bars=bars,
                 ticks_per_quarter=96, events=events)
    score["metadata"] = {
        "description": f"A repeating {name} groove with {character}. The part uses {counts}.",
        "feel": feel,
        "timing": timing_description(style, meter, unit, grouping, lag, events),
    }
    rows.append({
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": prompts[variant % len(prompts)]},
            {"role": "assistant", "content": json.dumps(score, ensure_ascii=False, separators=(",", ":"))},
        ],
        "groove": {"family": f"{name} with exact counts", "feel": feel, "color": character,
                   "influences": (), "cycle_bars": cycle},
    })


family("straight rock", "rock", "4/4", 116, "straight eighths",
       "Plant the backbeat on two and four.",
       (("accented eighth-note hats","h:0* 2 4* 6 8* 10 12* 14"),("dry quarter-note hats","h:0 4 8 12"),
            ("offbeat hats","h:2 6 10 14"),("a short open hat on the last and","h:0 2 4 6 8 10 12;o:14")),
       ("k:0 6 8", "k:0 2 8 14", "k:0 7 8 11", "k:0 4 8 10 15", "k:0 3 6 10 14"), hands="s:4 12")

family("half-time rock", "rock", "4/4", 122, "half time",
       "Put the snare on three and keep the pulse steady.",
       (("a steady eighth-note ride","r:0* 2 4 6 8* 10 12 14"),("dry eighth-note hats","h:0 2 4 6 8 10 12 14"),
            ("a quarter-note ride","r:0* 4 8* 12"),("short hat pairs","h:0 1 4 5 8 9 12 13")),
       ("k:0 6 11", "k:0 3 7 14", "k:0 2 6 10 13", "k:0 5 9 15", "k:0 3 6 12 14"), hands="s:8")

family("compound rock", "rock", "6/8", 77, "two broad pulses",
       "Set the snare on the second broad pulse and keep the eighth-note movement.",
       (("accented closed hats","h:0 2 4 6 8 10"),("a four-note ride line","r:0 4 6 10"),
            ("an open hat after the backbeat","h:0 2 4 6 10;o:8"),("a lighter edge on the ride","r:0 2 4 6 8 10")),
       ("k:0 4", "k:0 8 10", "k:0 3 7 10", "k:0 2 5 8", "k:0 4 8 11"), hands="s:6",grouping=(3,3))

family("progressive rock", "rock", "7/4", 118, "four plus three",
       "Ground both group starts with the kick. Let a floor tom answer in each group.",
       (("quarter-note hats","h:0 4 8 12 16 20 24"),("even eighth-note hats","h:0 2 4 6 8 10 12 14 16 18 20 22 24 26"),
            ("a repeating ride accent","r:0 4 8 12 16 20 24 26"),("two open-hat accents","h:0 2 4 8 12 16 18 20 24;o:14 26")),
       ("k:0 6 8 16 22", "k:0 3 11 16 19 24", "k:0 8 14 16 22 26", "k:0 6 10 16 18 23", "k:0 2 7 12 16 21 26"),
       hands="s:4 12 20;l:6 22", grouping=(4,3), artists=("Gavin Harrison",))

family("rudimental pocket funk", "funk", "4/4", 100, "diddles and soft ghosts",
       "Keep a low ghost-note dialogue beneath the backbeats.",
       (("short hat pairs","h:0 1 4 6 8 9 12 14"),("a sparser hat line","h:0 4 6 8 12 14"),
            ("an open hat on the and of four","h:0 2 4 6 8 10 12;o:14"),("a broken ride line","r:0 2 4 7 8 10 12 14")),
       ("k:0 3 8 11", "k:0 2 6 8 15", "k:0 5 8 10 13", "k:0 3 7 10 14", "k:0 2 7 9 11 15"),
       hands="s:4 12;g:3 7 11 15", artists=("Steve Gadd",))

family("displaced funk", "funk", "4/4", 109, "displaced backbeat",
       "Lean into a snare on the and of two, then answer on four.",
       (("accented broken hats","h:0* 2 4 5 8* 10 12 14"),("an opening hat on two","h:0 2 5 8 10 12 14;o:4"),
            ("a sparse ride line","r:0 3 4 8 11 12 14"),("short pairs and gaps on the hats","h:0 1 4 5 8 9 12 13 14")),
       ("k:0 3 8 11", "k:0 2 7 10 15", "k:0 4 8 13", "k:0 1 5 9 14", "k:0 3 7 8 10 15"),
       hands="s:6 12;g:2 9 14", artists=("David Garibaldi", "Mike Clark"), variants=2)

family("disco funk", "funk", "4/4", 114, "four on the floor",
       "Keep the kick on every quarter and slip ghost notes between the backbeats.",
       (("an open hat on all four offbeats", "h:0 4 8 12;o:2 6 10 14"),
        ("a syncopated hat line with two open accents", "h:0 1 4 5 8 10 12 13;o:6 14")),
       ("k:0 4 8 12;g:3 9", "k:0 4 8 12;g:5 15", "k:0 4 8 12;g:2 11",
        "k:0 4 8 12;g:6 13", "k:0 4 8 12;g:7 10"),
       hands="s:4 12", variants=2)

family("ghost-note funk", "funk", "4/4", 108, "tight ghost notes",
       "Let the quiet snare notes link the stronger backbeats.",
       (("a light sixteenth-note hat pulse","h:0* 1 2 4* 6 7 8* 9 10 12* 14 15"),("plain eighth-note hats","h:0 2 4 6 8 10 12 14"),
            ("an offbeat open hat on three and","h:0 2 4 6 8 12 14;o:10"),("gapped hats","h:0 1 4 7 8 10 12 15")),
       ("k:0 3 7 10 15", "k:0 2 6 8 11", "k:0 5 8 11 14", "k:0 1 6 9 13", "k:0 3 8 10 14"),
       hands="s:4 12;g:2 5 9 13", artists=("Clyde Stubblefield","Jabo Starks"))

family("New Orleans second-line funk", "funk", "4/4", 103, "second line",
       "Let the offbeat snare calls and bass-drum replies drive the loop.",
       (("time on the ride","r:0 2 4 6 8 10 12 14"),("a bell on the downbeats and offbeats","c:0 2 4 7 8 11 12 14"),
            ("spaced hats","h:0 2 5 6 8 10 12 14"),("a long hat accent on four and","h:0 2 4 6 8 10 12;o:14")),
       ("k:0 3 8 13", "k:0 6 8 11", "k:0 2 5 9 14", "k:0 4 7 10 15", "k:0 3 7 8 14"),
       hands="s:4 11;g:2 7 14", artists=("Zigaboo Modeliste",))

family("Chicago blues shuffle", "blues", "4/4", 106, "shuffle",
       "Keep a long-short shuffle against the two-and-four snare.",
       (("shuffling hats","h:0 2 3 5 6 8 9 11"),("a shuffled ride","r:0 2 3 5 6 8 9 11"),
            ("a late open-hat skip","h:0 2 3 5 6 8 9;o:11"),("a sparse shuffle ride","r:0 3 5 6 9 11")),
       ("k:0 3 6 9", "k:0 2 6 8", "k:0 5 6 11", "k:0 2 3 6 9 11", "k:0 3 5 8 9"),
       hands="s:3 9", unit=32, artists=("Fred Below", "Willie Big Eyes Smith"))

family("half-time blues shuffle", "blues", "4/4", 85, "half-time shuffle",
       "Put the main snare on beat three and tuck ghost notes into the middle triplets.",
       (("long-short hats","h:0 2 3 5 6 8 9 11"),("a long-short ride","r:0 2 3 5 6 8 9 11"),
            ("room around the half-time snare","h:0 2 3 5 6 9 11"),("an open hat on the final skip","h:0 2 3 5 6 8 9;o:11")),
       ("k:0 5 8", "k:0 2 6 11", "k:0 3 5 9", "k:0 5 7 10", "k:0 2 8 9 11"),
       hands="s:6;g:1 4 7 10", unit=32, artists=("Bernard Purdie",))

family("slow twelve-eight blues", "blues", "12/8", 68, "triplet blues",
       "Let the inner triplets breathe around a broad two-and-four backbeat.",
       (("triplet hats","h:0 2 4 6 8 10 12 14 16 18 20 22"),("a ride on the dotted-quarter pulses","r:0 6 12 18"),
            ("a shuffle ride","r:0 4 6 10 12 16 18 22"),("a broken triplet hat line","h:0 4 6 8 12 16 18 20 22")),
       ("k:0 8 12 20", "k:0 4 12 16", "k:0 10 12 22", "k:0 6 12 16 22", "k:0 8 11 15 20"),
       hands="s:6 18;g:10 16",grouping=(3,3,3,3),artists=("Sam Lay",))

family("small-group swing jazz", "jazz", "4/4", 142, "ride swing",
       "Close the foot hat on two and four and feather the kick under the ride.",
       (("a skip after two and four","r:0 3* 5 6 9* 11"),("a skip after four","r:0 3 6 9* 11"),
            ("skips after two and three","r:0 3* 5 6 8 9*"),("four lightly accented ride beats","r:0 3* 6 9*")),
       ("k:0 3 6 9;g:2 8", "k:0 6;g:5 11", "k:0 6 9;g:2 5 10", "k:0 3 8;g:4 7 10", "k:0 6 11;g:2 7 9"),
       hands="p:3 9", unit=32, artists=("Elvin Jones", "Bill Stewart"))

family("jazz waltz", "jazz", "3/4", 139, "waltz swing",
       "Keep the three-beat ride phrase turning, with spare snare comping.",
       (("a skip after beat two","r:0* 3 5 6"),("a skip after beat three","r:0* 3 6 8"),
            ("two trailing skip notes","r:0* 3 5 6 8"),("quarter-note ride","r:0* 3 6")),
       ("k:0 6;g:2 7", "k:0;g:2 5", "k:0 5;g:4 8", "k:0 3;g:2 6", "k:0 8;g:5 7"),
       hands="p:3", unit=32, artists=("Elvin Jones", "Max Roach"))

family("five-four jazz", "jazz", "5/4", 132, "three plus two swing",
       "Keep a long group of three and a short group of two clear on the ride.",
       (("a skip in each group","r:0 3 5 6 9 12 14"),("mostly quarter-note ride","r:0 3 6 9 12"),
            ("two skips in the long group","r:0 2 3 5 6 9 12"),("a light ride answer in the short group","r:0 3 6 9 11 12 14")),
       ("k:0 9;g:2 8 13", "k:0 6 9;g:4 11", "k:0 5 9;g:2 7 12", "k:0 9 14;g:5 8 11", "k:0 2 9 12;g:4 8 13"),
       hands="p:3 12", unit=32, grouping=(3,2), artists=("Max Roach", "Bill Stewart"), variants=2)

family("Afro-Cuban mambo", "latin", "4/4", 118, "three-two son clave",
       "Keep the two-bar 3–2 son clave intact with space around the rim replies.",
       (("closed hats between the clave strokes","v:0 6 12;h:2 4 8 10 14|v:4 8;h:0 2 6 10 12 14"),
            ("ride between clave strokes","v:0 6 12;r:2 4 8 10 14|v:4 8;r:0 2 6 12 14"),
            ("shaker between clave strokes","v:0 6 12;z:2 4 8 10 14|v:4 8;z:0 2 6 10 12 14"),
            ("sparse hats around the clave","v:0 6 12;h:4 8 14|v:4 8;h:0 10 12 14")),
       ("k:6 14", "k:0 6 14|k:6 14", "k:6 10 14|k:6 12 14", "k:4 6 14|k:2 6 14", "k:6 12 15|k:0 6 14"),
       hands="x:4 10;l:7 15|x:2 12;l:5 13", phrase_bars=2, artists=("Horacio Hernández", "Ignacio Berroa"))

family("songo", "latin", "4/4", 108, "two-three clave",
       "Keep the 2–3 clave audible while the floor tom and rim answer the timekeeper.",
       (("bell between the clave hits","v:4 8;c:0 2 6 10 12 14|v:0 6 12;c:2 4 8 10 14"),
            ("sixteenth bell anticipations","v:4 8;c:0 3 6 10 12 15|v:0 6 12;c:2 5 8 11 14"),
            ("a sparse bell on eighths","v:4 8;c:0 6 12 14|v:0 6 12;c:4 8 14"),
            ("lighter closed hats between the clave strokes","v:4 8;h:0 2 6 10 12 14|v:0 6 12;h:2 4 8 10 14")),
       ("k:0 7 10 14|k:0 6 11 14", "k:3 6 10 14|k:0 6 12 15", "k:2 6 8 14|k:0 7 11 14",
        "k:0 5 10 14|k:2 6 11 15", "k:0 6 13 15|k:0 3 10 14"),
       hands="x:3 11;l:6 14|x:4 10;l:5 13", phrase_bars=2, artists=("José Luis Changuito Quintana","Dafnis Prieto"))

family("bossa nova", "latin", "4/4", 124, "straight bossa",
       "Let the rim pattern answer across the two bars over small bass anticipations.",
       (("quiet eighth-note hats","h:0 2 4 6 8 10 12 14"),("an eighth-note ride","r:0 2 4 6 8 10 12 14"),
            ("a shaker pulse","z:0 2 4 6 8 10 12 14"),("a gapped hat pulse","h:0 2 4 6 8 12 14|h:0 4 6 8 10 12 14")),
       ("k:0 6 8 14", "k:0 6 8 14|k:0 6 8 15", "k:0 6 8 15|k:0 6 8 14",
        "k:0 5 8 14|k:0 6 8 14", "k:0 6 8 14|k:0 5 8 15"),
       hands="x:0 5 9 14|x:2 6 11 14", phrase_bars=2)

family("samba-jazz", "latin", "4/4", 138, "samba push",
       "Drive the pulse with bass drum anticipations and let the snare answer across it.",
       (("a light eighth-note ride","r:0 2 4 6 8 10 12 14"),("a shaker keeping eighths","z:0 2 4 6 8 10 12 14"),
            ("a broken ride pattern","r:0 3 4 6 8 11 12 14"),("tight hats on the offbeats","h:2 6 10 14")),
       ("k:0 3 4 7 8 11 12 15;g:1 6 10 13", "k:0 3 4 8 11 12 15;g:2 5 9 14",
        "k:0 3 4 7 8 11 12 15;g:2 6 9 13", "k:0 3 4 7 8 12 15;g:1 5 10 14",
        "k:0 3 4 8 11 12 15;g:1 6 10 13"),
       hands="p:4 12", artists=("Airto Moreira", "Kiko Freitas"))

family("half-time soul", "soul", "4/4", 88, "half time",
       "Hold the soft space between the kicks and lay the main backbeat on three.",
       (("close eighth-note hats","h:0 2 4 6 8 10 12 14"),("a brushed-light ride","r:0 4 8 12"),
            ("an open hat at the middle of the bar","h:0 2 4 6 10 12 14;o:8"),("sparse offbeat hats","h:2 6 10 14")),
       ("k:0 6 13", "k:0 3 10 15", "k:0 2 5 11 14", "k:0 6 9 12 15", "k:0 5 7 14"),
       hands="s:8;g:6 11", lag=2)

family("laid-back neo-soul", "soul", "4/4", 84, "late pocket",
       "Sit just behind two and four, with low snare ghosts and deep kick space.",
       (("short hat notes in pairs","h:0 1 4 5 8 9 12 13"),("a steady eighth-note hat","h:0 2 4 6 8 10 12 14"),
            ("a shaker with small gaps","z:0 2 4 7 8 10 12 15"),("soft eighth-note ride","r:0 2 4 6 8 10 12 14")),
       ("k:0 5 8 11", "k:0 3 8 13", "k:0 2 7 9 14", "k:0 5 10 14", "k:0 3 6 10 15"),
       hands="s:4 12;g:2 7 10", lag=4)

family("disco", "disco", "4/4", 121, "four on the floor",
       "Keep the kick on all four beats and the snare locked to two and four.",
       (("an open hat on every and","o:2 6 10 14"),("closed hats on the beat and open hats on the offbeats","h:0 4 8 12;o:2 6 10 14"),
            ("a tight open-close hat pattern","h:0 3 4 7 8 11 12 15;o:2 6 10 14"),
            ("two open-hat accents","h:0 2 4 8 10 12;o:6 14")),
       ("k:0 4 8 12", "k:0 4 8 12;g:3 11", "k:0 4 8 12;g:7 15", "k:0 4 8 12;g:2 10", "k:0 4 8 12;g:5 13"),
       hands="s:4 12")

family("one-drop reggae", "reggae", "4/4", 80, "one drop",
       "Leave the downbeat kick out and land the kick and rim together on three.",
       (("offbeat hats","h:2 6 10 14"),("light eighths with offbeat tips","h:0 2* 4 6* 8 10* 12 14*"),
            ("an open hat after three","h:0 2 4 6 8 12 14;o:10"),("a quiet offbeat shaker","z:2 6 10 14")),
       ("k:8", "k:8 14", "k:8 11", "k:8;l:2 10", "k:8;g:5 13"),
       hands="x:8", artists=("Carlton Barrett",))

family("rockers reggae", "reggae", "4/4", 91, "rockers pocket",
       "Plant the bass drum on one and three, with the rim click defining three.",
       (("closed hats accented offbeat","h:0 2* 4 6* 8 10* 12 14*"),("an open hat before the backbeat","h:0 2 4 8 10 12 14;o:6"),
            ("a sparse ride pulse","r:0 2 6 8 10 14"),("a tight shaker line","z:0 2 4 6 8 10 12 14")),
       ("k:0 8", "k:0 8 14", "k:0 6 8 13", "k:0 8;l:4 12", "k:0 7 8 11;g:14"), hands="x:8")

family("dusty hip-hop", "hip_hop", "4/4", 90, "loose backbeat",
       "Give the sample-like kick room and lay the backbeats slightly late.",
       (("scuffed eighth-note hats","h:0 2 4 6 8 10 12 14"),("gapped sixteenths","h:0 1 4 6 8 11 12 14"),
            ("hats on the offbeats","h:2 6 10 14"),("a quiet open hat on the last offbeat","h:0 2 4 6 8 10 12;o:14")),
       ("k:0 3 10;g:7", "k:0 6 8 14;g:11", "k:0 5 8 13;g:7 15", "k:0 2 6 11;g:14", "k:0 3 7 8 15;g:6"),
       hands="s:4 12", lag=5)

family("odd-time fusion", "fusion", "7/8", 120, "two plus two plus three",
       "Hear the groups of two, two and three. Keep the kick on the group starts and the low tom inside the ostinato.",
       (("accented eighth-note ride","r:0 2 4 6 8 10 12"),("a syncopated ride line","r:0 3 4 7 8 11 12"),
            ("a hat opening inside the long group","h:0 2 4 6 8 12;o:10"),("quarter accents with a short answer","h:0 4 8 10 13")),
       ("k:0 4 8", "k:0 2 4 8 12", "k:0 4 7 8 11", "k:0 3 4 8 10 13", "k:0 4 6 8 12"),
       hands="s:2 6;l:3 10",grouping=(2,2,3),artists=("Gavin Harrison",),variants=2)

family("short-long five-four jazz", "jazz", "5/4", 125, "two plus three swing",
       "Feel two beats first and three second, with the foot hat answering inside each group.",
       (("spaced ride skips","r:0 3 5 6 9 12 14"),("quarter ride with a light skip","r:0 3 6 8 9 12")),
       ("k:0 6;g:2 8 13", "k:0 5 6;g:4 11", "k:0 6 9;g:2 7 14",
        "k:0 3 6 12;g:5 8 10", "k:0 2 6 11;g:4 8 13"),
       hands="p:3 9", unit=32, grouping=(2,3), artists=("Max Roach","Bill Stewart"), variants=2)

family("long-short-short fusion", "fusion", "7/8", 114, "three plus two plus two",
       "Put the kick at each group start and let the floor tom answer within the long group.",
       (("eighth-note hats with group accents","h:0 2 4 6 8 10 12"),
        ("ride on the groups and offbeats","r:0 3 5 6 8 10 13")),
       ("k:0 6 10", "k:0 4 6 10 12", "k:0 3 6 9 10", "k:0 2 6 8 10 13", "k:0 5 6 10 12"),
       hands="s:4 8 12;l:2 7", grouping=(3,2,2), artists=("Gavin Harrison",), variants=2)

add_families(family)

add_counted_examples(counted_family)
add_repertoire(family)

assert len(rows) == 1330
assert len({row["messages"][1]["content"] for row in rows}) == len(rows)
(ROOT / "train.jsonl").write_text("".join(
    json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n" for row in rows
))
print(Counter(json.loads(row["messages"][2]["content"])["style"] for row in rows))
