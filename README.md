# Auth Service

Необходимо реализовать ядро сервиса аутентификации на Go. Сервис должен регистрировать пользователей, выполнять вход и возвращать данные текущего пользователя.


## Стек

- Go 1.26.5, `net/http` + [`chi`](https://github.com/go-chi/chi) роутер
- PostgreSQL, драйвер [`pgx/v5`](https://github.com/jackc/pgx) (пул `pgxpool`)
- Пароли — `bcrypt`
- Токены — stateless JWT (`golang-jwt/jwt/v5`, HS256)
- Тесты — `testing` + `testify`

## Команды:
### Запуск
>make ud
### Остановка
>make down
### Тесты
>make test


## API

### `POST /register`

```json
{"email": "user@example.com", "password": "strong-password"}
```
### ответ
| Код | Когда |
|---|---|
| 201 | пользователь создан |
| 400 | невалидный email / пароль короче 8 символов / битый JSON |
| 409 | email уже занят |

Ответ 201: `{"id": "...", "email": "..."}` — пароль/хеш никогда не возвращаются.

### `POST /login`

```json
{"email": "user@example.com", "password": "strong-password"}
```
### ответ
| Код | Когда |
|---|---|
| 200 | `{"token": "..."}` |
| 401 | неверный email или пароль (ответ одинаковый в обоих случаях) |
| 400 | битый JSON |

### `GET /me`
Заголовок `Authorization: Bearer <token>`.
### ответ
| Код | Когда |
|---|---|
| 200 | `{"id": 1, "email": "..."}` (id — UUID) |
| 401 | заголовок отсутствует / токен битый / токен просрочен |
