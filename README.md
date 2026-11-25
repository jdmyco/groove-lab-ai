# Groove Lab

Groove Lab is a local drum-groove gallery and comparison app. One Go server serves the gallery, the 500-example fine-tuning corpus, and procedural and local-model generation.

## Run the gallery

```sh
mise install
go run .
```

Open [localhost:8080](http://127.0.0.1:8080). Pick a groove from the library to open the drum sheet, or choose Preview to hear its loop. Written onsets stay on the staff grid while playback uses the score's timing offsets. The score header gives the meter, grouping, tempo, and bar count. Description, feel, and timing appear below the sheet and in the score JSON's `metadata` object.

## Generate and compare

The gallery defaults to **Procedural**. The same prompt yields the same score from the embedded catalog. The matcher recognizes styles, artist names, common feel and instrument terms, including exclusions, and explicit BPM, meter, groups, and bar counts. It can retime a groove and resize it in whole pattern cycles. For example, a two-bar clave cycle stays intact when expanded to four bars. Requests that have no matching catalog entry return an explanation.

```sh
curl -sS http://127.0.0.1:8080/api/generate \
  -H 'Content-Type: application/json' \
  -d '{"engine":"procedural","prompt":"Two bars of 7/8 fusion grouped 3+2+2 at 116 BPM."}'
```

Choose **Local LLM** in the gallery or send `"engine":"llm"` to use an OpenAI-compatible chat endpoint. Omitting the engine from an API request also selects the LLM. Both engines return a [GrooveScoreV1 object](score.schema.json), sorted by written tick and instrument.

The default model endpoint is `http://127.0.0.1:8765/v1/chat/completions`, with `model: "default_model"` to select the model configured in MLX LM. Override `LLM_URL`, `LLM_MODEL`, or the gallery's `PORT` when starting Go.

```sh
mlx_lm.server --model mlx-community/Qwen3-4B-4bit --port 8765 \
  --chat-template-args '{"enable_thinking":false}'
```

Pass `--adapter-path PATH` to the model server once the adapter is available. The Go app expects JSON score text in `choices[0].message.content` and feeds validation errors back to the model for up to three attempts in total.

## FormKit score and sound

The library and player reuse FormKit's dark and light palettes, Inter type, Bravura-based music glyphs, SVG drum-score renderer, sample-clock worklet, and Virtuosity Drums kit. GrooveScoreV1 written groups set the staff beaming in compound and mixed eighth-note meters. The audio clock uses each hit's sounding onset after its timing offset. A middle tom has its own staff position and uses the kit's low-tom sample.

The checked-in assets are bundled from [the pinned FormKit revision](assets/formkit/SOURCE.txt) and served by Go. To refresh them from a FormKit checkout with its locked dependencies installed, run `node formkit/sync.mjs ~/workspace/formkit`. The Go server runs independently of that checkout. [Acoustic kit credits](assets/formkit/audio/CREDITS.md) · [Music font license](assets/formkit/BopkitMusic-LICENSE.txt)

## Fine-tuning corpus

[training/train.jsonl](training/train.jsonl) contains 500 original loop arrangements in 28 families across rock, funk, blues, jazz, Latin, soul, disco, reggae, hip-hop, and fusion. It covers 3/4, 4/4, 5/4, 6/8, 7/8, 7/4, and 12/8. Every example has a practice-length phrase and a stable timekeeping part. Toms occur as part of recurring rhythmic figures. The 5/4 and 7/8 families each have two accent groupings.

FormKit's `src/music/groove-schema.ts`, `groove-playability.ts`, and genre, Latin, and blues studies informed the written/playback timing split, the kit voices, and the drum-set checks. The score uses a 96-tick quarter-note grid. Optional `grouping` counts are measured in the meter's denominator units, and `offset` moves the audible onset by up to six ticks. Older v1 scores with overlapping hand cymbals may need their hat part revoiced as foot hats for the current playability checks.

Artist tags describe stylistic references for newly arranged drum parts. The corpus draws on these references.

- Steve Gadd's rudimental pocket and Bernard Purdie's half-time shuffle. [Gadd's rudimental lessons](https://hudsonmusic.com/product/rudimental-concepts-part-2/) · [Purdie's shuffle lesson](https://www.drumeo.com/beat/bernard-purdie-drum-lessons/)
- David Garibaldi's displaced backbeat and Zigaboo Modeliste's New Orleans funk. [Garibaldi's Drumeo interview](https://www.drumeo.com/beat/david-garibaldi-drumeo-interview-podcast-22/) · [Modeliste's artist biography](https://www.zigaboo.com/press-kit/)
- Horacio Hernández's clave-centered independence and Airto Moreira's Brazilian vocabulary. [Conversations in Clave](https://hudsonmusic.com/product/conversations-in-clave/) · [Airto Moreira interview](https://daily.redbullmusicacademy.com/2017/06/interview-airto-moreira/)
- Gavin Harrison's accent groupings and Carlton Barrett's one-drop practice. [Harrison's groupings lesson](https://www.drumeo.com/beat/?p=7967) · [Carlton Barrett profile](https://thereggaemuseum.com/musician/carlton-barrett/)

The JSONL uses [MLX LM chat records](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md#data). Each row contains a shared system message, a natural-language request, and the compact score JSON as the assistant message. The score's `metadata` describes the beat, feel, and timing. The JSONL row's separate `groove` object identifies the catalog family, sound, influences, and cycle length used by the matcher. Use `--data training --mask-prompt` for QLoRA with a quantized checkpoint. Keep related families together when splitting data. For comparison, use the same unseen prompts for both engines and score prompt adherence, timing validity, and groove feel separately.

## Checks

```sh
python3 training/build.py
go test -race ./...
go vet ./...
go build ./...
```

The corpus builder can regenerate the JSONL from the authored cycles. Its checks examine hand and foot overlaps, stroke spacing across the repeat, and tom placement. Go also checks the embedded training rows against the score contract and tests both generation paths.
