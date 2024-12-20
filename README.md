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

Этот API калькулятор работает на порту 8080 по умолчанию.

> **Примечание:** Убедитесь, что ваша версия Go +- 1.23.0 и что порт 8080 свободен на вашей машине. В противном случае API не сможет работать корректно. 

Доступ к API по URL: `http://localhost:8080/api/v1/calculate`
  
## API Эндпоинты

### POST `/api/v1/calculate`

Этот эндпоинт принимает POST-запрос с телом, содержащим математическое выражение, и возвращает результат вычисления этого выражения.

## Как использовать 

1. Убедитесь, что приложение работает на порту 8080.
2. Для взаимодействия с API используйте команду curl или любой другой HTTP-клиент для отправки POST-запросов с JSON-данными (Postman, SOAP)
3. В запросе передавайте математическое выражение в поле expression.
API ответит с результатом вычислений или ошибкой в зависимости от правильности запроса.

## Примеры

В этом разделе представлены примеры использования API калькулятора с различными ответами от сервера. Для отправки запросов к API используется утилита `curl`, которая позволяет отправлять HTTP-запросы с командной строки.

200(OK)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "2+2*2"
}'
```
В данном примере мы отправляем POST-запрос на вычисление математического выражения 2 + 2 * 2.
Мы указываем заголовок Content-Type: application/json, что означает, что данные передаются в формате JSON.

Результат 
```shell
{
    "result": "6"
}
```
Сервер отвечает результатом вычисления: "6", что является правильным ответом для выражения 2 + 2 * 2.

405(Method not allowed)
```shell
curl --location --request GET 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    
}'
```
В данном примере мы пытаемся использовать GET-запрос вместо POST, что приводит к ошибке, так как метод GET не поддерживается для этого API.
Несмотря на указание заголовка Content-Type: application/json, для правильной работы API требуется метод POST.

Результат 
```shell
{
    "error": "Method not allowed"
}
```
Сервер сообщает, что метод GET не поддерживается для данного маршрута API.

422(Unprocessable Entity)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "abc"
}'
```
Здесь мы отправляем запрос с выражением "abc", которое не является валидным математическим выражением.
Сервер ожидает корректное математическое выражение в поле expression, и в данном случае "abc" не может быть обработано.

Результат 
```shell
{
    "error": "Expression is not valid"
}
```

Сервер сообщает об ошибке, что переданное выражение не является допустимым для вычисления.

500(Internal Server Error)
```shell
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
    "expression": "2+2"
'
```
В этом запросе имеется ошибка в формате JSON (отсутствует закрывающая скобка }), что приводит к внутренней ошибке сервера.
Это пример того, как некорректный запрос (с ошибкой синтаксиса JSON) может вызвать ошибку на сервере.

Результат 
```shell
{
    "error": "Internal server error"
}
```
Сервер возвращает ошибку 500, указывая на внутреннюю проблему в обработке запроса.

