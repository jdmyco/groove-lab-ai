package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed index.html
var page []byte

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
	port := env("PORT", "8080")
	model := modelClient{
		url:    env("LLM_URL", "http://127.0.0.1:8765/v1/chat/completions"),
		name:   env("LLM_MODEL", "default_model"),
		client: &http.Client{Timeout: 90 * time.Second},
	}
	server := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           loopbackOnly(newHandler(model)),
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

func newHandler(model modelClient) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("GET /training/train.jsonl", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		_, _ = w.Write(corpus)
	})
	mux.HandleFunc("POST /api/generate", model.handleGenerate)
	return mux
}

func (m modelClient) handleGenerate(w http.ResponseWriter, r *http.Request) {
	kind, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if kind != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Send the prompt as application/json.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<10))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "The request body is too large.")
		} else {
			writeError(w, http.StatusBadRequest, "The request body could not be read.")
		}
		return
	}
	var request struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(body, &request, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "Send an object with a prompt string.")
		return
	}
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" || utf8.RuneCountInString(prompt) > 1000 {
		writeError(w, http.StatusBadRequest, "Enter a prompt of 1–1,000 characters.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	userMessage := prompt
	for attempt := 0; attempt < maxAttempts; attempt++ {
		content, err := m.complete(ctx, userMessage)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				writeError(w, http.StatusGatewayTimeout, "The local model timed out.")
			} else {
				writeError(w, http.StatusServiceUnavailable, "The local model is unavailable. Check that it is running and LLM_URL and LLM_MODEL match.")
			}
			return
		}
		score, err := parseScore([]byte(content))
		if err == nil {
			writeJSON(w, http.StatusOK, score)
			return
		}
		userMessage = fmt.Sprintf("%s\n\nPrevious attempt failed validation: %s. Return the complete corrected score.", prompt, err)
	}
	writeError(w, http.StatusBadGateway, "The local model returned an invalid score after 3 attempts.")
}

func (m modelClient) complete(ctx context.Context, prompt string) (string, error) {
	body := struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
		Stream      bool      `json:"stream"`
	}{
		Model:       m.name,
		Messages:    []message{{"system", systemPrompt}, {"user", prompt}},
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
