.PHONY: help build run test clean docker-build docker-run fmt vet lint deps

# Переменные
BINARY_NAME=monitor
DOCKER_IMAGE=zabbix-monitor
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Флаги сборки
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

help: ## Показать это сообщение помощи
	@echo "Доступные команды:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

deps: ## Установить зависимости
	go mod download
	go mod tidy

fmt: ## Форматировать код
	go fmt ./...

vet: ## Статический анализ кода
	go vet ./...

lint: ## Линтинг кода (требует golangci-lint)
	golangci-lint run

test: ## Запустить тесты
	go test -v ./...

test-coverage: ## Запустить тесты с покрытием
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Отчет о покрытии: coverage.html"

build: deps fmt vet ## Собрать бинарник
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/monitor

build-linux: ## Собрать бинарник для Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux ./cmd/monitor

build-all: ## Собрать бинарники для всех платформ
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/monitor
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/monitor
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/monitor

run: build ## Собрать и запустить приложение
	./$(BINARY_NAME) --help

run-example: build ## Запустить с примером конфигурации
	./$(BINARY_NAME) \
		--zabbix-url="http://localhost:8080/api_jsonrpc.php" \
		--zabbix-user="Admin" \
		--zabbix-password="zabbix" \
		--zabbix-host="test-host" \
		--interval=30 \
		--log-level="debug"

docker-build: ## Собрать Docker образ
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: ## Запустить Docker контейнер
	docker run --rm $(DOCKER_IMAGE):latest --help

docker-compose-up: ## Запустить Zabbix инфраструктуру
	docker-compose up -d zabbix-db zabbix-server zabbix-web zabbix-agent

docker-compose-down: ## Остановить все сервисы
	docker-compose down

docker-compose-logs: ## Показать логи сервисов
	docker-compose logs -f

docker-compose-restart-monitor: ## Перезапустить monitor-cli
	docker-compose restart monitor-cli
	docker-compose logs -f monitor-cli

install: build ## Установить бинарник в $GOPATH/bin
	go install $(LDFLAGS) ./cmd/monitor

clean: ## Очистить сгенерированные файлы
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.out coverage.html
	docker-compose down --volumes --remove-orphans
	docker rmi $(DOCKER_IMAGE):latest $(DOCKER_IMAGE):$(VERSION) 2>/dev/null || true

check: deps fmt vet test ## Полная проверка кода

# Цели для разработки
dev-setup: ## Настройка окружения разработки
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go mod download

dev-run: ## Запуск для разработки с hot reload (требует air)
	air

# Цели для CI/CD
ci-test: deps vet test ## Тесты для CI
	go test -race -coverprofile=coverage.out ./...

ci-build: deps ## Сборка для CI
	CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME) ./cmd/monitor

# Показать информацию о версии
version: ## Показать информацию о версии
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)" 