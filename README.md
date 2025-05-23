# HTTP-Calculator. Распределенный вычислитель арифметических выражений 

Рәхим итегез!

Если возникли вопросы по моему проекту или трудности, добро пожаловать в мой телеграм:
```
https://t.me/akkoyash
```
![itachi](https://github.com/user-attachments/assets/570e2a24-5179-4e43-b202-4278853068c7)


## О проекте
Мой проект представляет собой систему для параллельного вычисления арифметических выражений в распределённой среде. Агент отвечает за выполнение вычислений: он запускает заданное количество воркеров, которые обрабатывают отдельные бинарные выражения, входящие в состав общего выражения, работая одновременно.

## Структура проекта
```
cmd/
  ├── agent/
  │   └── main.go
  ├── calculator/
  │   └── main.go
internal/
  ├── agent/
  │   └── agent_test.go
  │   └── agent.go
  ├── handlers/
  │   └── handlers_test.go
  │   └── handlers.go
  ├── models/
  │   ├── models.go
  ├── storage/
  │   ├── storage.go
styles/
  ├── style.css
templates/
  ├── expressions.html
  ├── index.html
tests/
  ├── integration/
  │   ├── integration_test.go
  ├── unit/
  │   ├── auth_test.go
  │   ├── storage_test.go
go.mod
go.sum
LICENSE.txt
README.md
```

## Установка и запуск проекта

### Установка

1. Клонируйте репозиторий
```sh
git clone https://github.com/gtrmalay/HTTP-Calculator.git
```
2. Перейдите в директорию проекта
```sh
cd .\HTTP-Calculator\
```
3. Установите зависимости
```sh
go mod tidy
```
### Запуск

1. Скачайте и установите PostgreSQL
```sh
https://www.postgresql.org/download/
```

2. Создайте БД в PostgreSQL

3. Настройте в коде строку подключения БД

В пакете cmd/calculator/main.go необходимо настроить строку подключения (написать имя пользователя(роли), название ранее созданной бд и пароль пользователя):
```
connStr = "user=postgres dbname=calculator_db password=your_db_pass sslmode=disable"
```

> **Примечание:**
> В строке подключения необходимо указать правильный пароль пользователя PostgreSQL.

<details>
<summary>Подробнее о строке подключения</summary>

- user=postgres — имя пользователя (роли) базы данных, под которым приложение будет подключаться. Обычно это postgres — стандартный администратор базы.

- dbname=calculator_db — название базы данных, к которой будет подключаться приложение. Перед запуском проекта нужно создать эту базу с таким именем в PostgreSQL.

- password=your_db_pass — пароль пользователя PostgreSQL. Этот пароль нужно задать при установке PostgreSQL или при создании пользователя. Важно, чтобы он совпадал с тем, что вы указываете здесь.

- sslmode=disable — отключение SSL-соединения. Для локальной разработки обычно SSL не нужен, поэтому он отключается. Если база находится на удалённом сервере, настройка может быть другой.
- 
</details>


<details>
<summary>SQL-Скрипт на случай, если что-то не так с миграциями(хотя все должно быть отлично) </summary>
  
```
-- USERS SEQUENCE
CREATE SEQUENCE IF NOT EXISTS public.users_id_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

-- USERS TABLE
CREATE TABLE IF NOT EXISTS public.users (
    id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    login varchar(255) NOT NULL,
    password_hash varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_login_key UNIQUE (login)
);

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

-- EXPRESSIONS SEQUENCE
CREATE SEQUENCE IF NOT EXISTS public.expressions_id_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

-- EXPRESSIONS TABLE
CREATE TABLE IF NOT EXISTS public.expressions (
    id integer NOT NULL DEFAULT nextval('expressions_id_seq'::regclass),
    user_id integer,
    expression text NOT NULL,
    result double precision,
    status varchar(50) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT expressions_pkey PRIMARY KEY (id),
    CONSTRAINT expressions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users (id)
);

ALTER SEQUENCE public.expressions_id_seq OWNED BY public.expressions.id;

-- TASKS TABLE
CREATE TABLE IF NOT EXISTS public.tasks (
    id varchar(36) NOT NULL,
    expression_id integer NOT NULL,
    arg1 varchar(255) NOT NULL,
    arg2 varchar(255) NOT NULL,
    operation varchar(10) NOT NULL,
    operation_time integer NOT NULL,
    status varchar(50) NOT NULL,
    result double precision,
    depends_on text[],
    CONSTRAINT tasks_pkey PRIMARY KEY (id),
    CONSTRAINT tasks_expression_id_fkey FOREIGN KEY (expression_id) REFERENCES public.expressions (id)
);

-- TASK_QUEUE TABLE
CREATE TABLE IF NOT EXISTS public.task_queue (
    task_id varchar(36) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT task_queue_pkey PRIMARY KEY (task_id),
    CONSTRAINT task_queue_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks (id)
);
```
</details>

2. Запустите оркестратор:
```sh
go run ./cmd/calculator/main.go
```
3. Запустите агента:
```sh
go run ./cmd/agent/main.go
```

В программе используются переменные среды, чтобы указать время выполнения операций, необходимо перед командой запуска программы в консоли указать значение переменной среды

Пример:
```sh
$env:TIME_ADDITION_MS="1000"; $env:TIME_SUBTRACTION_MS="1000"; $env:TIME_MULTIPLICATION_MS="2000"; $env:TIME_DIVISION_MS="2000"; go run ./cmd/calculator/main.go 
```
Чтобы указать количество горутин для вычисления задач тоже указывается значение переменной среды

Пример:
```sh
COMPUTING_POWER=3 go run ./cmd/agent/main.go
```

**Примечение:** если не указать определенные значения, то программа установит default значения по умолчанию

## Запуск тестов:

В пакете tests/integration/integration_test.go необходимо настроить строку подключения (написать название бд и пароль):

> **Примечание:** Лучше использовать отдельную базу данных для тестов, так как она будет очищать все данные перед тестами для корректной работы.

```
connStr = "user=postgres dbname=test_calculator_db password=your_db_pass sslmode=disable"
```

Команды для запуска тестов:
```
go test .\tests\unit\
go test .\tests\integration\
```

## Документация

### Регистрация и авторизация

Система поддерживает регистрацию новых пользователей и их аутентификацию для обеспечения доступа к вычислениям. Все защищённые эндпоинты (например, /api/v1/calculate, /api/v1/expressions) требуют авторизации через токен JWT.

****

- **Регистрация пользователя**

**Пример запроса через curl:**
```sh
curl --location 'localhost:8080/api/v1/register' \
--header 'Content-Type: application/json' \
--data '{
    "login": "user1",
    "password": "password123"
}'
```
**Ответ:**
```sh
{
    "status": "OK"
}
```
Код ответа:

200 - регистрация успешна
409 - пользователь с таким логином уже существует
400 - неверный формат запроса

### Пример запроса через Postman

![register](https://github.com/user-attachments/assets/8c1bba53-390b-44d3-990b-713abac799d9)


****

- **Авторизация пользователя**

**Пример запроса через curl:**

```sh
curl --location 'localhost:8080/api/v1/login' \
--header 'Content-Type: application/json' \
--data '{
    "login": "user1",
    "password": "password123"
}'
```
**Ответ:**
```sh
{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```
Код ответа:

200 - авторизация успешна
401 - неверные логин или пароль
400 - неверный формат запроса

### Пример запроса через Postman:

![login](https://github.com/user-attachments/assets/9279ec39-2c6b-4737-88dc-e823388c9a88)


### Авторизация через заголовок
После успешной авторизации в системе необходимо передавать токен авторизации в заголовке каждого защищенного запроса.

### Пример использования токена:

При отправке запроса на API необходимо добавить в заголовок `Authorization` значение токена в формате:

```http
Authorization: Bearer your_token_here
```

### Включение токена в заголовок в Postman

![header](https://github.com/user-attachments/assets/dd63950e-19be-4d2e-a762-443f1ed3f46d)


****

## Компоненты системы

### Оркестратор
Когда пользователь отправляет выражение на сервер, программа строит его представление в обратной польской нотации и возвращает ID выражения (если в выражении есть ошибка, вместо ID возвращается сообщение об ошибке). Выражение добавляется в список тасков со статусом "in pending", и начинается его вычисление.

Процесс расчёта устроен так: программа строит хеш-таблицу(map) на основе обратной польской нотации для дальнейшей работы. Затем она проходит по выражению, находит операнды и операторы, которые могут быть вычислены, и отправляет их в список задач (tasks) и POST-запросом отправляет json-задачу на сервер на энд-поинт /internal/task. Когда агент обращается к серверу, он получает задачу из списка посредством GET-запроса с сервера по энд-поинту /internal/task). После того как агент выполнит вычисления, он формирует результат в виде JSON-объекта и отправляет его на сервер с помощью POST-запроса на энд-поинт /internal/task. Сервер, в свою очередь, получает этот результат и обновляет состояние задачи в хеш-таблице, отмечая её как завершённую.
Если задача имеет зависимость от других задач, сервер должен отслеживать, какие задачи завершены, и только когда все зависимости будут выполнены, задача будет считаться готовой для выполнения. Для этого можно используются дополнительные поля в JSON-объекте задачи, которые будут содержать информацию о зависимостях и их текущем статусе.

### Агент
При старте агент запускает несколько горутин, количество которых определяется переменной среды (COMPUTING_POWER) при запуске (если не была указана, default значение "1", т.е. одна горутина), каждая из которых с небольшим интервалом отправляет GET-запрос на сервер для получения задачи. Если задача успешно получена, горутина выполняет вычисления, формирует ответ с ошибкой и результатом, и отправляет его обратно на сервер через POST-запрос. Если выражение вычислено корректно, поле ошибки остаётся пустым, а результат содержит значение. В противном случае, в поле ошибки будет указана ошибка, а результат будет пустым.

****


### Оркестратор
Оркестратор отвечает за прием арифметических выражений, их разбиение на отдельные операции и распределение этих задач между горутинами агента для выполнения.


#### API Оркестратора
- **Добавление вычисления арифметического выражения**
```sh
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2+2*2"
}'
```
**Ответ:**
```json
Формат:
{
    "id": <уникальный идентификатор выражения>
}

Пример:
{
    "id": "7"
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

### Пример добавления выражения через Postman

![calc](https://github.com/user-attachments/assets/b20a59cd-b81d-4c68-b3b5-8915d04aa670)

****

- **Получение списка выражений**
```sh
curl --location 'localhost:8080/api/v1/expressions'
```
**Ответ:**
```json
Формат:
{
    "expressions": [
        {
            "id": <идентификатор выражения>,
            "user_id": <идентификатор пользователя>,
            "expression": <выражение>,
            "result": <результат вычисления выражения>,
            "status": <статус вычисления выражения>,
            "created_at": <дата и время создания выражения>
        }
    ]
}

Пример:
{
    "expressions": [
        {
            "id": "7",
            "user_id": "0",
            "expression": "2+2*2",
            "result": "6",
            "status": "completed",
            "created_at": "2025-05-11T15:21:01.012817Z"
        },
        {
             "id": "8",
            "user_id": "1",
            "expression": "2+2",
            "result": "4",
            "status": "completed",
            "created_at": "2025-05-11T15:21:03.012817Z"
        }
    ]
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

### Пример получения списка выражений через Postman

![expresses](https://github.com/user-attachments/assets/ab9b0925-1b34-47b2-a79a-1a64e0c0cd07)


- **Получение выражения по его идентификатору**
```sh
Формат:
curl --location 'localhost:8080/api/v1/expressions/:id'

Пример:
curl --location 'localhost:8080/api/v1/expressions/accfe4a0-6a7b-40b3-9881-9c59020248aa'
```
**Ответ:**
```json
Формат:
{
    "expression": {
        "id": <идентификатор выражения>,
        "status": <статус вычисления выражения>,
        "result": <результат выражения>
    }
}

Пример:
{
    "expression": {
        "id": "accfe4a0-6a7b-40b3-9881-9c59020248aa",
        "status": "completed",
        "result": 87
    }
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

- **Получение задачи для выполнения**
```sh
curl --location 'localhost:8080/internal/task'
```
**Ответ:**
```json
Формат
{
    "task": {
        "id": <идентификатор задачи>,
        "arg1": <имя первого аргумента>,
        "arg2": <имя второго аргумента>,
        "operation": <операция>
    }
}
```

- **Прием результата обработки данных**
```sh
curl --location 'localhost:8080/internal/task' \
--header 'Content-Type: application/json' \
--data '{
  "id": 7dc0b599-d044-4336-81a7-85c7c4117b9d,
  "result": 85
}'
```

---

### Агент
Агент получает задачи от оркестратора, выполняет их и отправляет обратно результаты.
Агент запускает несколько вычислительных горутин, количество которых регулируется переменной среды `COMPUTING_POWER`.

## Примеры запросов и ответов для сервера

### 1. Добавление вычисления арифметического выражения

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2 + 2 * 2"
}'
```

**Ответ**:
```json
{
    "id": "7"
}
```

**Код ответа**:
- 201 - выражение принято для вычисления

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2 / 0"
}'
```

**Ответ**:
```json
{
    "error": "Division by zero"
}
```

**Код ответа**:
- 422 - невалидные данные

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "exp":
}'
```

**Ответ**:
```json
{
    "error": "Internal server error"
}
```

**Код ответа**:
- 500 - некорректный запрос

---

### 2. Получение списка выражений

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions'
```

**Ответ**:
```json
{
    "expressions": [
        {
            "id": 1741187753311405100,
            "status": "in process",
            "result": ""
        },
        {
            "id": 174118775331145300,
            "status": "done",
            "result": 6.0000
        }
    ]
}
```

**Код ответ**:
- 200 - успешно получен список выражений

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions'
```

**Ответ**:
```json
{
    "error":"empty base"
}
```

**Код ответ**:
- 500 - база данных пустая

---

### 3. Получение выражения по его идентификатору

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions/:1741187753311405100'
```

**Ответ**:
```json
{
    "expression": {
        "id": 1741187753311405100,
        "status": "in pending",
        "result": ""
    }
}
```

**Код ответа**:
- 200 - успешно получено выражение

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions/:1741187753311643100'
```

**Ответ**:
```json
{
    "error":"Task not found"
}
```

**Код ответа**:
- 404 - нет такого выражения

### Примечание

Также запросы к серверу также могут быть выполнены с помощью Postman. Для этого достаточно:

1. Указать правильный HTTP-метод (GET, POST и т. д.).
2. Настроить соответствующие заголовки и тело запроса (если это необходимо).
3. Отправить запрос на нужный эндпоинт вашего сервера.

Postman позволяет удобно тестировать и отлаживать API, а также проверять ответы на запросы.

### Лицензия MIT

Проект лицензирован под лицензией MIT. Это означает, что вы можете свободно использовать, изменять и распространять программное обеспечение любым способом, при условии выполнения следующих условий:

1. Оригинальное уведомление о авторских правах и это разрешение должны быть включены в любые копии или существенные части программного обеспечения.
2. Программное обеспечение предоставляется "как есть", без какой-либо гарантии, явной или подразумеваемой, включая, но не ограничиваясь, гарантии товарной пригодности, соответствия для определенной цели и ненарушения прав. В любом случае авторы или владельцы авторских прав не несут ответственности за любой иск, ущерб или другие обязательства, возникшие из, в связи с или в результате использования программного обеспечения или других сделок с программным обеспечением.

Эта лицензия позволяет вам свободно использовать код для личных, образовательных или коммерческих целей с минимальными ограничениями.
