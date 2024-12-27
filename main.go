package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type Expression struct {
	ExpressionStr string `json:"expression"`
}

// ф-я - число ли?
func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)

	return err == nil
}

// ф-я - операция ли?
func isOperation(s string) bool {
	return s == "+" || s == "-" || s == "/" || s == "*"
}

// функция для проверки приоритета операции
func Priority(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	}
	return 0
}

func Tokenisation(expression string) []string {
	tokens := []string{}
	currentNumber := ""

	for i, char := range expression {

		if char == ' ' {
			continue
		}

		if isNumber(currentNumber+string(char)) || string(char) == "." {
			currentNumber += string(char)
		} else {
			// проверка на минус перед числом (либо начало выражения, либо до минуса откр. скобка, либо до минуса имеется другой знак)
			if char == '-' && (i == 0 || expression[i-1] == '(') {
				currentNumber += string(char) // добавляем '-' к числу
			} else {
				if currentNumber != "" {
					tokens = append(tokens, currentNumber) // Добавляем число
					currentNumber = ""
				}
				tokens = append(tokens, string(char)) // Добавляем оператор или скобку
			}
		}
	}

	if currentNumber != "" {
		tokens = append(tokens, currentNumber) // Добавляем последнее число
	}

	return tokens
}

// ф-я: токены --> ОПЗ
func RPN(tokens []string) ([]string, error) {
	output := []string{}
	var operators []string

	for i, token := range tokens {
		if isNumber(token) {
			output = append(output, token)
		} else if isOperation(token) {

			if token == "-" && (i == 0 || tokens[i-1] == "(" || isOperation(tokens[i-1])) {
				output = append(output, "0")
			}

			for len(operators) > 0 && Priority(token) <= Priority(operators[len(operators)-1]) {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}

			operators = append(operators, token)

		} else if token == "(" {
			operators = append(operators, token)
		} else if token == ")" {

			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}

			if len(operators) == 0 {
				return nil, errors.New("Что-то не так со скобками!")
			}

			operators = operators[:len(operators)-1]
		}
	}

	for len(operators) > 0 {
		output = append(output, operators[len(operators)-1])
		operators = operators[:len(operators)-1]
	}

	return output, nil
}

// вычисление a ? b
func ApplyOperation(a float64, b float64, op string) (float64, error) {

	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, errors.New("Делить на ноль нельзя!")
		}
		return a / b, nil
	default:
		return 0, errors.New("В выражении есть неопознанные операции!")
	}

}

// вычисление ОПЗ
func ApplyRPN(rpn []string) (float64, error) {
	var stack []float64

	for _, token := range rpn {
		if isNumber(token) {

			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, err
			}

			stack = append(stack, num)

		} else if isOperation(token) {
			if len(stack) < 2 {
				return 0, errors.New("Не хватает операндов для вычисления!")
			}

			b := stack[len(stack)-1]
			stack = stack[:len(stack)-1] // Удаляем верхний элемент

			a := stack[len(stack)-1]
			stack = stack[:len(stack)-1] // удаляем след. элемент

			result, err := ApplyOperation(a, b, token)
			if err != nil {
				return 0, err
			}

			stack = append(stack, result)
		}
	}

	if len(stack) != 1 {

		return 0, errors.New("Выражение некорректно!")
	}

	return stack[0], nil
}

func isValidExpression(expression string) bool {
	// проверка на корректность выражения
	for _, r := range expression {
		if !(r >= '0' && r <= '9' || r == '+' || r == '-' || r == '*' || r == '/' || r == ' ' || r == '(' || r == ')') {
			return false
		}
	}

	return true
}

// cама ф-я Calc
func Calc(expression string) (float64, error) {
	tokens := Tokenisation(expression)
	rpn, err := RPN(tokens)

	if err != nil {

		return 0, err

	}
	result, err := ApplyRPN(rpn)

	if err != nil {

		return 0, err

	}
	return result, nil
}

func ExpressionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed) //ошибка 405
		w.Write([]byte(`{"error": "Method not allowed"}`))

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) //ошибка 500
		w.Write([]byte(`{"error": "Internal server error"}`))

		return
	}
	defer r.Body.Close()

	var expressionObj Expression //объект для выражения
	err = json.Unmarshal(body, &expressionObj)
	if err != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) //ошибка 500
		w.Write([]byte(`{"error": "Internal server error"}`))

		return
	}

	expression := expressionObj.ExpressionStr //выражение строкой

	if !isValidExpression(expression) {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) //ошибка 422
		w.Write([]byte(`{"error": "Expression is not valid"}`))

		return
	}

	resultFloat, calcErr := Calc(expression) //вычисление ответа в float64
	if calcErr != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) //ошибка 422
		w.Write([]byte(`{"error": "Expression is not valid"}`))

		return
	}

	resultString := strconv.FormatFloat(resultFloat, 'f', -1, 64)

	resultBody := map[string]string{
		"result": resultString,
	}

	w.Header().Set("Content-Type", "application/json")
	resultJson, err := json.Marshal(resultBody)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) //ошибка 500

		return
	}

	w.Write(resultJson)
}

func main() {
	http.HandleFunc("/api/v1/calculate", ExpressionHandler)

	http.ListenAndServe(":8080", nil)
}
