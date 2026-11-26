package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed index.html
var page []byte

//go:embed assets/formkit
var formkitAssets embed.FS

//go:embed training/train.jsonl
var corpus []byte

//go:embed training/system-prompt.txt
var instruction string

var systemPrompt = strings.TrimSpace(instruction)

const maxAttempts = 3

type modelClient struct {
	url    string
	name   string
	client *http.Client
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	if _, err := loadCatalog(); err != nil {
		log.Fatal(err)
	}
	port := env("PORT", "8080")
	client := &http.Client{Timeout: 90 * time.Second}
	model := modelClient{
		url:    env("LLM_URL", "http://127.0.0.1:8765/v1/chat/completions"),
		name:   env("LLM_MODEL", "default_model"),
		client: client,
	}
	handler := newHandler(model)
	registerComparison(handler,
		modelClient{
			url:    env("LLM_BASE_URL", "http://127.0.0.1:8766/v1/chat/completions"),
			name:   env("LLM_BASE_MODEL", "default_model"),
			client: client,
		},
		modelClient{
			url:    env("LLM_ADAPTER_URL", "http://127.0.0.1:8767/v1/chat/completions"),
			name:   env("LLM_ADAPTER_MODEL", "default_model"),
			client: client,
		},
	)
	server := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           loopbackOnly(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      185 * time.Second,
	}
	log.Printf("Groove Lab at http://%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && !strings.EqualFold(host, "localhost") {
			http.Error(w, "Groove Lab only accepts loopback hostnames.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newHandler(model modelClient) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("GET /training/train.jsonl", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		_, _ = w.Write(corpus)
	})
	mux.Handle("GET /assets/formkit/", http.FileServerFS(formkitAssets))
	kit, err := fs.Sub(formkitAssets, "assets/formkit/audio")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /audio/", http.StripPrefix("/audio/", http.FileServerFS(kit)))
	mux.HandleFunc("POST /api/generate", model.handleGenerate)
	return mux
}

type generationRequest struct {
	Prompt    string `json:"prompt"`
	Engine    string `json:"engine"`
	BaseScore *Score `json:"base_score"`
}

func readGenerationRequest(w http.ResponseWriter, r *http.Request) (generationRequest, bool) {
	var request generationRequest
	kind, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if kind != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Send the prompt as application/json.")
		return request, false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "The request body is too large.")
		} else {
			writeError(w, http.StatusBadRequest, "The request body could not be read.")
		}
		return request, false
	}
	if err := json.Unmarshal(body, &request, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "Send an object with a prompt string.")
		return request, false
	}
	request.Prompt = strings.TrimSpace(request.Prompt)
	if request.Prompt == "" || utf8.RuneCountInString(request.Prompt) > 1000 {
		writeError(w, http.StatusBadRequest, "Enter a prompt of 1–1,000 characters.")
		return request, false
	}
	return request, true
}

func (m modelClient) handleGenerate(w http.ResponseWriter, r *http.Request) {
	request, ok := readGenerationRequest(w, r)
	if !ok {
		return
	}
	switch request.Engine {
	case "procedural":
		serveProcedural(w, request.Prompt, request.BaseScore)
		return
	case "", "llm":
		if request.BaseScore != nil {
			writeError(w, http.StatusBadRequest, "Editing a base score requires the procedural engine.")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "Choose the procedural or llm engine.")
		return
	}

	m.generateScore(w, r, systemPrompt, request.Prompt)
}

func (m modelClient) generateScore(w http.ResponseWriter, r *http.Request, instructions, prompt string) {
	counts, _, err := parseHitCounts(prompt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	userMessage := prompt
	var validationErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		w.Header().Set("X-Groove-Attempts", strconv.Itoa(attempt))
		content, err := m.complete(ctx, instructions, userMessage)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				writeError(w, http.StatusGatewayTimeout, "The local model timed out.")
			} else {
				writeError(w, http.StatusServiceUnavailable, "The local model is unavailable. Check that it is running and the configured model URL and name match.")
			}
			return
		}
		score, err := parseScore([]byte(content))
		if err == nil {
			err = verifyHitCounts(score, counts)
		}
		if err == nil {
			describeScore(&score)
			writeJSON(w, http.StatusOK, score)
			return
		}
		validationErr = err
		userMessage = fmt.Sprintf("%s\n\nPrevious attempt failed validation: %s. Return the complete corrected score.", prompt, err)
	}
	writeError(w, http.StatusBadGateway, fmt.Sprintf("The local model returned an invalid score. Last validation error was %v.", validationErr))
}

func (m modelClient) complete(ctx context.Context, instructions, prompt string) (string, error) {
	body := struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
		Stream      bool      `json:"stream"`
	}{
		Model:       m.name,
		Messages:    []message{{"system", instructions}, {"user", prompt}},
		MaxTokens:   5500,
		Temperature: 0.35,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model returned HTTP %d", response.StatusCode)
	}
	const maxResponse = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return "", err
	}
	var envelope struct {
		Choices []struct {
			Message      message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if len(raw) > maxResponse || json.Unmarshal(raw, &envelope) != nil ||
		len(envelope.Choices) == 0 || envelope.Choices[0].FinishReason == "length" {
		return "", nil
	}
	return envelope.Choices[0].Message.Content, nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, value)
}
