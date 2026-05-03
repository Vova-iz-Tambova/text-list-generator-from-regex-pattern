package main

type GenerateRequest struct {
	Pattern string `json:"pattern"`
}

type GenerateResponse struct {
	Accepted []string `json:"accepted"`
	Rejected []string `json:"rejected"`
}

// Для поддержки множественных паттернов
type GenerateMultiRequest struct {
	Patterns []string `json:"patterns"`
}

type PatternResult struct {
	Pattern  string   `json:"pattern"`
	Accepted []string `json:"accepted"`
	Rejected []string `json:"rejected"`
	Error    string   `json:"error,omitempty"`
}

type GenerateMultiResponse struct {
	Results []PatternResult `json:"results"`
}

type Position struct {
	Chars []rune
}

type QuantifiedPosition struct {
	Base Position
	Min  int
	Max  int
}

type LookaheadAlternative struct {
	Chars []rune
}

type PatternNode struct {
	Position      *Position
	Quantified    *QuantifiedPosition
	LookaheadAlts []LookaheadAlternative
	IsLookahead   bool
	IsLookbehind  bool
}
