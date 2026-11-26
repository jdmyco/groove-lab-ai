"""Vary prompt phrasing while keeping each canonical score and its family fixed."""

from collections import Counter


# Each short prompt is paired with a matching score in the named training family.
ANCHORS = {
    "bossa nova": "Write a bossa nova latin groove",
    "Purdie shuffle": "Give me a Purdie shuffle",
    "Afro-Cuban mambo": "give me a mamba",
    "disco": "give me a disco groove",
    "sixteenth-note ghost funk": "Write a funky groove at 80 bpm with sixteenth notes on the hihat and ghost notes",
    "single paradiddle funk": "Write a slow paradiddle groove with accented half notes",
    "Mozambique latin kit": "Can you make a Mozambique groove for drum kit?",
    "cáscara latin kit": "Write a two-bar cascara groove",
    "baião latin kit": "Make a Brazilian baiao drum groove",
    "choro latin kit": "Give me a light Brazilian choro beat",
    "soca carnival latin kit": "Give me a danceable soca groove",
    "boogaloo latin kit": "Make a boogaloo pocket with a bell pulse and a backbeat",
}


def prompt_variants(row, score):
    name = row["groove"]["family"].removesuffix(" with exact counts")
    color = row["groove"]["color"]
    meter = score["meter"]
    grouping = "+".join(map(str, score.get("grouping", ())))
    grouped_meter = f"{meter} grouped {grouping}" if grouping else meter
    bpm = score["tempo"]["bpm"]
    tempo = f"{bpm} dotted-quarter BPM" if score["tempo"]["beat_unit"] == "dotted_quarter" else f"{bpm} BPM"
    length = ("", "one bar", "two bars", "three bars", "four bars")[score["bars"]]
    counts = Counter(e["instrument"] for e in score["events"])
    ghosts = sum(e["instrument"] == "snare" and e["velocity"] <= 55 for e in score["events"])
    n = counts["kick"]
    kick = f"{n} kick hit" if n == 1 else f"{n} kick hits"
    if ghosts:
        count = f"{kick} and {ghosts} snare ghost " + ("note" if ghosts == 1 else "notes")
    else:
        count = kick

    return (
        f"Can you write a {name} drum part at {tempo} in {grouped_meter}? Use {color} for {length}.",
        f"I want a {name} pocket with {color}. Keep it to {length} in {grouped_meter} at {tempo}.",
        f"Loop {length} of {name}. Think {color}, {grouped_meter}, {tempo}.",
        f"Set the tempo to {tempo}. In {grouped_meter}, give me {length} of {name} with {color}.",
        f"I need a {name} loop for practice. Keep {color}, take {length}, and play it at {tempo} in {grouped_meter}.",
        f"Try {color} for {length} of {name} and make it loop.",
        f"Play {length} of {name} at {tempo}. I want {color}. The time is {grouped_meter}.",
        f"Compose a repeating {name} beat in {grouped_meter}. Give it {color} and a tempo of {tempo}.",
        f"Drum part for a {name} feel at {tempo}. Use {length}, {grouped_meter}, and {color}.",
        f"How about a {name} groove with {color} at {tempo}? It should repeat after {length} in {grouped_meter}.",
        f"Give me something to practice with {color}, in a {name} pocket. {length.capitalize()} at {tempo}, {grouped_meter}.",
        f"Give me a {name} drum loop with exactly {count} in the whole loop. Make it {length} at {tempo} in {grouped_meter}.",
        f"Fit {count} into {length} of {name} in {grouped_meter}. Keep {color} and set it to {tempo}.",
        f"Could you make {length} of {name} with {color} and {count}? Tempo {tempo}, {grouped_meter}.",
        f"Make it {name}, {grouped_meter}, {tempo}. Keep {color} for {length}.",
        f"I'm after {color} in a {name} groove. Write {length} in {grouped_meter} to loop at {tempo}.",
        f"Give me a {name} beat at {tempo}. Keep {color} and limit it to {length}.",
        f"{length.capitalize()} of {name} with {color}, please. Use {grouped_meter} and {tempo}.",
    )


def with_prompt(row, prompt):
    system, _, assistant = row["messages"]
    return {**row, "messages": [system, {"role": "user", "content": prompt}, assistant]}


def augment(train, reserved, total):
    used = {p.strip().casefold() for p in reserved}
    results = []
    pending_anchors = dict(ANCHORS)

    for row, score in train:
        family = row["groove"]["family"]
        if family not in pending_anchors:
            continue
        if family == "sixteenth-note ghost funk" and score["tempo"]["bpm"] != 80:
            continue
        prompt = pending_anchors.pop(family)
        results.append(with_prompt(row, prompt))
        used.add(prompt.casefold())
    if pending_anchors:
        raise ValueError(f"Missing training families for example prompts: {sorted(pending_anchors)}")

    # Rotate wording, then take one candidate per score per pass. Sparse or
    # repeated descriptions can collide, so reserve each prompt exactly once.
    streams = []
    for i, (row, score) in enumerate(train):
        variants = prompt_variants(row, score)
        start = i % len(variants)
        streams.append((row, iter(variants[start:] + variants[:start])))
    while streams and len(results) < total:
        active = []
        for row, candidates in streams:
            for prompt in candidates:
                key = prompt.casefold()
                if key in used:
                    continue
                used.add(key)
                results.append(with_prompt(row, prompt))
                active.append((row, candidates))
                break
            if len(results) == total:
                break
        streams = active
    if len(results) != total:
        raise ValueError(f"Only {len(results)} unique additional prompts were available")
    return results
