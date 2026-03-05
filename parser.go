package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// decodeUnicodeEscapes преобразует \uXXXX в символы
func decodeUnicodeEscapes(s string) string {
	var result strings.Builder
	i := 0
	runes := []rune(s)

	for i < len(runes) {
		// Проверяем \uXXXX
		if i+5 < len(runes) && runes[i] == '\\' && runes[i+1] == 'u' {
			hex := strings.ToLower(string(runes[i+2 : i+6]))
			code, err := strconv.ParseInt(hex, 16, 32)
			if err == nil {
				result.WriteRune(rune(code))
				i += 6
				continue
			}
		}
		
		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

func extractLookaheadAlternatives(runes []rune, start int) []LookaheadAlternative {
	var alternatives []LookaheadAlternative
	i := start + 3

	currentAlt := []rune{}
	depth := 0

	for i < len(runes) && !(runes[i] == ')' && depth == 0) {
		if runes[i] == '(' {
			depth++
			currentAlt = append(currentAlt, runes[i])
		} else if runes[i] == ')' {
			depth--
			currentAlt = append(currentAlt, runes[i])
		} else if runes[i] == '|' && depth == 0 {
			if len(currentAlt) > 0 {
				alternatives = append(alternatives, LookaheadAlternative{Chars: currentAlt})
			}
			currentAlt = []rune{}
		} else {
			currentAlt = append(currentAlt, runes[i])
		}
		i++
	}

	if len(currentAlt) > 0 {
		alternatives = append(alternatives, LookaheadAlternative{Chars: currentAlt})
	}

	return alternatives
}

func parsePattern(pattern string) ([]PatternNode, error) {
	// Декодируем Unicode
	pattern = decodeUnicodeEscapes(pattern)

	var nodes []PatternNode
	runes := []rune(pattern)
	i := 0

	for i < len(runes) {
		char := runes[i]

		// Негативный lookbehind (?<!...)
		if char == '(' && i+3 < len(runes) && runes[i+1] == '?' && runes[i+2] == '<' && runes[i+3] == '!' {
			alts := extractLookaheadAlternatives(runes, i)
			if len(alts) > 0 {
				nodes = append(nodes, PatternNode{
					LookaheadAlts: alts,
					IsLookahead:   true,
					IsLookbehind:  true,
				})
			}
			end := findGroupEnd(runes, i)
			if end == -1 {
				return nil, fmt.Errorf("незакрытая группа")
			}
			i = end + 1
			continue
		}

		// Негативный lookahead (?!...)
		if char == '(' && i+2 < len(runes) && runes[i+1] == '?' && runes[i+2] == '!' {
			alts := extractLookaheadAlternatives(runes, i)
			if len(alts) > 0 {
				nodes = append(nodes, PatternNode{
					LookaheadAlts: alts,
					IsLookahead:   true,
					IsLookbehind:  false,
				})
			}
			end := findGroupEnd(runes, i)
			if end == -1 {
				return nil, fmt.Errorf("незакрытая группа")
			}
			i = end + 1
			continue
		}

		// Позитивный lookahead (?=...) - игнорируем
		if char == '(' && i+2 < len(runes) && runes[i+1] == '?' && runes[i+2] == '=' {
			end := findGroupEnd(runes, i)
			if end == -1 {
				return nil, fmt.Errorf("незакрытая группа")
			}
			i = end + 1
			continue
		}

		// Диапазон [а-я]
		if char == '[' {
			end := -1
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == ']' {
					end = j
					break
				}
			}
			if end == -1 {
				return nil, fmt.Errorf("незакрытая скобка [")
			}

			content := runes[i+1 : end]
			var chars []rune

			for k := 0; k < len(content); k++ {
				if k+2 < len(content) && content[k+1] == '-' {
					start := content[k]
					endChar := content[k+2]
					for c := start; c <= endChar; c++ {
						chars = append(chars, c)
					}
					k += 2
				} else {
					chars = append(chars, content[k])
				}
			}

			if len(chars) == 0 {
				chars = []rune{' '}
			}

			basePos := Position{Chars: chars}
			i = end + 1

			if i < len(runes) && runes[i] == '{' {
				min, max, newPos, err := parseQuantifier(runes, i)
				if err != nil {
					return nil, err
				}
				i = newPos
				nodes = append(nodes, PatternNode{
					Quantified: &QuantifiedPosition{
						Base: basePos,
						Min:  min,
						Max:  max,
					},
				})
			} else {
				nodes = append(nodes, PatternNode{
					Position: &basePos,
				})
			}
			continue
		}

		// Квантификатор {n,m}
		if char == '{' {
			if len(nodes) == 0 {
				i++
				continue
			}
			min, max, newPos, err := parseQuantifier(runes, i)
			if err != nil {
				return nil, err
			}
			i = newPos
			lastNode := nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]

			if lastNode.Position != nil {
				nodes = append(nodes, PatternNode{
					Quantified: &QuantifiedPosition{
						Base: *lastNode.Position,
						Min:  min,
						Max:  max,
					},
				})
			}
			continue
		}

		// Опциональный ?
		if char == '?' {
			if len(nodes) > 0 {
				lastNode := nodes[len(nodes)-1]
				nodes = nodes[:len(nodes)-1]

				if lastNode.Position != nil {
					nodes = append(nodes, PatternNode{
						Quantified: &QuantifiedPosition{
							Base: *lastNode.Position,
							Min:  0,
							Max:  1,
						},
					})
				}
			}
			i++
			continue
		}

		// Звёздочка *
		if char == '*' {
			if len(nodes) > 0 {
				lastNode := nodes[len(nodes)-1]
				nodes = nodes[:len(nodes)-1]
				nodes = append(nodes, PatternNode{
					Quantified: &QuantifiedPosition{
						Base: *lastNode.Position,
						Min:  0,
						Max:  5,
					},
				})
			}
			i++
			continue
		}

		// Плюс +
		if char == '+' {
			if len(nodes) > 0 {
				lastNode := nodes[len(nodes)-1]
				nodes = nodes[:len(nodes)-1]
				nodes = append(nodes, PatternNode{
					Quantified: &QuantifiedPosition{
						Base: *lastNode.Position,
						Min:  1,
						Max:  5,
					},
				})
			}
			i++
			continue
		}

		// Обычный символ (включая пробел)
		if unicode.IsLetter(char) || unicode.IsDigit(char) || unicode.IsSymbol(char) || char == ' ' {
			basePos := Position{Chars: []rune{char}}
			i++

			if i < len(runes) && runes[i] == '{' {
				min, max, newPos, err := parseQuantifier(runes, i)
				if err != nil {
					return nil, err
				}
				i = newPos
				nodes = append(nodes, PatternNode{
					Quantified: &QuantifiedPosition{
						Base: basePos,
						Min:  min,
						Max:  max,
					},
				})
			} else {
				nodes = append(nodes, PatternNode{
					Position: &basePos,
				})
			}
			continue
		}

		i++
	}

	return nodes, nil
}

func parseQuantifier(runes []rune, start int) (int, int, int, error) {
	end := -1
	for j := start + 1; j < len(runes); j++ {
		if runes[j] == '}' {
			end = j
			break
		}
	}
	if end == -1 {
		return 0, 0, 0, fmt.Errorf("незакрытая скобка {")
	}

	contentRunes := runes[start+1 : end]
	content := string(contentRunes)
	var min, max int
	var err error

	comma := -1
	for idx, c := range content {
		if c == ',' {
			comma = idx
			break
		}
	}

	if comma == -1 {
		min, err = strconv.Atoi(content)
		if err != nil {
			return 0, 0, 0, err
		}
		max = min
	} else {
		min, err = strconv.Atoi(content[:comma])
		if err != nil {
			return 0, 0, 0, err
		}
		maxContent := content[comma+1:]
		if maxContent == "" {
			max = 999
		} else {
			max, err = strconv.Atoi(maxContent)
			if err != nil {
				return 0, 0, 0, err
			}
		}
	}

	if max > 10 {
		max = 10
	}

	return min, max, end + 1, nil
}

func findGroupEnd(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if runes[i] == '(' {
			depth++
		} else if runes[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func calculateTotal(nodes []PatternNode) int {
	total := 1
	hasLookahead := false

	for _, node := range nodes {
		if node.IsLookahead {
			hasLookahead = true
			continue
		}
		if node.Quantified != nil {
			charCount := len(node.Quantified.Base.Chars)
			if charCount == 0 {
				charCount = 1
			}
			for count := node.Quantified.Min; count <= node.Quantified.Max; count++ {
				ways := 1
				for i := 0; i < count; i++ {
					ways *= charCount
				}
				total *= ways
			}
		} else if node.Position != nil {
			charCount := len(node.Position.Chars)
			if charCount == 0 {
				charCount = 1
			}
			total *= charCount
		}
	}

	if hasLookahead {
		total *= 2
	}

	if total == 0 {
		total = 1
	}
	return total
}