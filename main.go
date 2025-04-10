package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type Variable struct {
	isInt bool
	value float64
}

type Function struct {
	params     []string
	expression string
}

var variables = make(map[string]*Variable)
var functions = make(map[string]*Function)

func setVariable(name string, isInt bool, val float64) {
	if v, ok := variables[name]; ok {
		isInt = v.isInt

		if isInt {
			v.value = float64(int64(val))
		} else {
			v.value = val
		}
		return
	}

	if isInt {
		val = float64(int64(val))
	}
	variables[name] = &Variable{isInt: isInt, value: val}
}

func getVariable(name string) (*Variable, bool) {
	v, ok := variables[name]
	return v, ok
}

func setFunction(name string, params []string, expr string) {
	functions[name] = &Function{
		params:     params,
		expression: expr,
	}
}

func getFunction(name string) (*Function, bool) {
	f, ok := functions[name]
	return f, ok
}

type TokenType int

const (
	TokenNumber TokenType = iota
	TokenIdent
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenLParen
	TokenRParen
	TokenComma
	TokenEOF
	TokenError
)

type Token struct {
	typ   TokenType
	value string
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(s string) *Lexer {
	return &Lexer{input: []rune(s), pos: 0}
}

func (l *Lexer) nextRune() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r := l.input[l.pos]
	l.pos++
	return r
}

func (l *Lexer) peekRune() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) NextToken() Token {
	for unicode.IsSpace(l.peekRune()) {
		l.nextRune()
	}

	r := l.peekRune()
	if r == 0 {
		return Token{typ: TokenEOF, value: ""}
	}

	switch r {
	case '+':
		l.nextRune()
		return Token{typ: TokenPlus, value: "+"}
	case '-':
		l.nextRune()
		return Token{typ: TokenMinus, value: "-"}
	case '*':
		l.nextRune()
		return Token{typ: TokenStar, value: "*"}
	case '/':
		l.nextRune()
		return Token{typ: TokenSlash, value: "/"}
	case '(':
		l.nextRune()
		return Token{typ: TokenLParen, value: "("}
	case ')':
		l.nextRune()
		return Token{typ: TokenRParen, value: ")"}
	case ',':
		l.nextRune()
		return Token{typ: TokenComma, value: ","}
	}

	if unicode.IsDigit(r) {
		startPos := l.pos
		dotCount := 0
		for unicode.IsDigit(l.peekRune()) || l.peekRune() == '.' {
			if l.peekRune() == '.' {
				dotCount++
				if dotCount > 1 {
					break
				}
			}
			l.nextRune()
		}
		numStr := string(l.input[startPos:l.pos])
		return Token{typ: TokenNumber, value: numStr}
	}

	if unicode.IsLetter(r) || r == '_' {
		startPos := l.pos
		for unicode.IsLetter(l.peekRune()) || unicode.IsDigit(l.peekRune()) || l.peekRune() == '_' {
			l.nextRune()
		}
		ident := string(l.input[startPos:l.pos])
		return Token{typ: TokenIdent, value: ident}
	}

	return Token{typ: TokenError, value: string(r)}
}

func processLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	// Убираем trailing ';'
	if strings.HasSuffix(line, ";") {
		line = line[:len(line)-1]
	}
	line = strings.TrimSpace(line)

	// 1) Проверим, не print ли это
	//    - "print;" или "print varName;"
	if strings.HasPrefix(line, "print") {
		rest := strings.TrimSpace(line[len("print"):])
		if rest == "" {
			// вывести все переменные
			fmt.Println("== Список всех переменных ==")
			for name, v := range variables {
				if v.isInt {
					fmt.Printf("%s = %d (int)\n", name, int64(v.value))
				} else {
					fmt.Printf("%s = %g (float)\n", name, v.value)
				}
			}
		} else {
			// print varName
			rest = strings.TrimSpace(rest)
			if rest[0] == '=' {
				// теоретически такого не должно быть, но на всякий случай
				rest = rest[1:]
				rest = strings.TrimSpace(rest)
			}
			varName := rest
			if v, ok := getVariable(varName); ok {
				if v.isInt {
					fmt.Printf("%s = %d (int)\n", varName, int64(v.value))
				} else {
					fmt.Printf("%s = %g (float)\n", varName, v.value)
				}
			} else {
				fmt.Printf("ОШИБКА: переменная \"%s\" не объявлена\n", varName)
			}
		}
		return
	}

	// 2) Проверим, не функция ли это
	//    Признак – наличие двоеточия ':' после списка параметров
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		left := strings.TrimSpace(parts[0])  // foo(x, y)
		right := strings.TrimSpace(parts[1]) // (x*y+2)...

		// Разберём left, чтобы извлечь имя функции и параметры
		// Формат:  functionName(param1, param2, ...)
		idxOpenParen := strings.Index(left, "(")
		idxCloseParen := strings.Index(left, ")")
		if idxOpenParen == -1 || idxCloseParen == -1 || idxCloseParen < idxOpenParen {
			fmt.Println("ОШИБКА: неверный формат определения функции:", line)
			return
		}
		funcName := strings.TrimSpace(left[:idxOpenParen])
		paramsStr := left[idxOpenParen+1 : idxCloseParen]
		paramsStr = strings.TrimSpace(paramsStr)
		var paramNames []string
		if paramsStr != "" {
			arr := strings.Split(paramsStr, ",")
			for _, p := range arr {
				paramNames = append(paramNames, strings.TrimSpace(p))
			}
		}

		// Сохраняем функцию
		setFunction(funcName, paramNames, right)
		return
	}

	// 3) Проверим, не инициализация ли это переменной
	if strings.Contains(line, ")=") {
		// Пример: myvar(i)=15
		parts := strings.SplitN(line, ")=", 2)
		left := parts[0] // myvar(i
		right := strings.TrimSpace(parts[1])
		idxOpenParen := strings.Index(left, "(")
		if idxOpenParen == -1 {
			fmt.Println("ОШИБКА: неверный формат при инициализации переменной:", line)
			return
		}
		varName := strings.TrimSpace(left[:idxOpenParen])
		typeChar := strings.TrimSpace(left[idxOpenParen+1:]) // i или f

		// Вычислим выражение
		val, ok := evaluateExpression(right)
		if !ok {
			return
		}
		if typeChar == "i" {
			setVariable(varName, true, val)
		} else if typeChar == "f" {
			setVariable(varName, false, val)
		} else {
			fmt.Println("ОШИБКА: неизвестный тип переменной:", typeChar)
		}
		return
	}

	// 4) Иначе, это либо обычное присваивание вида varName=expr,
	//    либо что-то некорректное.
	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		varName := strings.TrimSpace(parts[0])
		expr := strings.TrimSpace(parts[1])

		val, ok := evaluateExpression(expr)
		if !ok {
			return
		}
		// Если переменная уже объявлена, берём её тип, иначе выводим из значения.
		v, found := getVariable(varName)
		if found {
			// сохраняем значение с учётом её типа
			if v.isInt {
				v.value = float64(int64(val))
			} else {
				v.value = val
			}
		} else {
			// Выводим тип из результата (если число целое, значит int, иначе float)
			isInt := float64(int64(val)) == val
			setVariable(varName, isInt, val)
		}
		return
	}

	// Если ничего из вышеперечисленного не подошло, считаем строку некорректной
	fmt.Println("ОШИБКА: не могу разобрать инструкцию:", line)

}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Передай файл")
		return
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Ошибка открытия файла: ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Ошибка чтения файла", err)
		return
	}

}
