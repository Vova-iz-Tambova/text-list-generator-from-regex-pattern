package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var (
	cancelMu sync.Mutex
	cancelCh = make(map[string]chan struct{})
)

// ✅ ИСПРАВЛЕНО: "data: %s\n\n" вместо " %s\n\n"
func sendSSE(w http.ResponseWriter, flusher http.Flusher, msg SSEMessage) {
	data, _ := json.Marshal(msg)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	pattern := r.URL.Query().Get("pattern")
	excludeUppercase := r.URL.Query().Get("exclude_uppercase") == "true"
	excludeLatin := r.URL.Query().Get("exclude_latin") == "true"
	excludeDigits := r.URL.Query().Get("exclude_digits") == "true"
	excludeSpecial := r.URL.Query().Get("exclude_special") == "true"
	disableUnicode := r.URL.Query().Get("disable_unicode") == "true"
	generateNegative := r.URL.Query().Get("generate_negative") == "true"

	req := GenerateRequest{
		Pattern:          pattern,
		ExcludeUppercase: excludeUppercase,
		ExcludeLatin:     excludeLatin,
		ExcludeDigits:    excludeDigits,
		ExcludeSpecial:   excludeSpecial,
		DisableUnicode:   disableUnicode,
		GenerateNegative: generateNegative,
	}

	if req.Pattern == "" {
		sendSSE(w, flusher, SSEMessage{Type: "error", Error: "Паттерн не указан"})
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("%p", &req)
	}

	nodes, err := parsePattern(req.Pattern, req.DisableUnicode, req.GenerateNegative)
	if err != nil {
		sendSSE(w, flusher, SSEMessage{Type: "error", Error: err.Error()})
		return
	}

	total := calculateTotal(nodes)
	count := 0
	generated := 0
	rejected := 0

	cancelMu.Lock()
	cancelCh[sessionID] = make(chan struct{})
	cancelChan := cancelCh[sessionID]
	cancelMu.Unlock()

	defer func() {
		cancelMu.Lock()
		close(cancelCh[sessionID])
		delete(cancelCh, sessionID)
		cancelMu.Unlock()
	}()

	sendSSE(w, flusher, SSEMessage{Type: "progress", Progress: 0, Total: total})

	generateRecursiveStream(nodes, 0, "", &count, &generated, &rejected, total, req, sessionID, cancelChan, w, flusher)

	sendSSE(w, flusher, SSEMessage{Type: "complete", Count: generated, RejectedCount: rejected, Total: total})
}

func handleCancel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req CancelRequest
	json.NewDecoder(r.Body).Decode(&req)

	cancelMu.Lock()
	if ch, ok := cancelCh[req.SessionID]; ok {
		close(ch)
		delete(cancelCh, req.SessionID)
	}
	cancelMu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasPrefix(path, "/static/") {
		http.ServeFile(w, r, "."+path)
		return
	}

	if path == "/" {
		http.ServeFile(w, r, "static/index.html")
		return
	}

	http.NotFound(w, r)
}