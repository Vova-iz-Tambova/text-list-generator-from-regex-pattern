package main

import (
	"net/http"
	"unicode"
)

func passesFilters(s string, req GenerateRequest) bool {
	for _, r := range s {
		if req.ExcludeUppercase && unicode.IsUpper(r) {
			return false
		}
		if req.ExcludeLatin && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
		if req.ExcludeDigits && unicode.IsDigit(r) {
			return false
		}
		if req.ExcludeSpecial {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

// Генерация комбинаций lookahead
func generateLookaheadCombinations(alt LookaheadAlternative, callback func(string)) {
	generateLookaheadRecursive(alt.Chars, 0, "", callback)
}

func generateLookaheadRecursive(chars []rune, index int, current string, callback func(string)) {
	if index >= len(chars) {
		if current != "" {
			callback(current)
		}
		return
	}

	char := chars[index]

	if char == '[' {
		end := -1
		for j := index + 1; j < len(chars); j++ {
			if chars[j] == ']' {
				end = j
				break
			}
		}
		if end != -1 {
			var rangeChars []rune
			for k := index + 1; k < end; k++ {
				if k+2 < end && chars[k+1] == '-' {
					for c := chars[k]; c <= chars[k+2]; c++ {
						rangeChars = append(rangeChars, c)
					}
					k += 2
				} else if chars[k] != '-' {
					rangeChars = append(rangeChars, chars[k])
				}
			}
			for _, rc := range rangeChars {
				generateLookaheadRecursive(chars, end+1, current+string(rc), callback)
			}
			return
		}
	}

	if unicode.IsLetter(char) || unicode.IsDigit(char) {
		generateLookaheadRecursive(chars, index+1, current+string(char), callback)
	} else {
		generateLookaheadRecursive(chars, index+1, current, callback)
	}
}

// ✅ ИСПРАВЛЕНО: Два этапа генерации (как в debug.go)
func generateRecursiveStream(
	nodes []PatternNode,
	index int,
	current string,
	count *int,
	generated *int,
	rejected *int,
	total int,
	req GenerateRequest,
	sessionID string,
	cancelChan <-chan struct{},
	w http.ResponseWriter,
	flusher http.Flusher,
) {
	select {
	case <-cancelChan:
		return
	default:
	}

	// ✅ ЭТАП 1: Если дошли до конца ИЛИ до lookahead — генерируем базовое слово
	if index >= len(nodes) {
		if current != "" {
			*count++
			
			if !passesFilters(current, req) {
				return
			}
			
			*generated++
			sendSSE(w, flusher, SSEMessage{Type: "word", Word: current})

			if (*generated+*rejected)%100 == 0 {
				progress := (*count * 100) / total
				if progress > 100 {
					progress = 100
				}
				sendSSE(w, flusher, SSEMessage{
					Type:          "progress",
					Progress:      progress,
					Count:         *generated,
					RejectedCount: *rejected,
					Total:         total,
				})
			}
		}
		return
	}

	node := nodes[index]

	// ✅ ЭТАП 2: Если lookahead — генерируем отклонённые для ТЕКУЩЕГО слова
	if node.IsLookahead {
		for _, alt := range node.LookaheadAlts {
			generateLookaheadCombinations(alt, func(suffix string) {
				if suffix != "" {
					rejectedWord := current + suffix
					*count++
					
					if passesFilters(rejectedWord, req) {
						*rejected++
						sendSSE(w, flusher, SSEMessage{Type: "rejected", Word: rejectedWord})
					}

					if (*generated+*rejected)%100 == 0 {
						progress := (*count * 100) / total
						if progress > 100 {
							progress = 100
						}
						sendSSE(w, flusher, SSEMessage{
							Type:          "progress",
							Progress:      progress,
							Count:         *generated,
							RejectedCount: *rejected,
							Total:         total,
						})
					}
				}
			})
		}
		// ✅ Продолжаем рекурсию для базовых слов (не возвращаем!)
		generateRecursiveStream(nodes, index+1, current, count, generated, rejected, total, req, sessionID, cancelChan, w, flusher)
		return
	}

	// ✅ ЭТАП 3: Обычные узлы — генерируем комбинации
	if node.Quantified != nil {
		for repeatCount := node.Quantified.Min; repeatCount <= node.Quantified.Max; repeatCount++ {
			generateQuantifiedCombinations(
				node.Quantified.Base.Chars,
				repeatCount,
				"",
				func(repeated string) {
					generateRecursiveStream(
						nodes, index+1, current+repeated,
						count, generated, rejected, total, req, sessionID,
						cancelChan, w, flusher,
					)
				},
			)
		}
	} else if node.Position != nil {
		for _, char := range node.Position.Chars {
			select {
			case <-cancelChan:
				return
			default:
			}

			if char == 0 {
				generateRecursiveStream(nodes, index+1, current, count, generated, rejected, total, req, sessionID, cancelChan, w, flusher)
			} else {
				generateRecursiveStream(nodes, index+1, current+string(char), count, generated, rejected, total, req, sessionID, cancelChan, w, flusher)
			}
		}
	}
}

func generateQuantifiedCombinations(chars []rune, repeatCount int, current string, callback func(string)) {
	if repeatCount == 0 {
		callback(current)
		return
	}

	for _, char := range chars {
		generateQuantifiedCombinations(chars, repeatCount-1, current+string(char), callback)
	}
}