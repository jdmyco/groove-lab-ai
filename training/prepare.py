"""Export deterministic MLX LM splits without sharing groove families between them."""

import argparse
from collections import defaultdict
from hashlib import sha256
import json
from pathlib import Path


def prepare(output):
    source = Path(__file__).with_name("train.jsonl")
    rows = []
    counted_families = defaultdict(set)
    families = defaultdict(set)
    for line in source.read_text().splitlines(keepends=True):
        row = json.loads(line)
        score = json.loads(row["messages"][-1]["content"])
        style = score["style"]
        label = row["groove"]["family"]
        family = label.removesuffix(" with exact counts")
        if family != label:
            counted_families[style].add(family)
        families[style].add(family)
        rows.append(((style, family), line))

    # Hold out one entire family per style for each evaluation split.
    # Reserve counted families in both held-out splits.
    split = {}
    for style, names in families.items():
        ordered = sorted(names, key=lambda name: sha256(f"{style}/{name}".encode()).digest())
        counted = [name for name in ordered if name in counted_families[style]]
        ordinary = [name for name in ordered if name not in counted_families[style]]
        valid, test = ordinary[:2]
        if style in ("blues", "jazz"):
            valid = counted[0]
        elif style in ("disco", "funk", "hip_hop"):
            test = counted[0]
        split[style, valid] = "valid"
        split[style, test] = "test"

    output.mkdir(parents=True)
    for name in ("train", "valid", "test"):
        selected = [line for family, line in rows if split.get(family, "train") == name]
        (output / f"{name}.jsonl").write_text("".join(selected))
        print(f"{name}: {len(selected)} examples")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path, help="New directory for the data splits")
    prepare(parser.parse_args().output)
