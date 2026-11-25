"""Count-focused loops with fixed pulses and instrument-specific response slots."""

from collections import Counter


# Each pocket has a timekeeping figure, backbeat, kick anchors, and response slots.
# Slots are ordered by musical importance. A busier variation uses more answers.
POCKETS = (
    ("syncopated funk", "funk", "4/4", 106, "Sixteenth-note pocket", 24, (), 0,
     "h:0* 2 4 6 8* 10 12 14", "s:4 12", (0, 8), (3, 10, 15, 6, 13, 5),
     (7, 11, 3, 14, 6, 15, 10), "an eighth-note hat line with an accented backbeat"),
    ("late soul", "soul", "4/4", 86, "Laid-back backbeat", 24, (), 3,
     "h:0 2 4 6 8 10 12;o:14", "s:4 12", (0, 10), (7, 14, 3, 8, 5, 11),
     (3, 11, 7, 15, 6, 14, 2), "a late backbeat and a short open hat"),
    ("ride-led rock", "rock", "4/4", 114, "Half time", 24, (), 0,
     "r:0* 2 4 6 8* 10 12 14", "s:8", (0, 6), (10, 14, 7, 3, 12, 5),
     (7, 11, 3, 12, 6, 14, 5), "a steady ride over a beat-three snare"),
    ("dusty hip-hop pocket", "hip_hop", "4/4", 82, "Dilla-inspired push and pull", 24, (), 4,
     "h:0 2 4 7 8 10 12 14", "s:4 12", (0, 7), (10, 3, 14, 5, 8, 13),
     (6, 11, 3, 14, 2, 9, 15), "uneven hat placement and a late backbeat"),
    ("offbeat reggae", "reggae", "4/4", 82, "One-drop pocket", 24, (), 0,
     "h:2 6 10 14", "x:8", (8,), (14, 6, 11, 3, 10), (7, 15, 3, 11, 13, 5),
     "offbeat hats and a kick and rim accent on three"),
    ("open-hat disco", "disco", "4/4", 116, "Four on the floor", 24, (), 0,
     "h:0 4 8 12;o:2 6 10 14", "s:4 12", (0, 4, 8, 12), (), (7, 11, 15, 3, 5, 13),
     "quarter-note kicks and open offbeat hats"),
    ("rolling 6/8 blues", "blues", "6/8", 69, "Two broad pulses", 24, (3, 3), 0,
     "h:0* 2 4 6* 8 10", "s:6", (0,), (4, 8, 10, 2, 5, 7, 11),
     (5, 9, 3, 11, 1, 7), "two broad pulses and a snare at the second pulse"),
    ("slow 12/8 blues", "blues", "12/8", 61, "Four compound pulses", 24, (3, 3, 3, 3), 0,
     "r:0* 2 4 6* 8 10 12* 14 16 18* 20 22", "s:6 18", (0, 12), (10, 16, 22, 5, 8, 15, 20),
     (5, 11, 17, 23, 3, 9, 15), "a rolling ride and snare on the second and fourth compound pulses"),
    ("small-group jazz waltz", "jazz", "3/4", 142, "Swing waltz", 32, (), 0,
     "r:0* 3 5 6 8", "p:3 6", (0,), (5, 8, 3, 2, 7), (2, 4, 7, 5, 1, 8),
     "a triplet ride skip and light foot hats"),
    ("small-group swing jazz", "jazz", "4/4", 146, "Ride swing", 32, (), 0,
     "r:0 3 5 6 9 11", "p:3 9", (0, 6), (5, 11, 8, 2, 9), (2, 4, 7, 10, 5, 8),
     "a swung ride and foot hats on two and four"),
    ("soft bossa nova", "latin", "4/4", 110, "Straight bossa", 24, (), 0,
     "h:0 2 4 6 8 10 12 14", "x:0 6 12", (0, 6, 8, 14), (4, 10, 2, 12),
     (3, 7, 11, 15, 5, 9), "soft hats, a repeating rim phrase and a bossa bass figure"),
    ("short-short-long fusion", "fusion", "7/8", 114, "Two plus two plus three", 24, (2, 2, 3), 0,
     "h:0 2 4 6 8 10 12", "s:4 10", (0, 4, 8), (11, 3, 13, 5, 9, 2, 7),
     (3, 7, 11, 13, 1, 5, 9), "an accented 2+2+3 pulse with a recurring snare answer"),
)

NAMES = {"kick": "kick hits", "ghost": "ghost notes", "ride": "ride hits",
         "open_hat": "open hi-hat hits", "rim": "rim clicks", "crash": "crashes",
         "foot_hat": "foot hi-hat hits"}
NUMBERS = ("zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
           "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen",
           "seventeen", "eighteen", "nineteen", "twenty")


def add_counted_examples(emit):
    for pocket_index, pocket in enumerate(POCKETS):
        (name, style, meter, bpm, feel, unit, grouping, lag, top, hands, kick_base,
         kick_replies, ghost_replies, character) = pocket
        for variant in range(20):
            group, response = divmod(variant, 5)
            bars = (4, 2, 3, 1)[group]
            if group == 3 and response == 4:
                bars = 2
            numbered = ("kick", "ghost", "kick", "ghost", "ghost")[response]
            if response == 4:
                numbered = {
                    "rock": "ride", "disco": "open_hat", "hip_hop": "kick",
                    "reggae": "rim", "jazz": "foot_hat", "latin": "rim",
                }.get(style, "ghost")
            extra_kicks = response + (group % 2)
            ghost_density = (4, 2, 3, 1, 5)[response]
            kick_limit = min(len(kick_replies), extra_kicks)
            cells = []
            for bar in range(bars):
                shift = (group + bar) % len(ghost_replies)
                ghosts = (ghost_replies[shift:] + ghost_replies[:shift])[:ghost_density]
                if variant == 2:
                    ghosts = (ghost_replies[shift:] + ghost_replies[:shift])[:(4 if bar < 3 else 3)]
                if variant == 4 and style == "hip_hop" or variant == 19 and style == "rock":
                    kicks = ()
                else:
                    # Keep downbeat anchors and alternate where the extra replies land.
                    kick_pool = kick_replies
                    if kick_pool and (bar + group) % 2 == 1:
                        kick_pool = kick_pool[1:] + kick_pool[:1]
                    kicks = kick_base + kick_pool[:kick_limit]
                    if variant == 2 and style == "funk":
                        kicks = kick_base + kick_pool[:3]  # 20 kicks and 15 ghost notes.
                cell = [top, hands]
                if kicks:
                    cell.append("k:" + " ".join(map(str, sorted(kicks))))
                if ghosts:
                    cell.append("g:" + " ".join(map(str, sorted(ghosts))))
                # A repeated crash on the first written beat is part of the pocket.
                if variant == 3 and style in ("rock", "fusion"):
                    symbol, positions = cell[0].split(";")[0].split(":")
                    remaining = " ".join(positions.split()[1:])
                    cell[0] = f"a:0;{symbol}:{remaining}"
                cells.append(";".join(cell))
            metrics = [numbered]
            if response in (2, 3) and numbered != "kick":
                metrics.append("kick")
            if response == 2:
                metrics.append("ghost")
            if variant == 3 and style in ("rock", "fusion"):
                metrics.append("crash")
            if variant == 19 and style == "rock":
                metrics.append("kick")
            if response == 4 and style == "soul":
                metrics.append("open_hat")
            metrics = list(dict.fromkeys(metrics))
            emit(name, style, meter, bpm + (0, -2, 3, 1, -3)[response], feel,
                 unit, grouping, lag, character, cells, metrics, variant)


def count_line(counts, variant):
    labels = []
    for voice, by_bar in counts.items():
        per_bar = variant % 5 in (1, 4) and len(by_bar) > 1 and len(set(by_bar)) == 1
        n = by_bar[0] if per_bar else sum(by_bar)
        number = NUMBERS[n] if variant % 3 == 1 and n < len(NUMBERS) else str(n)
        name = NAMES[voice]
        if voice == "kick" and variant % 3 == 2:
            name = "bass hits"
        if n == 1:
            name = "crash" if voice == "crash" else name.removesuffix("s")
        labels.append(f"{number} {name}" + (" per bar" if per_bar else ""))
    return " and ".join(labels)


def counted_events(events, metrics, bar_ticks, bars):
    buckets = []
    for bar in range(bars):
        counters = Counter()
        for hit in events:
            if hit["tick"] // bar_ticks == bar:
                instrument = hit["instrument"]
                counters[instrument] += 1
                if instrument == "snare" and hit["velocity"] <= 55:
                    counters["ghost"] += 1
        buckets.append(counters)
    return {voice: [b[voice] for b in buckets] for voice in metrics}
