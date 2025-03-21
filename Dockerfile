# Базовый образ с Go 1.24
FROM golang:1.24-alpine AS builder

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum для загрузки зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование остального кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o focuz-api main.go

# Финальный образ
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/focuz-api .
COPY --from=builder /app/migrations ./migrations

# Установка зависимостей для миграций и работы с PostgreSQL
RUN apk --no-cache add postgresql-client

# Запуск приложения
EXPOSE 8080
CMD ["./focuz-api"]