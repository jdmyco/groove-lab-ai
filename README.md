# Groove Lab

Generate, inspect, and play drum grooves in your browser. Groove Lab includes a 1,330-groove catalog, a procedural generator, and a comparison page for base Qwen and a fine-tuned MLX adapter.

## Quick start

Use Go 1.27.3 or later. The commands below use `mise` to install the pinned Go version. If you already have Go installed, skip that step.

```sh
git clone https://github.com/jdmyco/groove-lab-ai.git
cd groove-lab-ai
mise install
go run .
```

Open the [groove gallery](http://127.0.0.1:8080). You can browse, play, and generate grooves with the procedural engine as soon as Go is running. The [model comparison](http://127.0.0.1:8080/compare) uses two local MLX servers.

Try one of these prompts in the gallery or comparison page.

- Write a Purdie shuffle at 84 BPM with exactly six ghost notes.
- Write a soft bossa nova with syncopated cross-stick.
- Make me a funky groove.

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

Go serves assets bundled from the pinned FormKit revision. To refresh them from a FormKit checkout with its locked dependencies installed, run `node formkit/sync.mjs /path/to/formkit`. Bundled assets carry their own credits and notices. [FormKit revision](assets/formkit/SOURCE.txt) · [Acoustic kit credits](assets/formkit/audio/CREDITS.md) · [Font and library notices](assets/formkit)

## Fine-tuning corpus

[training/train.jsonl](training/train.jsonl) contains 1,330 original loop arrangements in 74 families across rock, funk, blues, jazz, Latin, soul, disco, reggae, hip-hop, and fusion. It covers 3/4, 4/4, 5/4, 6/8, 7/8, 7/4, and 12/8. Every example has a practice-length phrase and a stable timekeeping part. Toms occur as part of recurring rhythmic figures. The 5/4 and 7/8 families each have two accent groupings. Search the gallery for **exact counts** to see the 240 count-focused presets. They cover whole-loop and per-bar counts, zero hits, and combinations of multiple drum voices.

FormKit's `src/music/groove-schema.ts`, `groove-playability.ts`, and genre, Latin, and blues studies informed the written/playback timing split, the kit voices, and the drum-set checks. The score uses a 96-tick quarter-note grid. Optional `grouping` counts are measured in the meter's denominator units, and `offset` moves the audible onset by up to six ticks. Older v1 scores with overlapping hand cymbals may need their hat part revoiced as foot hats for the current playability checks.

Artist tags describe stylistic references for newly arranged drum parts. The corpus draws on these references.

- Steve Gadd's rudimental pocket and Bernard Purdie's half-time shuffle. [Gadd's rudimental lessons](https://hudsonmusic.com/product/rudimental-concepts-part-2/) · [Purdie's shuffle lesson](https://www.drumeo.com/beat/bernard-purdie-drum-lessons/)
- David Garibaldi's displaced backbeat and Zigaboo Modeliste's New Orleans funk. [Garibaldi's Drumeo interview](https://www.drumeo.com/beat/david-garibaldi-drumeo-interview-podcast-22/) · [Modeliste's artist biography](https://www.zigaboo.com/press-kit/)
- Horacio Hernández's clave-centered independence and Airto Moreira's Brazilian vocabulary. [Conversations in Clave](https://hudsonmusic.com/product/conversations-in-clave/) · [Airto Moreira interview](https://daily.redbullmusicacademy.com/2017/06/interview-airto-moreira/)
- The expanded Latin set uses a seven-stroke bembé bell cycle and a seven-in-sixteen partido alto rim timeline. [Traditional timeline analysis](https://ehess.modelisationsavoirs.fr/atiam/biblio/Toussaint-BRIDGES2005.pdf)
- Gavin Harrison's accent groupings and Carlton Barrett's one-drop practice. [Harrison's groupings lesson](https://www.drumeo.com/beat/?p=7967) · [Carlton Barrett profile](https://thereggaemuseum.com/musician/carlton-barrett/)

The new repertoire includes a two-bar Purdie-style shuffle, slow single paradiddles with half-note accents, and continuous sixteenth-note hi-hat funk at 80 BPM. The Latin and Caribbean arrangements include Mozambique, cáscara, baião, choro, soca, and boogaloo. The sources informed the groove characteristics and kit orchestration. [Bernard Purdie’s shuffle lesson](https://www.drumeo.com/beat/bernard-purdie-drum-lessons/) · [Yamaha’s paradiddle lesson](https://hub.yamaha.com/drums/d-how-to/practice-exercises-for-the-beginning-drummer/) · [Juan Carlito Mendoza’s Latin kit lesson](https://www.drumeo.com/beat/5-latin-grooves-every-drummer-should-know/) · [Edú Ribeiro’s Brazilian drumming course](https://www.openstudiojazz.com/courses/brazilian-jazz-drumming/) · [Adam Kuns’s soca lesson](https://www.youtube.com/watch?v=pTdjGyqiJFc) · [Joe Bataan’s Latin soul interview](https://www.redbullmusicacademy.com/lectures/joe-bataan-extraordinary-joe/)

The JSONL uses [MLX LM chat records](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md#data). Each row contains a shared system message, a natural-language request, and the compact score JSON as the assistant message. The score's `metadata` describes the beat, feel, and timing. The JSONL row's separate `groove` object identifies the catalog family, sound, influences, and cycle length used by the matcher. The folder layout is small.

- `index.html` is the playable gallery and generator.
- `main.go` runs the Go API and calls the local model. `procedural.go`, `edits.go`, and `counts.go` implement catalog matching, beat edits, and exact counts.
- `score.go` and `score.schema.json` define and validate the shared score.
- `training/build.py` rebuilds the presets from authored parts in `training/expansion.py`, `training/repertoire.py`, and `training/counted.py`. `training/prepare.py` exports the splits and uses `training/paraphrases.py` to add varied requests for the same training scores.
- `assets/formkit/` contains the notation, styling, and sounds that the gallery serves. `formkit/` contains the adapter and asset sync script.

### Train an adapter with MLX LM

On Apple Silicon, run these from the repository root with `uv` installed. The first command creates 10,950 training, 190 validation, and 190 test examples. It assigns groove families to splits before adding 10,000 prompt variations to the training split. The second starts a 1,000-step QLoRA run on quantized Qwen. With batch size one, a full pass through the expanded training set is 10,950 steps.

```sh
# Skip this line if you already prepared the splits.
python3 -B training/prepare.py .groove-training/data-v2
uv run --no-project --python 3.13 --with 'mlx-lm[train]' mlx_lm.lora \
  --model mlx-community/Qwen3-4B-4bit --train \
  --data .groove-training/data-v2 --mask-prompt --iters 1000 \
  --batch-size 1 --max-seq-length 4096 --grad-checkpoint \
  --adapter-path .groove-training/adapters-v2
```

MLX writes `adapters.safetensors` and `adapter_config.json` to `.groove-training/adapters-v2/`. Use a fresh `--adapter-path` for each run you want to keep. To serve an exported adapter, put those two files in one directory and pass that directory as `--adapter-path`. Load it with the same base checkpoint used for training.

Generated training splits, adapter weights, and checkpoints live under the ignored `.groove-training/` directory. The checked-in catalog and scripts can regenerate the dataset.

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
  --adapter-path .groove-training/adapters-v2 \
  --chat-template-args '{"enable_thinking":false}'
```

The comparison page sends the same prompt and JSON-format example to both models and displays their scores and attempt counts independently. For generated scores, Go rounds fractional ticks, clips positions one tick over an edge, removes repeated downbeats at the loop boundary, maps familiar instrument labels such as `hi_hat`, and drops unusable events. The page reports repairs. The repaired score still has to pass playability checks and match requested hit counts. For an invalid score, it sends validation feedback back to that model for up to five attempts total. The gallery's **Local LLM** selector uses its own endpoint on port 8765 by default. Keep the model ID at `default_model` for these MLX servers so MLX selects the adapter loaded on port 8767. For other servers, set `LLM_BASE_URL` or `LLM_ADAPTER_URL` to change a comparison endpoint and `LLM_BASE_MODEL` or `LLM_ADAPTER_MODEL` if it requires a different model ID.

Use `.groove-training/data-v2/test.jsonl` to evaluate the adapter on families held out from training. The procedural engine has the full catalog, including those families. For a fair head-to-head comparison, write a shared set of new prompts outside the catalog and measure exact-count accuracy, score validity, and how well each loop sounds. `--mask-prompt` trains on the score completion. See the [MLX LM fine-tuning guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md).

## Checks

```sh
python3 -B training/build.py
go test -race ./...
go vet ./...
go build ./...
```

The corpus builder can regenerate the JSONL from the authored cycles. Its checks examine hand and foot overlaps, stroke spacing across the repeat, and tom placement. Go also checks the embedded training rows against the score contract and tests both generation paths.
