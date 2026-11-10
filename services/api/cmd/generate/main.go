package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jdmyco/groove-lab-ai/services/api/internal/groove"
)

func main() {
	// Read a JSON request from standard input.
	var req groove.Request
	decoder := json.NewDecoder(os.Stdin)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		fail(err)
	}

	// Reject multiple JSON values in the input.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		fail(fmt.Errorf("expected exactly one JSON object"))
	}

	// Generate a groove.
	score, err := groove.Generate(req)
	if err != nil {
		fail(err)
	}

	// Validate the result before returning it.
	if err := groove.Validate(score); err != nil {
		fail(err)
	}

	// Write the result as formatted JSON.
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(score); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
