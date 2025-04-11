package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Тип для хранения информации о переменной
type Variable struct {
	isInt bool    // true, если переменная целая
	value float64 // текущее числовое значение (даже для целых храним в float64, чтобы упрощать вычисления)
}

// Тип для хранения информации о функции
type Function struct {
	params     []string // имена параметров
	expression string   // строка-выражение (парсится при вычислении)
}

// Глобальные карты для хранения переменных и функций
var variables = make(map[string]*Variable)
var functions = make(map[string]*Function)

// === Вспомогательные функции для хранения/поиска переменных и функций ===

func setVariable(name string, isInt bool, val float64) {
	// Если переменная уже существует, используем уже заданный тип (при отсутствии явной инициализации)
	if v, ok := variables[name]; ok {
		// Приведение типа, если нужно
		isInt = v.isInt
		if isInt {
			// Транкция (округление к 0) при записи в целую переменную
			v.value = float64(int64(val))
		} else {
			v.value = val
		}
		return
	}

	// Если переменная новая
	if isInt {
		val = float64(int64(val)) // округляем для целочисленной
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

// === Парсер выражений (упрощённый рекурсивный спуск) ===

// Токенизация
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
	// Пропускаем пробелы
	for unicode.IsSpace(l.peekRune()) {
		l.nextRune()
	}

	r := l.peekRune()
	if r == 0 {
		return Token{typ: TokenEOF, value: ""}
	}

	// Разбираем спецсимволы
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

	// Числа (упрощённо)
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

	// Идентификаторы (имена переменных/функций)
	if unicode.IsLetter(r) || r == '_' {
		startPos := l.pos
		for unicode.IsLetter(l.peekRune()) || unicode.IsDigit(l.peekRune()) || l.peekRune() == '_' {
			l.nextRune()
		}
		ident := string(l.input[startPos:l.pos])
		return Token{typ: TokenIdent, value: ident}
	}

	// Если что-то непонятное, считаем ошибкой
	return Token{typ: TokenError, value: string(r)}
}

// Рекурсивный спуск:
// expr = term { ("+" | "-") term }
// term = factor { ("*" | "/") factor }
// factor = number | ident [ "(" exprlist ")" ] | "(" expr ")"
// exprlist = expr { "," expr }

type Parser struct {
	lexer  *Lexer
	curr   Token
	errMsg string
}

func NewParser(input string) *Parser {
	p := &Parser{lexer: NewLexer(input)}
	p.next()
	return p
}

func (p *Parser) next() {
	p.curr = p.lexer.NextToken()
}

func (p *Parser) error(msg string) {
	p.errMsg = msg
}

func (p *Parser) parseExpression() float64 {
	val := p.parseTerm()
	for p.curr.typ == TokenPlus || p.curr.typ == TokenMinus {
		op := p.curr.typ
		p.next()
		right := p.parseTerm()
		if op == TokenPlus {
			val += right
		} else {
			val -= right
		}
	}
	return val
}

func (p *Parser) parseTerm() float64 {
	val := p.parseFactor()
	for p.curr.typ == TokenStar || p.curr.typ == TokenSlash {
		op := p.curr.typ
		p.next()
		right := p.parseFactor()
		if op == TokenStar {
			val *= right
		} else {
			// деление
			if right == 0 {
				// В реальном интерпретаторе нужно как-то обрабатывать деление на ноль.
				// Здесь просто разделим на 0.0, что даст +Inf/-Inf.
				val = val / 0.0
			} else {
				val /= right
			}
		}
	}
	return val
}

func (p *Parser) parseFactor() float64 {
	switch p.curr.typ {
	case TokenNumber:
		// конвертируем в float64
		f, err := strconv.ParseFloat(p.curr.value, 64)
		if err != nil {
			p.error("Невозможно преобразовать число: " + p.curr.value)
			return 0
		}
		p.next()
		return f
	case TokenIdent:
		// Может быть переменная, может быть вызов функции
		identName := p.curr.value
		p.next()
		if p.curr.typ == TokenLParen {
			// вызов функции
			// Считываем аргументы
			p.next() // пропускаем '('
			args := []float64{}
			if p.curr.typ != TokenRParen {
				for {
					argVal := p.parseExpression()
					args = append(args, argVal)
					if p.curr.typ == TokenComma {
						p.next()
						continue
					}
					break
				}
			}
			if p.curr.typ != TokenRParen {
				p.error("Ожидалась закрывающая скобка в вызове функции")
				return 0
			}
			p.next() // пропускаем ')'

			// Ищем функцию
			fn, ok := getFunction(identName)
			if !ok {
				// Ошибка: функция не найдена
				fmt.Printf("ОШИБКА: использование не объявленной функции \"%s\"\n", identName)
				return 0
			}

			// Проверка числа параметров
			if len(fn.params) != len(args) {
				p.error(fmt.Sprintf("Функция %s ожидала %d аргументов, передано %d",
					identName, len(fn.params), len(args)))
				return 0
			}

			// Вычисляем путём временного создания окружения
			return evaluateFunction(fn, args)
		} else {
			// переменная
			v, ok := getVariable(identName)
			if !ok {
				// Ошибка: переменная не найдена
				fmt.Printf("ОШИБКА: использование не объявленной переменной \"%s\"\n", identName)
				return 0
			}
			return v.value
		}
	case TokenLParen:
		p.next()
		val := p.parseExpression()
		if p.curr.typ != TokenRParen {
			p.error("Ожидалась закрывающая скобка )")
			return val
		}
		p.next()
		return val
	default:
		p.error("Неожиданный токен: " + p.curr.value)
		return 0
	}
}

// evaluateFunction – вычисляет тело функции, подставляя аргументы в параметры.
// Для простоты делаем: во время вычисления выражения функции создаём «временные» переменные с именами параметров
// и после вычисления восстанавливаем старые значения (или отсутствие таковых).
func evaluateFunction(fn *Function, args []float64) float64 {
	// Сохраним текущее состояние переменных, которые совпадают с именами параметров.
	backup := make(map[string]*Variable)
	// Для каждого параметра создаём/перезаписываем переменную
	for i, paramName := range fn.params {
		if orig, found := getVariable(paramName); found {
			backup[paramName] = &Variable{isInt: orig.isInt, value: orig.value}
		}
		// При подстановке аргументов типа не знаем, пусть будет float, если дробь – значит float.
		isInteger := float64(int64(args[i])) == args[i]
		setVariable(paramName, isInteger, args[i])
	}

	// Вычислим выражение
	p := NewParser(fn.expression)
	val := p.parseExpression()
	if p.errMsg != "" {
		fmt.Println("ОШИБКА при вычислении функции:", p.errMsg)
	}

	// Восстановим старые значения переменных
	for _, paramName := range fn.params {
		// Удаляем временную переменную (или восстанавливаем из backup)
		delete(variables, paramName)
		if bkp, ok := backup[paramName]; ok {
			// восстановить
			variables[paramName] = bkp
		}
	}

	return val
}

// evaluateExpression – вспомогательная функция для вычисления произвольной строки-выражения
func evaluateExpression(expr string) (float64, bool) {
	p := NewParser(expr)
	val := p.parseExpression()
	if p.errMsg != "" {
		fmt.Println("ОШИБКА при вычислении выражения:", p.errMsg)
		return 0, false
	}
	return val, true
}

// === Разбор инструкций ===

func processLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	// Убираем trailing ';' (по условию – каждая инструкция заканчивается точкой с запятой)
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

	// 2) Проверим, не функция ли это:  name(arg1, arg2, ...): выражение
	//    Признак – наличие двоеточия ':' после списка параметров
	if strings.Contains(line, ":") {
		// Пример: foo(x, y): (x*y+2)...
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

		// Сохраняем функцию в карту
		setFunction(funcName, paramNames, right)
		return
	}

	// 3) Проверим, не инициализация ли переменной с типом:  varName(i)=...  или varName(f)=...
	//    Ищем шаблон:  что-то(...)=<что-то>
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
		fmt.Println("Использование: go run main.go <путь_к_файлу_инструкций>")
		return
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		processLine(line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	variables = nil
	functions = nil
}
