<!--toc:start-->
- [Об архитектуре](#об-архитектуре)
  - [Требования к системе](#требования-к-системе)
  - [Сервисы](#сервисы)
- [Public API specification](#public-api-specification)
  - [Authenticator service](#authenticator-service)
    - [POST /signup](#post-signup)
      - [Request format](#request-format)
      - [Response format](#response-format)
      - [Status codes](#status-codes)
    - [POST /login](#post-login)
      - [Request format](#request-format)
      - [Response format](#response-format)
      - [Status codes](#status-codes)
  - [Shortener service](#shortener-service)
    - [POST /create_short_url (no `JWT` cookie)](#post-createshorturl-no-jwt-cookie)
      - [Request format](#request-format)
      - [Response format](#response-format)
      - [Status codes](#status-codes)
    - [POST /create_short_url (with `JWT` cookie)](#post-createshorturl-with-jwt-cookie)
      - [Request format](#request-format)
      - [Response format](#response-format)
      - [Status codes](#status-codes)
  - [Viewer service](#viewer-service)
    - [GET /history (requires `JWT` cookie)](#get-history-requires-jwt-cookie)
      - [Request format](#request-format)
      - [Response format](#response-format)
      - [Status codes](#status-codes)
<!--toc:end-->

# Об архитектуре

## Требования к системе
* Создание короткой ссылки из данной длинной. 
* Способность справляться с высокой нагрузкой на получение длинной ссылки
по данной короткой.
* Давать больше функциональности зарегистрированным пользователям:
  * Выбрать более длительный срок хранения ссылки: выбор предоставляется из
  30, 90 или 365 дней.
  * Предоставлять историю созданных ссылок

Каждый выделеный сервис заминается только какой-то одной своей конкретной вещью.
Это позволяет заменять их реализацию в любой момент. Выделение в отдельные сервисы 
так же даёт лучшую масштабируемость и устойчивость всей системы.

## Сервисы
* Authenticator - авторизация пользователей на сайте.
* Blackbox - выдача и валидация JWT.
* Shortener - занимается генерацией короткой ссылки. 
* Storage - занимается пакетной записью ссылок в БД.
* Redirector - перенаправляет пользователей с короткой ссылки на длинную.
* Viewer - читает и дает данные из БД пользователю.

Схема межсервисного взаимодействия:

![](./images/services.svg)

Пользователь работает с сайтом в синхронном режиме: например, нажал на кнопку сокращения
ссылки и сразу же её получил. Такое ограничение заставляет нас также использовать
синхронную модель связи между сервисами, участвующими в таком синхронном режиме.
Для этого я выбрал протокол gRPC, так как он выигрывает в скорости передачи и сериализации у
JSON'ов.

Ожидается, что задача сократителя ссылок является read-heavy для БД (на получение длинных ссылок из коротких). 
Чтобы уменьшить нагрузку на БД при чтении длинных ссылок, я добавил на нее кэширвание - редис. 
Использовал сквозное кэширование. Время жизни ключа в редисе - 24 часа.

Для оптимизации работы БД на запись был введен сервис Storage. Он слушает кафку на предмет наличия новых сокращений
и регистрации новых пользователей, после чего вставляет данные сразу пачкой.

# Public API specification

## Authenticator service

address: localhost:8080

### POST /signup

Signs up new user

#### Request format

```
{
    name: string, 
    email: string,
    password: string
}
```

#### Response format

```
{
    message: error description or "success"
}
```

On success also sends two cookies: `auth` and `JWT`.

#### Status codes

* 200 on success
* 400 on invalid form data
* 409 on if user already exists
* 422 on bad JSON data
* 500 on some internal error

### POST /login

Logs in user

#### Request format

```
{
    email: string,
    password: string
}
```

#### Response format

```
{
    message: error description or "success"
}
```

On success also sends two cookies: `auth` and `JWT`.

#### Status codes

* 200 on success
* 400 on invalid form data
* 403 on authentication failure
* 422 on bad JSON data
* 500 on some internal error


## Shortener service

address: localhost:8081

### POST /create_short_url (no `JWT` cookie)

Create a short URL from a given one. This short link will
live 30 days.

#### Request format

```
{
    url: url to shorten,
}
```

#### Response format

```
{
    message: string
}
```

* On success `message` contains short URL.
* On failure `message` contains error description.

#### Status codes

* 200 on success
* 400 on invalid form data
* 422 on bad JSON data
* 500 on some internal error


### POST /create_short_url (with `JWT` cookie)

Create a short URL from a given one. User is prompted to choose expiration date of
short link: 30, 90 or 365 days.

#### Request format

```
{
    url: url to shorten,
    expiration: one of [30, 90, 365]
}
```

#### Response format

```
{
    message: string 
}
```

* On success `message` contains short URL.
* On failure `message` contains error description.

#### Status codes

* 200 on success
* 400 on invalid form data
* 403 on invalid JWT
* 422 on bad JSON data
* 422 on encountering badly formed `JWT` cookie
* 500 on some internal error
* 503 on blackbox service request timeout

## Viewer service

address: localhost:8082

### GET /history (requires `JWT` cookie)

#### Request format

Empty body

#### Response format

* on success returns list of info about shortened urls:
```
[
    {
        short_url: string,
        long_url: string,
        expiration_date: string
    },
    ...
]
```

* on failure returns error description:
```
{
    message: string
}

```

#### Status codes

* 200 on success
* 403 on invalid `JWT`
* 412 on absence of `JWT` cookie
* 422 on encountering badly formed `JWT` cookie
* 500 on some internal error
* 503 on blackbox service request timeout

