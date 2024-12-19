# Calculator API

Этот проект представляет собой простой REST API для вычисления математических выражений. API поддерживает базовые арифметические операции, такие как сложение, вычитание, умножение и деление, а также поддерживает скобки и отрицательные числа.

## Особенности

- Поддержка базовых операций: `+`, `-`, `*`, `/`. 
- Обработка скобок для учета приоритета операций.
- Поддержка чисел с плавающей запятой.
- Результаты возвращаются в формате JSON.
- Обработка недопустимых выражений с возвратом соответствующих кодов ошибок.

## Структура проекта

- `main.go` — основной файл программы, который содержит логику API.
- `main_test.go` — тестовый файл для функции Calc(main.go).
- `go.mod` — файл с зависимостями и настройками Go модуля.
- `README.md` -  файл с описанием проекта, его функционала, инструкций по установке и запуску, примерами использования API и т. п.
- `LICENSE.txt` -  файл, в котором указана лицензия, под которой распространяется проект.

## Как запустить
Для запуска проекта необходимо:
1. Установить [Go](https://go.dev/dl/).
2. Склонировать проект с GitHub:
```bash
git clone https://github.com/gtrmalay/HTTP-Calculator.git
```
3. Перейти в директорию проекта.
4. Запустить команду в терминале:
```shell
go run main.go
```
> **Примечание:** Убедитесь, что ваша версия Go +- 1.23.0
> Доступ к API по URL: `http://localhost:8080/api/v1/calculate`
  
## API Эндпоинты

### POST `/api/v1/calculate`

Этот эндпоинт принимает POST-запрос с телом, содержащим математическое выражение, и возвращает результат вычисления этого выражения.

#### Формат запроса

Запрос должен содержать JSON-объект с полем `expression`, которое представляет собой строку с математическим выражением.

## Примеры

200(OK)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "2+2*2"
}'
```
Результат 
```shell
{
    "result": "6"
}
```

405(Method not allowed)
```shell
curl --location --request GET 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    
}'
```
Результат 
```shell
{
    "error": "Method not allowed"
}
```

422(Unprocessable Entity)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "abc"
}'
```
Результат 
```shell
{
    "error": "Expression is not valid"
}
```

500(Internal Server Error)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "2+2"
'
```
Результат 
```shell
{
    "error": "Internal server error"
}
```
