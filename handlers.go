package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	patterns := r.URL.Query()["patterns"]
	pattern := r.URL.Query().Get("pattern")
	excludeUppercase := r.URL.Query().Get("exclude_uppercase") == "true"
	excludeLatin := r.URL.Query().Get("exclude_latin") == "true"
	excludeDigits := r.URL.Query().Get("exclude_digits") == "true"
	excludeSpecial := r.URL.Query().Get("exclude_special") == "true"

	log.Printf("handleGenerate: patterns=%v, pattern=%s", patterns, pattern)

	// Если передан параметр patterns (массив), обрабатываем множественные паттерны
	if len(patterns) > 0 {
		log.Printf("Processing multiple patterns: %d", len(patterns))
		processMultiplePatterns(w, patterns, excludeUppercase, excludeLatin, excludeDigits, excludeSpecial)
		return
	}

	// Обратная совместимость: одиночный паттерн через параметр pattern
	if pattern == "" {
		log.Printf("Empty pattern, returning empty response")
		json.NewEncoder(w).Encode(GenerateResponse{})
		return
	}

	log.Printf("Processing single pattern: %s", pattern)
	nodes, err := parsePattern(pattern)
	if err != nil {
		log.Printf("Parse error: %v", err)
		json.NewEncoder(w).Encode(GenerateResponse{
			Accepted: []string{"Ошибка: " + err.Error()},
			Rejected: []string{},
		})
		return
	}

	var accepted, rejected []string
	seenWords := make(map[string]bool)

	generateRecursiveStream(nodes, 0, "", seenWords, &accepted, &rejected, excludeUppercase, excludeLatin, excludeDigits, excludeSpecial)

	log.Printf("Single pattern result: accepted=%d, rejected=%d", len(accepted), len(rejected))
	json.NewEncoder(w).Encode(GenerateResponse{Accepted: accepted, Rejected: rejected})
}

// processMultiplePatterns обрабатывает массив паттернов и возвращает результаты для каждого
func processMultiplePatterns(w http.ResponseWriter, patterns []string, excludeUppercase, excludeLatin, excludeDigits, excludeSpecial bool) {
	log.Printf("processMultiplePatterns: %d patterns", len(patterns))
	var results []PatternResult

	for i, p := range patterns {
		if p == "" {
			log.Printf("Pattern %d is empty, skipping", i)
			continue
		}
		log.Printf("Processing pattern %d: %s", i, p)
		result := PatternResult{Pattern: p}
		nodes, err := parsePattern(p)
		if err != nil {
			log.Printf("Pattern %d parse error: %v", i, err)
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		var accepted, rejected []string
		seenWords := make(map[string]bool)
		generateRecursiveStream(nodes, 0, "", seenWords, &accepted, &rejected, excludeUppercase, excludeLatin, excludeDigits, excludeSpecial)
		result.Accepted = accepted
		result.Rejected = rejected
		log.Printf("Pattern %d result: accepted=%d, rejected=%d", i, len(accepted), len(rejected))
		results = append(results, result)
	}

	log.Printf("Total results: %d", len(results))
	json.NewEncoder(w).Encode(GenerateMultiResponse{Results: results})
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
