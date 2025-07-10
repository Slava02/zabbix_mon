# Zabbix Monitoring CLI

Легковесная CLI утилита для мониторинга системных ресурсов Linux и отправки метрик в Zabbix 6.0.

## Особенности

- 🚀 **Производительность**: Минимальное потребление ресурсов
- 📊 **Полный мониторинг**: CPU, память, диск, сеть
- 🔄 **Надежность**: Автоматические повторные попытки при ошибках
- 🐳 **Docker Ready**: Готовые образы и compose файлы
- 🛠 **Конфигурируемость**: Флаги CLI и переменные окружения
- 📝 **Логирование**: Структурированные логи с различными уровнями
- 📡 **Zabbix Sender**: Использует правильный протокол для отправки данных

## Быстрый старт

### 1. Установка через Go

```bash
go install github.com/yourusername/zabbix_mon/cmd/monitor@latest
```

### 2. Запуск с Docker Compose (рекомендуется для тестирования)

```bash
# Клонируем репозиторий
git clone https://github.com/yourusername/zabbix_mon.git
cd zabbix_mon

# Запускаем Zabbix инфраструктуру
docker-compose up -d zabbix-db zabbix-server zabbix-web

# Ждем инициализации (2-3 минуты)
docker-compose logs -f zabbix-server

# Создаем хост в Zabbix (через веб-интерфейс http://localhost:8080)
# Логин: Admin, Пароль: zabbix

# Запускаем утилиту мониторинга
docker-compose up monitor-cli
```

### 3. Ручной запуск

```bash
monitor \
  --zabbix-url="http://localhost:8080/api_jsonrpc.php" \
  --zabbix-user="Admin" \
  --zabbix-password="zabbix" \
  --zabbix-host="test_host" \
  --interval=10 \
  --log-level="info"
```

## Архитектура отправки данных

Утилита использует **двухэтапный подход** для работы с Zabbix:

1. **Zabbix API** (HTTP JSON-RPC) для:
   - Аутентификации 
   - Поиска хостов
   - Создания элементов данных (trapper типа)

2. **Zabbix Sender протокол** (TCP порт 10051) для:
   - Отправки метрик в trapper элементы
   - Высокопроизводительной передачи данных

Это обеспечивает правильную работу с Zabbix 6.0 в соответствии со стандартами.

## Конфигурация

### Флаги командной строки

| Флаг | Описание | По умолчанию |
|------|----------|--------------|
| `--zabbix-url` | URL Zabbix API | `http://localhost:10051/api_jsonrpc.php` |
| `--zabbix-user` | Имя пользователя Zabbix | `Admin` |
| `--zabbix-password` | Пароль пользователя | `zabbix` |
| `--zabbix-host` | Имя хоста в Zabbix | `monitoring-host` |
| `--interval` | Интервал сбора в секундах | `10` |
| `--log-level` | Уровень логирования | `info` |
| `--batch-size` | Размер пакета метрик | `50` |

### Переменные окружения

Все флаги можно задать через переменные окружения:

```bash
export ZABBIX_URL="http://localhost:8080/api_jsonrpc.php"
export ZABBIX_USER="Admin"
export ZABBIX_PASSWORD="zabbix"
export ZABBIX_HOST="production-server"
export INTERVAL="10"
export LOG_LEVEL="info"
export BATCH_SIZE="50"

monitor
```

## Собираемые метрики

### CPU Метрики
- `system.cpu.util[,idle]` - Утилизация CPU (%)
- `system.cpu.load[percpu,avg1]` - Load average за 1 минуту
- `system.cpu.load[percpu,avg5]` - Load average за 5 минут  
- `system.cpu.load[percpu,avg15]` - Load average за 15 минут

### Память
- `vm.memory.size[total]` - Общий объем памяти (байты)
- `vm.memory.size[used]` - Используемая память (байты)
- `vm.memory.size[available]` - Доступная память (байты)
- `vm.memory.util` - Утилизация памяти (%)

### Диск (корневой раздел)
- `vfs.fs.size[/,total]` - Общий размер диска (байты)
- `vfs.fs.size[/,used]` - Используемое место (байты)
- `vfs.fs.size[/,free]` - Свободное место (байты)
- `vfs.fs.pused[/]` - Утилизация диска (%)

### Сеть (все интерфейсы)
- `net.if.in[all]` - Входящий трафик (байты)
- `net.if.out[all]` - Исходящий трафик (байты)
- `net.if.in[all,packets]` - Входящие пакеты
- `net.if.out[all,packets]` - Исходящие пакеты
- `net.if.in[all,errors]` - Ошибки входящих пакетов
- `net.if.out[all,errors]` - Ошибки исходящих пакетов

## Настройка Zabbix

### 1. Доступ к Web интерфейсу

После запуска docker-compose:
- URL: http://localhost:8080
- Логин: `Admin`
- Пароль: `zabbix`

### 2. Создание хоста для мониторинга

1. Перейдите в **Configuration → Hosts**
2. Нажмите **Create host**
3. Заполните:
   - **Host name**: `test_host` (или значение `--zabbix-host`)
   - **Groups**: выберите существующую группу или создайте новую
   - **Interfaces**: можно оставить пустым для trapper элементов

### 3. Автоматическое создание элементов

Утилита автоматически создает все необходимые элементы данных при первом запуске. 
Элементы будут иметь тип **Zabbix trapper**, что позволяет отправлять данные через Sender протокол.

## Разработка

### Структура проекта

```
zabbix_mon/
├── cmd/monitor/           # Точка входа CLI
├── internal/
│   ├── collector/         # Сбор системных метрик
│   ├── zabbix/           # Клиент Zabbix API и Sender
│   │   ├── client.go     # API клиент
│   │   ├── sender.go     # Zabbix Sender протокол
│   │   └── types.go      # Типы данных
│   ├── scheduler/        # Планировщик задач
│   ├── config/           # Конфигурация
│   └── logger/           # Логирование
├── docker-compose.yml    # Docker окружение
├── Dockerfile           # Сборка образа
└── README.md           # Документация
```

### Сборка из исходников

```bash
# Клонирование
git clone https://github.com/yourusername/zabbix_mon.git
cd zabbix_mon

# Установка зависимостей
go mod download

# Сборка
go build -o monitor ./cmd/monitor

# Запуск
./monitor --help
```

### Запуск тестов

```bash
# Юнит тесты
go test ./...

# Тесты с покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Docker сборка

```bash
# Сборка образа
docker build -t zabbix-monitor .

# Запуск
docker run --rm zabbix-monitor --help
```

## Примеры использования

### Базовый мониторинг

```bash
monitor \
  --zabbix-url="http://zabbix.company.com/api_jsonrpc.php" \
  --zabbix-user="monitoring" \
  --zabbix-password="secret" \
  --zabbix-host="$(hostname)" \
  --interval=30
```

### Продакшен с переменными окружения

```bash
export ZABBIX_URL="https://zabbix.prod.com/api_jsonrpc.php"
export ZABBIX_USER="prod_monitor"
export ZABBIX_PASSWORD="$PROD_ZABBIX_PASSWORD"
export ZABBIX_HOST="$(hostname)"
export LOG_LEVEL="warn"

monitor
```

### Systemd сервис

Создайте файл `/etc/systemd/system/zabbix-monitor.service`:

```ini
[Unit]
Description=Zabbix Monitoring CLI
After=network.target

[Service]
Type=simple
User=monitor
Group=monitor
Environment="ZABBIX_URL=http://zabbix.local/api_jsonrpc.php"
Environment="ZABBIX_USER=Admin"
Environment="ZABBIX_PASSWORD=zabbix"
Environment="ZABBIX_HOST=%H"
Environment="LOG_LEVEL=info"
ExecStart=/usr/local/bin/monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Активация:
```bash
sudo systemctl daemon-reload
sudo systemctl enable zabbix-monitor
sudo systemctl start zabbix-monitor
```

## Профилирование производительности

Утилита поддерживает профилирование для анализа производительности с помощью pprof.

### Включение профилирования

#### HTTP сервер pprof

```bash
# Запуск с HTTP сервером профилирования на порту 6060
monitor --profile --profile-http-port=6060 \
  --zabbix-url="http://localhost:8080/api_jsonrpc.php" \
  --zabbix-user="Admin" \
  --zabbix-password="zabbix" \
  --zabbix-host="test_host"
```

Доступные endpoints:
- http://localhost:6060/ - информация о доступных профилях
- http://localhost:6060/debug/pprof/ - интерфейс pprof
- http://localhost:6060/debug/pprof/heap - профиль памяти
- http://localhost:6060/debug/pprof/profile - CPU профиль (30 сек)
- http://localhost:6060/debug/pprof/goroutine - профиль горутин

#### Сохранение профилей в файлы

```bash
# CPU профиль в файл на 60 секунд
monitor --profile --profile-cpu=cpu.prof --profile-time=60 \
  --zabbix-url="http://localhost:8080/api_jsonrpc.php" \
  --zabbix-user="Admin" \
  --zabbix-password="zabbix" \
  --zabbix-host="test_host"

# Профиль памяти (сохраняется при завершении)
monitor --profile --profile-mem=mem.prof \
  --zabbix-url="http://localhost:8080/api_jsonrpc.php" \
  --zabbix-user="Admin" \
  --zabbix-password="zabbix" \
  --zabbix-host="test_host"
```

#### Переменные окружения

```bash
export PROFILE_ENABLE=true
export PROFILE_HTTP_PORT=6060
export PROFILE_CPU_FILE=cpu.prof
export PROFILE_MEM_FILE=mem.prof
export PROFILE_TIME=30

monitor
```

### Анализ профилей

#### Интерактивный анализ

```bash
# Анализ CPU профиля
go tool pprof http://localhost:6060/debug/pprof/profile

# Анализ памяти
go tool pprof http://localhost:6060/debug/pprof/heap

# Из файла
go tool pprof cpu.prof
go tool pprof mem.prof
```

#### Веб-интерфейс

```bash
# CPU профиль с веб-интерфейсом
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile

# Профиль памяти
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
```

#### Командная строка

```bash
# Топ функций по CPU
go tool pprof -top http://localhost:6060/debug/pprof/profile

# Топ функций по памяти
go tool pprof -top http://localhost:6060/debug/pprof/heap

# Граф вызовов
go tool pprof -svg http://localhost:6060/debug/pprof/profile > cpu.svg
```

### Мониторинг статистик памяти

При включенном профилировании утилита автоматически логирует статистику памяти каждые 10 циклов сбора:

```
INFO    Memory statistics
        {"alloc_mb": 2, "total_alloc_mb": 15, "sys_mb": 8, "num_gc": 3, "goroutines": 12}
```

Параметры:
- `alloc_mb` - текущее потребление памяти (MB)
- `total_alloc_mb` - общее количество выделенной памяти (MB)
- `sys_mb` - память выделенная от ОС (MB) 
- `num_gc` - количество циклов сборки мусора
- `goroutines` - количество активных горутин

### Полные флаги профилирования

| Флаг | Описание | По умолчанию |
|------|----------|-------------|
| `--profile` | Включить профилирование | `false` |
| `--profile-http-port` | Порт HTTP сервера pprof | `6060` |
| `--profile-cpu` | Файл CPU профиля | "" (отключено) |
| `--profile-mem` | Файл профиля памяти | "" (отключено) |
| `--profile-time` | Время записи CPU профиля (сек) | `30` |


### Отладка

Включите детальное логирование:

```bash
monitor --log-level=debug
```

Проверьте подключение к Zabbix:

```bash
# Проверка API (должен вернуть версию)
curl -X POST http://localhost:8080/api_jsonrpc.php \
  -H "Content-Type: application/json-rpc" \
  -d '{"jsonrpc":"2.0","method":"apiinfo.version","params":{},"id":1}'

# Проверка порта 10051 (Zabbix Server)
telnet localhost 10051
```

Проверьте логи Docker Compose:

```bash
docker-compose logs monitor-cli
docker-compose logs zabbix-server
```

### Тестирование отправки данных

Можно протестировать отправку тестовых данных:

```bash
# Простая проверка Zabbix Sender (если установлен)
echo "test_host system.cpu.util[,idle] $(date +%s) 25.5" | zabbix_sender -z localhost -T -i -
```



