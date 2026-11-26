package main

import (
	_ "embed"
	"net/http"
)

//go:embed compare.html
var comparePage []byte

// Give both models the same JSON layout, including the nested fields that
// small instruction models tend to place at the top level.
const comparisonScoreExample = `
Use the structure in this example and compose a new score matching the user request.
{"schema_version":"1.0","style":"jazz","meter":"3/4","tempo":{"bpm":110,"beat_unit":"quarter"},"bars":1,"ticks_per_quarter":96,"events":[{"tick":0,"instrument":"ride","velocity":70}],"metadata":{"description":"Ride on the downbeat.","feel":"Spacious","timing":"Straight quarter pulse."}}
Use only these top-level keys plus grouping when needed. Omit grouping for ungrouped meter.
Use meter signatures such as "4/4" without a tick count. Write ticks as integers. In 4/4, beats 1 through 4 are at ticks 0, 96, 192, and 288 of each bar.
Return the raw JSON object only, without Markdown or additional prose.`

func registerComparison(mux *http.ServeMux, base, adapted modelClient) {
	mux.HandleFunc("GET /compare", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(comparePage)
	})
	mux.HandleFunc("POST /api/compare/base", base.handleComparison)
	mux.HandleFunc("POST /api/compare/adapted", adapted.handleComparison)
}

func (m modelClient) handleComparison(w http.ResponseWriter, r *http.Request) {
	request, ok := readGenerationRequest(w, r)
	if !ok {
		return
	}
	if request.BaseScore != nil || request.Engine != "" && request.Engine != "llm" {
		writeError(w, http.StatusBadRequest, "Send a prompt for this comparison model.")
		return
	}
	m.generateScore(w, r, systemPrompt+"\n"+comparisonScoreExample, request.Prompt)
}
