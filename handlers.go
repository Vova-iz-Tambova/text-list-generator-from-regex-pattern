package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		json.NewEncoder(w).Encode(GenerateResponse{})
		return
	}

	nodes, err := parsePattern(pattern)
	if err != nil {
		json.NewEncoder(w).Encode(GenerateResponse{
			Accepted: []string{"Ошибка: " + err.Error()}, 
			Rejected: []string{},
		})
		return
	}

	var accepted, rejected []string
	seenWords := make(map[string]bool)

	generateRecursiveStream(nodes, 0, "", seenWords, &accepted, &rejected)

	json.NewEncoder(w).Encode(GenerateResponse{Accepted: accepted, Rejected: rejected})
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" || path == "/index.html" {
		http.ServeFile(w, r, "static/index.html")
		return
	}
	if strings.HasPrefix(path, "/static/") {
		http.ServeFile(w, r, "."+path)
		return
	}
	http.NotFound(w, r)
}