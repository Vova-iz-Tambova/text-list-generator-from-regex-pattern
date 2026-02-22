// debug_test.go - Запуск: go run debug_test.go
// +build ignore

package main

import (
	"fmt"
	"unicode"
)

// === Типы ===
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
	Position        *Position
	Quantified      *QuantifiedPosition
	LookaheadAlts   []LookaheadAlternative
	IsLookahead     bool
}

// === Парсер для [аА][бБ](?![вВ]) ===
func parseSimplePattern() []PatternNode {
	return []PatternNode{
		{Position: &Position{Chars: []rune{'а', 'А'}}},
		{Position: &Position{Chars: []rune{'б', 'Б'}}},
		{
			IsLookahead: true,
			LookaheadAlts: []LookaheadAlternative{
				{Chars: []rune{'[', 'в', 'В', ']'}},
			},
		},
	}
}

// === Генерация lookahead ===
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

	// ✅ Обработка диапазона [...]
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

// === ИСПРАВЛЕННАЯ основная генерация ===
func generateAll(nodes []PatternNode) (accepted, rejected []string) {
	// 1. Находим lookahead узел и отделяем его
	var baseNodes []PatternNode
	var lookaheadNode *PatternNode

	for i, node := range nodes {
		if node.IsLookahead {
			lookaheadNode = &nodes[i]
			break
		}
		baseNodes = append(baseNodes, node)
	}

	// 2. Генерируем базовые слова (без lookahead)
	var baseWords []string
	generateBaseRecursive(baseNodes, 0, "", &baseWords)

	// 3. Для каждого базового слова:
	//    - Добавляем в accepted
	//    - Генерируем отклонённые с lookahead
	for _, baseWord := range baseWords {
		accepted = append(accepted, baseWord)

		if lookaheadNode != nil {
			for _, alt := range lookaheadNode.LookaheadAlts {
				generateLookaheadCombinations(alt, func(suffix string) {
					if suffix != "" {
						rejected = append(rejected, baseWord+suffix)
					}
				})
			}
		}
	}

	return
}

// ✅ Генерация базовых слов (без lookahead)
func generateBaseRecursive(nodes []PatternNode, index int, current string, words *[]string) {
	if index >= len(nodes) {
		if current != "" {
			*words = append(*words, current)
		}
		return
	}

	node := nodes[index]

	if node.Position != nil {
		for _, char := range node.Position.Chars {
			generateBaseRecursive(nodes, index+1, current+string(char), words)
		}
	}
}

// === Тест ===
func main() {
	fmt.Println("🧪 Тест паттерна: [аА][бБ](?![вВ])")
	fmt.Println("══════════════════════════════════")

	nodes := parseSimplePattern()
	accepted, rejected := generateAll(nodes)

	fmt.Printf("\n✅ Принятые (%d):\n", len(accepted))
	for _, w := range accepted {
		fmt.Printf("  • %s\n", w)
	}

	fmt.Printf("\n❌ Отклонённые (%d):\n", len(rejected))
	for _, w := range rejected {
		fmt.Printf("  • %s\n", w)
	}

	expectedAccepted := 4
	expectedRejected := 8

	fmt.Println("\n══════════════════════════════════")
	if len(accepted) == expectedAccepted && len(rejected) == expectedRejected {
		fmt.Println("✅ Тест ПРОЙДЕН!")
	} else {
		fmt.Printf("❌ Тест НЕ ПРОЙДЕН!\n")
		fmt.Printf("   Ожидалось: %d принятых, %d отклонённых\n", expectedAccepted, expectedRejected)
		fmt.Printf("   Получено:  %d принятых, %d отклонённых\n", len(accepted), len(rejected))
	}
}