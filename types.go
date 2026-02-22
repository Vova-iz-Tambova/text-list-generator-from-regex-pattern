package main

type GenerateRequest struct {
	Pattern          string `json:"pattern"`
	ExcludeUppercase bool   `json:"exclude_uppercase"`
	ExcludeLatin     bool   `json:"exclude_latin"`
	ExcludeDigits    bool   `json:"exclude_digits"`
	ExcludeSpecial   bool   `json:"exclude_special"`
	DisableUnicode   bool   `json:"disable_unicode"`
	GenerateNegative bool   `json:"generate_negative"`
}

type SSEMessage struct {
	Type          string `json:"type"`
	Word          string `json:"word,omitempty"`
	Progress      int    `json:"progress,omitempty"`
	Total         int    `json:"total,omitempty"`
	Count         int    `json:"count,omitempty"`
	RejectedCount int    `json:"rejected_count,omitempty"`
	Error         string `json:"error,omitempty"`
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
	Position         *Position
	Quantified       *QuantifiedPosition
	LookaheadAlts    []LookaheadAlternative
	IsLookahead      bool
}

type CancelRequest struct {
	SessionID string `json:"session_id"`
}