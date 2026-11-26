package main

import (
	_ "embed"
	"net/http"
)

//go:embed compare.html
var comparePage []byte

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
	m.generateScore(w, r, request.Prompt)
}
