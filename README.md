# Groovelab

Groove generator - currently deterministic - next step, k8s ai inference to generate grooves based on plaintext input

## Run

Requires Go (managed with `mise`)

```bash
cd ~/workspace/groove-lab-ai
mise install

# Generate a groove from example input
go run ./services/api/cmd/generate < schemas/examples/request.json

# Save the output
go run ./services/api/cmd/generate < schemas/examples/request.json > schemas/examples/rock.json

# Run tests
go test ./...
```
