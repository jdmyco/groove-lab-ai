# Groove Lab

Groove Lab is a local drum-groove gallery and generator. One Go server serves the HTML page, the 100-example fine-tuning corpus, and an API for a local language model.

## Run the gallery

```sh
mise install
go run .
```

Open [localhost:8080](http://127.0.0.1:8080). Each card pairs the original prompt with a playable sequencer and the score JSON. The gallery uses synthesized drums and works with the Go server alone.

## Connect a local model

The app sends OpenAI-compatible chat requests to `http://127.0.0.1:8765/v1/chat/completions` with `model: "default_model"`, which selects the model configured on MLX LM's server. Override `LLM_URL`, `LLM_MODEL`, or the gallery's `PORT` when starting Go. Start Qwen3 through an existing MLX LM installation with the following command.

```sh
mlx_lm.server --model mlx-community/Qwen3-4B-4bit --port 8765 \
  --chat-template-args '{"enable_thinking":false}'
```

Pass `--adapter-path PATH` to the model server once an adapter is available. Keep `LLM_MODEL=default_model` to use the adapter configured on that server. The Go app binds to loopback and expects a non-streaming chat completion with score JSON in `choices[0].message.content`.

Submit a prompt in the page or send an API request.

```sh
curl -sS http://127.0.0.1:8080/api/generate \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"Make me a funky upbeat groove with quiet snare ghost notes."}'
```

The API returns a [GrooveScoreV1 object](score.schema.json), sorted by tick and instrument. It makes up to three attempts for malformed scores and gives the model validation feedback on each retry. It returns an error if generation fails. The page displays a successful result above the gallery, ready to play.

## Fine-tuning data

[training/train.jsonl](training/train.jsonl) contains 100 distinct drum patterns, 25 per style across rock, funk, blues, and jazz. It includes one- to four-bar phrases, all four supported meters, and straight, swung, and off-grid timing. Each [MLX LM chat record](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md#data) has a shared system message, a natural-language user prompt, and a compact score JSON string in the assistant message. The corpus combines eight seed grooves and 92 scores expanded from authored rhythmic patterns.

Point MLX LM at `--data training --mask-prompt` to use the corpus for supervised fine-tuning. The quantized Qwen checkpoint uses QLoRA. Create a separate evaluation set of different groove families when comparing the adapter to the base model. Keep the [system prompt](training/system-prompt.txt) in sync across the corpus and inference.

## Checks

```sh
go test ./...
go vet ./...
go build ./...
```
