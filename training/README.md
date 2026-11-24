# GrooveScore seed examples

[seed-scores.jsonl](seed-scores.jsonl) contains eight original, hand-specified drum patterns in [MLX LM chat format](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md#data). Each row pairs a natural-language brief with a compact [GrooveScoreV1](../schemas/groove.schema.json) JSON string. Use these as seed examples for supervised fine-tuning and as a format smoke test. For a quality comparison, add distinct grooves and hold out whole pattern families, including any tempo variants and prompt paraphrases.

For an MLX LM training smoke test, copy the seed file to `train.jsonl` in a scratch directory and pass that directory to `--data`. Use `--mask-prompt` to compute loss on the assistant completion. Keep the system instruction consistent across fine-tuning and inference.

| Row | Style | Meter | Bars | What to listen for |
| --- | --- | --- | --- | --- |
| 1 | Rock | 4/4 | 1 | Straight eighths and an extra kick on the and of 3 |
| 2 | Rock | 4/4 | 2 | Steady backbeat followed by a descending tom fill |
| 3 | Funk | 4/4 | 1 | Syncopated kick, low snare ghosts, and a single open hat |
| 4 | Funk | 4/4 | 2 | Late snare, a kick pickup into bar two, and a lighter answer |
| 5 | Blues | 4/4 | 2 | Long-short shuffle hats and a snare pickup |
| 6 | Blues | 6/8 | 2 | Two dotted-quarter pulses per bar at 62 BPM |
| 7 | Jazz | 4/4 | 2 | Ride skip notes, light kick, and sparse snare comping |
| 8 | Jazz | 5/4 | 2 | Ride accents grouped 3+2 and second-bar comping |

Extract the first score from the repository root.

```sh
sed -n '1p' training/seed-scores.jsonl | jq '.messages[-1].content | fromjson' > /tmp/groove-rock.json
```

Open [the demo player](../web/player.html) and select the exported file. The player accepts 4/4 at 96 ticks per quarter. Its grid shows eighth-note positions for kick, snare, and closed hat. Playback schedules off-grid hits, and all instruments other than kick and snare share one noise sound. Use a renderer with separate drum sounds and the correct tempo unit to audition the full set.
