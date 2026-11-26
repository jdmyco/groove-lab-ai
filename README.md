# Groove Lab

Groove Lab is a local drum-groove gallery and comparison app. One Go server serves the gallery, the 1,240-example fine-tuning corpus, and procedural and local-model generation.

## Run the gallery

```sh
mise install
go run .
```

Open [localhost:8080](http://127.0.0.1:8080). Pick a groove from the library to open the drum sheet, or choose Preview to hear its loop. Written onsets stay on the staff grid while playback uses the score's timing offsets. The score header gives the meter, grouping, tempo, and bar count. Description, feel, and timing appear below the sheet and in the score JSON's `metadata` object.

## Generate and compare

The gallery defaults to **Procedural**. The same prompt yields the same score from the embedded catalog and the requested edits. The matcher recognizes styles, artist names, common feel and instrument terms, including exclusions, and explicit BPM, meter, groups, and bar counts. It can retime a groove and resize it in whole pattern cycles. The expanded catalog adds pockets such as motorik rock, bebop, partido alto, steppers, and short-long-short fusion. For example, a two-bar clave cycle stays intact when expanded to four bars. Requests that have no matching catalog entry return an explanation.

```sh
curl -sS http://127.0.0.1:8080/api/generate \
  -H 'Content-Type: application/json' \
  -d '{"engine":"procedural","prompt":"Two bars of 7/8 fusion grouped 3+2+2 at 116 BPM."}'
```

With **Target** set to **Automatic**, follow-ups such as “Make it 200 BPM” edit the selected groove. Choose **Current groove** to force an edit or **New groove** to start fresh. API callers can pass the previous score as `base_score` with `"engine":"procedural"`.

- A direction such as “crash on the 3” places the hit on the third counted beat of each bar. Lists such as “bass drum on 1, 2, 3 and 4” place one hit on each named beat. “Remove crash on 3” clears that placement.
- “Make it swung” shifts the eighth-note offbeat in quarter-note meters. “Add a doublestroke on the downbeat” adds two snare strokes by default. Name an instrument to use that voice instead.
- Ask for “15 ghost notes”, “20 bass hits”, or “5 kicks per bar and 12 ghost notes total”. Exact counts apply to the full loop unless you specify “per bar”. A bass hit is a kick. A ghost note is a snare stroke with velocity 55 or lower and contributes to any snare total. The procedural engine preserves explicit beat placements and reports when the counts cannot fit. The local model gets another attempt if its returned score has the wrong count.
- Multiple directions can go in one prompt. Explicit placements take priority, and the engine removes conflicting original strokes when necessary to keep the result playable. The score description reports any simplification.

Choose **Local LLM** in the gallery or send `"engine":"llm"` to use an OpenAI-compatible chat endpoint. Omitting the engine from an API request also selects the LLM. Both engines return a [GrooveScoreV1 object](score.schema.json), sorted by written tick and instrument.

The default model endpoint is `http://127.0.0.1:8765/v1/chat/completions`, with `model: "default_model"` to select the model configured in MLX LM. Override `LLM_URL`, `LLM_MODEL`, or the gallery's `PORT` when starting Go.

```sh
mlx_lm.server --model mlx-community/Qwen3-4B-4bit --port 8765 \
  --chat-template-args '{"enable_thinking":false}'
```

Pass `--adapter-path PATH` to the model server once the adapter is available. The Go app expects JSON score text in `choices[0].message.content` and feeds validation errors back to the model for up to five attempts in total.

## FormKit score and sound

The library and player reuse FormKit's dark and light palettes, Inter type, Bravura-based music glyphs, SVG drum-score renderer, sample-clock worklet, and Virtuosity Drums kit. GrooveScoreV1 written groups set the staff beaming in compound and mixed eighth-note meters. The audio clock uses each hit's sounding onset after its timing offset. A middle tom has its own staff position and uses the kit's low-tom sample.

The checked-in assets are bundled from [the pinned FormKit revision](assets/formkit/SOURCE.txt) and served by Go. To refresh them from a FormKit checkout with its locked dependencies installed, run `node formkit/sync.mjs ~/workspace/formkit`. The Go server runs independently of that checkout. [Acoustic kit credits](assets/formkit/audio/CREDITS.md) · [Music font license](assets/formkit/BopkitMusic-LICENSE.txt)

## Fine-tuning corpus

[training/train.jsonl](training/train.jsonl) contains 1,240 original loop arrangements in 65 families across rock, funk, blues, jazz, Latin, soul, disco, reggae, hip-hop, and fusion. It covers 3/4, 4/4, 5/4, 6/8, 7/8, 7/4, and 12/8. Every example has a practice-length phrase and a stable timekeeping part. Toms occur as part of recurring rhythmic figures. The 5/4 and 7/8 families each have two accent groupings. Search the gallery for **exact counts** to see the 240 count-focused presets. They cover whole-loop and per-bar counts, zero hits, and combinations of multiple drum voices.

FormKit's `src/music/groove-schema.ts`, `groove-playability.ts`, and genre, Latin, and blues studies informed the written/playback timing split, the kit voices, and the drum-set checks. The score uses a 96-tick quarter-note grid. Optional `grouping` counts are measured in the meter's denominator units, and `offset` moves the audible onset by up to six ticks. Older v1 scores with overlapping hand cymbals may need their hat part revoiced as foot hats for the current playability checks.

Artist tags describe stylistic references for newly arranged drum parts. The corpus draws on these references.

- Steve Gadd's rudimental pocket and Bernard Purdie's half-time shuffle. [Gadd's rudimental lessons](https://hudsonmusic.com/product/rudimental-concepts-part-2/) · [Purdie's shuffle lesson](https://www.drumeo.com/beat/bernard-purdie-drum-lessons/)
- David Garibaldi's displaced backbeat and Zigaboo Modeliste's New Orleans funk. [Garibaldi's Drumeo interview](https://www.drumeo.com/beat/david-garibaldi-drumeo-interview-podcast-22/) · [Modeliste's artist biography](https://www.zigaboo.com/press-kit/)
- Horacio Hernández's clave-centered independence and Airto Moreira's Brazilian vocabulary. [Conversations in Clave](https://hudsonmusic.com/product/conversations-in-clave/) · [Airto Moreira interview](https://daily.redbullmusicacademy.com/2017/06/interview-airto-moreira/)
- The expanded Latin set uses a seven-stroke bembé bell cycle and a seven-in-sixteen partido alto rim timeline. [Traditional timeline analysis](https://ehess.modelisationsavoirs.fr/atiam/biblio/Toussaint-BRIDGES2005.pdf)
- Gavin Harrison's accent groupings and Carlton Barrett's one-drop practice. [Harrison's groupings lesson](https://www.drumeo.com/beat/?p=7967) · [Carlton Barrett profile](https://thereggaemuseum.com/musician/carlton-barrett/)

The JSONL uses [MLX LM chat records](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md#data). Each row contains a shared system message, a natural-language request, and the compact score JSON as the assistant message. The score's `metadata` describes the beat, feel, and timing. The JSONL row's separate `groove` object identifies the catalog family, sound, influences, and cycle length used by the matcher. The folder layout is small.

- `index.html` is the playable gallery and generator.
- `main.go` runs the Go API and calls the local model. `procedural.go`, `edits.go`, and `counts.go` implement catalog matching, beat edits, and exact counts.
- `score.go` and `score.schema.json` define and validate the shared score.
- `training/build.py` rebuilds the presets from authored parts, with more families in `training/expansion.py` and the count exercises in `training/counted.py`. `training/prepare.py` exports the training and evaluation splits.
- `assets/formkit/` contains the notation, styling, and sounds that the gallery serves. `formkit/` contains the adapter and asset sync script.

### Train an adapter with MLX LM

On Apple Silicon, run these from the repository root with `uv` installed. The first command creates 860 training, 190 validation, and 190 test examples while holding each groove family together. The second starts a 300-step QLoRA run on quantized Qwen.

```sh
# Skip this line if you already prepared the splits.
python3 training/prepare.py .groove-training/data
uv run --no-project --python 3.13 --with 'mlx-lm[train]' mlx_lm.lora \
  --model mlx-community/Qwen3-4B-4bit --train \
  --data .groove-training/data --mask-prompt --iters 300 \
  --batch-size 1 --max-seq-length 4096 --grad-checkpoint \
  --adapter-path .groove-training/adapters
```

MLX writes `adapters.safetensors` and `adapter_config.json` to `.groove-training/adapters/`. To keep a previous run, give the new run a different `--adapter-path` and use that path when serving.

### Compare base and adapted Qwen

Run these in separate terminals from the repository root. Start Go with `go run .` in a third terminal, then open the [comparison page](http://127.0.0.1:8080/compare).

**Terminal 1 — base Qwen on port 8766**

```sh
uv run --no-project --python 3.13 --with mlx-lm mlx_lm.server \
  --model mlx-community/Qwen3-4B-4bit --host 127.0.0.1 --port 8766 \
  --chat-template-args '{"enable_thinking":false}'
```

**Terminal 2 — Qwen with the adapter on port 8767**

```sh
uv run --no-project --python 3.13 --with mlx-lm mlx_lm.server \
  --model mlx-community/Qwen3-4B-4bit --host 127.0.0.1 --port 8767 \
  --adapter-path .groove-training/adapters \
  --chat-template-args '{"enable_thinking":false}'
```

The comparison page sends the same prompt and JSON-format example to both models and displays their scores and attempt counts independently. Go validates each score and checks requested hit counts. For an invalid score, it sends validation feedback back to that model for up to five attempts total. The gallery's **Local LLM** selector uses its own endpoint on port 8765 by default. Keep the model ID at `default_model` for these MLX servers so MLX selects the adapter loaded on port 8767. For other servers, set `LLM_BASE_URL` or `LLM_ADAPTER_URL` to change a comparison endpoint and `LLM_BASE_MODEL` or `LLM_ADAPTER_MODEL` if it requires a different model ID.

Use `.groove-training/data/test.jsonl` to evaluate the adapter on families held out from training. The procedural engine has the full catalog, including those families. For a fair head-to-head comparison, write a shared set of new prompts outside the catalog and measure exact-count accuracy, score validity, and how well each loop sounds. `--mask-prompt` trains on the score completion. See the [MLX LM fine-tuning guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md).

## Checks

```sh
python3 -B training/build.py
go test -race ./...
go vet ./...
go build ./...
```

The corpus builder can regenerate the JSONL from the authored cycles. Its checks examine hand and foot overlaps, stroke spacing across the repeat, and tom placement. Go also checks the embedded training rows against the score contract and tests both generation paths.
