# Multi-stage build для оптимального размера образа
FROM golang:1.23-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем рабочую директорию
WORKDIR /build

# Копируем go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка бинарника с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o monitor ./cmd/monitor

# Финальный образ
FROM scratch

# Копируем сертификаты CA
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копируем информацию о временных зонах
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копируем собранный бинарник
COPY --from=builder /build/monitor /usr/local/bin/monitor

# Метаданные
LABEL maintainer="your-email@example.com" \
      description="System monitoring CLI utility for Zabbix" \
      version="1.0.0"

# Пользователь по умолчанию (для безопасности)
USER 1000:1000

# Точка входа
ENTRYPOINT ["/usr/local/bin/monitor"]

# Команда по умолчанию
CMD ["--help"] 