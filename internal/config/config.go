package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	// Zabbix настройки
	ZabbixURL      string
	ZabbixUser     string
	ZabbixPassword string
	ZabbixHost     string

	// Общие настройки
	Interval  time.Duration
	LogLevel  string
	BatchSize int

	// HTTP клиент настройки
	HTTPTimeout      time.Duration
	MaxRetries       int
	RetryBackoffBase time.Duration

	// Профилирование
	ProfileEnable   bool
	ProfileHTTPPort int
	ProfileCPUFile  string
	ProfileMemFile  string
	ProfileTime     int
}

// NewConfig создает новую конфигурацию с значениями по умолчанию
func NewConfig() *Config {
	return &Config{
		ZabbixURL:        "http://localhost:10051/api_jsonrpc.php",
		ZabbixUser:       "Admin",
		ZabbixPassword:   "zabbix",
		ZabbixHost:       "monitoring-host",
		Interval:         10 * time.Second,
		LogLevel:         "info",
		BatchSize:        50,
		HTTPTimeout:      30 * time.Second,
		MaxRetries:       3,
		RetryBackoffBase: 1 * time.Second,
		ProfileEnable:    false,
		ProfileHTTPPort:  6060,
		ProfileCPUFile:   "",
		ProfileMemFile:   "",
		ProfileTime:      30,
	}
}

// Load загружает конфигурацию из флагов командной строки и переменных окружения
func (c *Config) Load(cmd *cobra.Command) error {
	// Загружаем из переменных окружения сначала
	c.loadFromEnv()

	// Затем из флагов (они имеют приоритет)
	if cmd.Flags().Changed("zabbix-url") {
		c.ZabbixURL, _ = cmd.Flags().GetString("zabbix-url")
	}
	if cmd.Flags().Changed("zabbix-user") {
		c.ZabbixUser, _ = cmd.Flags().GetString("zabbix-user")
	}
	if cmd.Flags().Changed("zabbix-password") {
		c.ZabbixPassword, _ = cmd.Flags().GetString("zabbix-password")
	}
	if cmd.Flags().Changed("zabbix-host") {
		c.ZabbixHost, _ = cmd.Flags().GetString("zabbix-host")
	}
	if cmd.Flags().Changed("interval") {
		intervalSec, _ := cmd.Flags().GetInt("interval")
		c.Interval = time.Duration(intervalSec) * time.Second
	}
	if cmd.Flags().Changed("log-level") {
		c.LogLevel, _ = cmd.Flags().GetString("log-level")
	}
	if cmd.Flags().Changed("batch-size") {
		c.BatchSize, _ = cmd.Flags().GetInt("batch-size")
	}
	if cmd.Flags().Changed("profile") {
		c.ProfileEnable, _ = cmd.Flags().GetBool("profile")
	}
	if cmd.Flags().Changed("profile-http-port") {
		c.ProfileHTTPPort, _ = cmd.Flags().GetInt("profile-http-port")
	}
	if cmd.Flags().Changed("profile-cpu") {
		c.ProfileCPUFile, _ = cmd.Flags().GetString("profile-cpu")
	}
	if cmd.Flags().Changed("profile-mem") {
		c.ProfileMemFile, _ = cmd.Flags().GetString("profile-mem")
	}
	if cmd.Flags().Changed("profile-time") {
		c.ProfileTime, _ = cmd.Flags().GetInt("profile-time")
	}

	return c.Validate()
}

// loadFromEnv загружает конфигурацию из переменных окружения
func (c *Config) loadFromEnv() {
	if url := os.Getenv("ZABBIX_URL"); url != "" {
		c.ZabbixURL = url
	}
	if user := os.Getenv("ZABBIX_USER"); user != "" {
		c.ZabbixUser = user
	}
	if pass := os.Getenv("ZABBIX_PASSWORD"); pass != "" {
		c.ZabbixPassword = pass
	}
	if host := os.Getenv("ZABBIX_HOST"); host != "" {
		c.ZabbixHost = host
	}
	if intervalStr := os.Getenv("INTERVAL"); intervalStr != "" {
		if intervalSec, err := strconv.Atoi(intervalStr); err == nil {
			c.Interval = time.Duration(intervalSec) * time.Second
		}
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}
	if batchSizeStr := os.Getenv("BATCH_SIZE"); batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil {
			c.BatchSize = batchSize
		}
	}
	if profileStr := os.Getenv("PROFILE_ENABLE"); profileStr != "" {
		if profile, err := strconv.ParseBool(profileStr); err == nil {
			c.ProfileEnable = profile
		}
	}
	if profilePortStr := os.Getenv("PROFILE_HTTP_PORT"); profilePortStr != "" {
		if port, err := strconv.Atoi(profilePortStr); err == nil {
			c.ProfileHTTPPort = port
		}
	}
	if cpuFile := os.Getenv("PROFILE_CPU_FILE"); cpuFile != "" {
		c.ProfileCPUFile = cpuFile
	}
	if memFile := os.Getenv("PROFILE_MEM_FILE"); memFile != "" {
		c.ProfileMemFile = memFile
	}
	if profileTimeStr := os.Getenv("PROFILE_TIME"); profileTimeStr != "" {
		if profileTime, err := strconv.Atoi(profileTimeStr); err == nil {
			c.ProfileTime = profileTime
		}
	}
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.ZabbixURL == "" {
		return fmt.Errorf("zabbix URL is required")
	}
	if c.ZabbixUser == "" {
		return fmt.Errorf("zabbix user is required")
	}
	if c.ZabbixPassword == "" {
		return fmt.Errorf("zabbix password is required")
	}
	if c.ZabbixHost == "" {
		return fmt.Errorf("zabbix host is required")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	// Проверяем уровень логирования
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	// Валидация профилирования
	if c.ProfileEnable {
		if c.ProfileHTTPPort <= 0 || c.ProfileHTTPPort > 65535 {
			return fmt.Errorf("invalid profile HTTP port: %d", c.ProfileHTTPPort)
		}
		if c.ProfileTime <= 0 {
			return fmt.Errorf("profile time must be positive")
		}
	}

	return nil
}

// AddFlags добавляет флаги в cobra команду
func AddFlags(cmd *cobra.Command) {
	cmd.Flags().String("zabbix-url", "", "Zabbix API URL")
	cmd.Flags().String("zabbix-user", "", "Zabbix username")
	cmd.Flags().String("zabbix-password", "", "Zabbix password")
	cmd.Flags().String("zabbix-host", "", "Host name in Zabbix")
	cmd.Flags().Int("interval", 10, "Collection interval in seconds")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().Int("batch-size", 50, "Batch size for sending metrics")

	// Флаги профилирования
	cmd.Flags().Bool("profile", false, "Enable profiling")
	cmd.Flags().Int("profile-http-port", 6060, "HTTP port for pprof endpoints")
	cmd.Flags().String("profile-cpu", "", "CPU profile output file")
	cmd.Flags().String("profile-mem", "", "Memory profile output file")
	cmd.Flags().Int("profile-time", 30, "CPU profile duration in seconds")
}
