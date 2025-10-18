package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Операции в map: ключ — оператор, значение — функция
	ops := map[rune]func(int, int) (int, error){
		'+': func(a, b int) (int, error) { return a + b, nil },
		'-': func(a, b int) (int, error) { return a - b, nil },
		'*': func(a, b int) (int, error) { return a * b, nil },
		'/': func(a, b int) (int, error) {
			if b == 0 {
				return 0, fmt.Errorf("деление на ноль")
			}
			return a / b, nil // целочисленное деление
		},
	}

	sc := bufio.NewScanner(os.Stdin)
	fmt.Println("Мини-калькулятор. Вводи выражения вроде: 7+3, 9 -6, -7*8, 7--3, 12/4")
	fmt.Println("Выход: exit или quit")

	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break // EOF
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "exit" || lower == "quit" {
			fmt.Println("Пока!")
			return
		}

		// Удалим пробелы чтобы было удобнее парсить
		expr := strings.ReplaceAll(line, " ", "")

		// Найти позицию оператора (+-*/), игнорируя первый символ (там может быть унарный минус)
		opPos, opRune := findOperator(expr)
		if opPos == -1 {
			fmt.Println("Ошибка: не удалось найти оператор (+, -, *, /).")
			continue
		}

		// Разбить на левый и правый операнд по найденной позиции
		left := expr[:opPos]
		right := expr[opPos+1:]
		if left == "" || right == "" {
			fmt.Println("Ошибка: неверный формат выражения. Пример: -7+3")
			continue
		}

		a, err1 := strconv.Atoi(left)
		b, err2 := strconv.Atoi(right)
		if err1 != nil || err2 != nil {
			fmt.Println("Ошибка: не удалось преобразовать операнды в числа.")
			continue
		}

		opFunc, ok := ops[opRune]
		if !ok {
			fmt.Printf("Ошибка: неподдерживаемый оператор: %c\n", opRune)
			continue
		}

		res, err := opFunc(a, b)
		if err != nil {
			fmt.Println("Ошибка:", err)
			continue
		}

		fmt.Printf("%d %c %d = %d\n", a, opRune, b, res)
	}
}

// findOperator возвращает индекс и сам оператор (+, -, *, /),
// пропуская позицию 0 (чтобы не спутать унарный минус)
func findOperator(s string) (int, rune) {
	for i, r := range s {
		if i == 0 {
			continue // первый символ может быть унарным минусом
		}
		switch r {
		case '+', '-', '*', '/':
			return i, r
		}
	}
	return -1, 0
}
