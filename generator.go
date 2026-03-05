package main

import (
	"unicode"
)

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

func generateRecursiveStream(
	nodes []PatternNode,
	index int,
	current string,
	seenWords map[string]bool,
	accepted, rejected *[]string,
) {
	if index >= len(nodes) {
		if current != "" && !seenWords[current] {
			seenWords[current] = true
			
			// ✅ Базовое слово → принято
			*accepted = append(*accepted, current)
			
			// ✅ Генерируем отклонённые (с суффиксами lookahead)
			for _, node := range nodes {
				if node.IsLookahead && !node.IsLookbehind {
					for _, alt := range node.LookaheadAlts {
						generateLookaheadCombinations(alt, func(suffix string) {
							if suffix == "" {
								return
							}
							word := current + suffix
							if !seenWords[word] {
								seenWords[word] = true
								*rejected = append(*rejected, word)
							}
						})
					}
				}
				
				// ✅ Генерируем отклонённые (с префиксами lookbehind)
				if node.IsLookahead && node.IsLookbehind {
					for _, alt := range node.LookaheadAlts {
						generateLookaheadCombinations(alt, func(prefix string) {
							if prefix == "" {
								return
							}
							word := prefix + current
							if !seenWords[word] {
								seenWords[word] = true
								*rejected = append(*rejected, word)
							}
						})
					}
				}
			}
		}
		return
	}

	node := nodes[index]

	// Пропускаем lookahead/lookbehind при генерации базовых слов
	if node.IsLookahead {
		generateRecursiveStream(nodes, index+1, current, seenWords, accepted, rejected)
		return
	}

	if node.Quantified != nil {
		for n := node.Quantified.Min; n <= node.Quantified.Max; n++ {
			generateQuantifiedCombinations(node.Quantified.Base.Chars, n, "", func(repeated string) {
				generateRecursiveStream(nodes, index+1, current+repeated, seenWords, accepted, rejected)
			})
		}
	} else if node.Position != nil {
		for _, char := range node.Position.Chars {
			generateRecursiveStream(nodes, index+1, current+string(char), seenWords, accepted, rejected)
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