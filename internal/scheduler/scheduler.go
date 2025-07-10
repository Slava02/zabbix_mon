package scheduler

import (
	"context"
	"fmt"
	"time"

	"zabbix_mon/internal/collector"
	"zabbix_mon/internal/config"
	"zabbix_mon/internal/zabbix"

	"go.uber.org/zap"
)

// Scheduler отвечает за планирование и координацию работы
type Scheduler struct {
	config    *config.Config
	collector *collector.Collector
	zabbix    *zabbix.Client
	logger    *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

// New создает новый планировщик
func New(cfg *config.Config, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		config:    cfg,
		collector: collector.New(logger),
		zabbix:    zabbix.NewClient(cfg.ZabbixURL, cfg.ZabbixUser, cfg.ZabbixPassword, cfg.HTTPTimeout, logger),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start запускает планировщик
func (s *Scheduler) Start() error {
	s.logger.Info("Starting scheduler",
		zap.Duration("interval", s.config.Interval),
		zap.String("zabbix_host", s.config.ZabbixHost))

	// Инициализируем Zabbix клиент
	if err := s.zabbix.Initialize(s.ctx, s.config.ZabbixHost); err != nil {
		return fmt.Errorf("failed to initialize Zabbix client: %w", err)
	}

	// Запускаем основной цикл мониторинга
	go s.monitoringLoop()

	s.logger.Info("Scheduler started successfully")
	return nil
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	s.cancel()
}

// Wait ожидает завершения планировщика
func (s *Scheduler) Wait() {
	<-s.ctx.Done()
}

// monitoringLoop основной цикл мониторинга
func (s *Scheduler) monitoringLoop() {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	// Выполняем первый сбор сразу
	s.collectAndSend()

	for {
		select {
		case <-ticker.C:
			s.collectAndSend()
		case <-s.ctx.Done():
			s.logger.Info("Monitoring loop stopped")
			return
		}
	}
}

// collectAndSend собирает метрики и отправляет их в Zabbix
func (s *Scheduler) collectAndSend() {
	start := time.Now()

	// Создаем контекст с таймаутом для операций
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Собираем метрики
	metrics, err := s.collector.Collect(ctx)
	if err != nil {
		s.logger.Error("Failed to collect metrics", zap.Error(err))
		return
	}

	collectDuration := time.Since(start)

	// Отправляем метрики в Zabbix
	sendStart := time.Now()
	if err := s.sendMetricsWithRetry(ctx, metrics); err != nil {
		s.logger.Error("Failed to send metrics after retries", zap.Error(err))
		return
	}

	sendDuration := time.Since(sendStart)
	totalDuration := time.Since(start)

	s.logger.Info("Metrics processed successfully",
		zap.Duration("collect_time", collectDuration),
		zap.Duration("send_time", sendDuration),
		zap.Duration("total_time", totalDuration),
		zap.Time("timestamp", metrics.Timestamp))
}

// sendMetricsWithRetry отправляет метрики с повторными попытками
func (s *Scheduler) sendMetricsWithRetry(ctx context.Context, metrics *collector.MetricSet) error {
	var lastErr error
	backoff := s.config.RetryBackoffBase

	for attempt := 0; attempt < s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Warn("Retrying metric send",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", s.config.MaxRetries),
				zap.Duration("backoff", backoff))

			// Ждем с экспоненциальным back-off
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}

			backoff *= 2 // экспоненциальное увеличение
		}

		err := s.zabbix.SendMetrics(ctx, metrics)
		if err == nil {
			if attempt > 0 {
				s.logger.Info("Metrics sent successfully after retry",
					zap.Int("attempts", attempt+1))
			}
			return nil
		}

		lastErr = err
		s.logger.Warn("Failed to send metrics",
			zap.Error(err),
			zap.Int("attempt", attempt+1))

		// Если это ошибка аутентификации, пытаемся переподключиться
		if s.isAuthError(err) && attempt < s.config.MaxRetries-1 {
			s.logger.Info("Authentication error detected, re-initializing Zabbix client")
			if reInitErr := s.zabbix.Initialize(ctx, s.config.ZabbixHost); reInitErr != nil {
				s.logger.Error("Failed to re-initialize Zabbix client", zap.Error(reInitErr))
			}
		}
	}

	return fmt.Errorf("failed to send metrics after %d attempts: %w", s.config.MaxRetries, lastErr)
}

// isAuthError проверяет, является ли ошибка ошибкой аутентификации
func (s *Scheduler) isAuthError(err error) bool {
	errStr := err.Error()
	// Проверяем на типичные ошибки аутентификации Zabbix
	return err != nil && (fmt.Sprintf("%v", err) == "Session terminated, re-login, please." ||
		fmt.Sprintf("%v", err) == "Not authorized" ||
		fmt.Sprintf("%v", err) == "Invalid params" &&
			(fmt.Sprintf("%v", errStr) == "Session terminated" ||
				fmt.Sprintf("%v", errStr) == "invalid auth token"))
}

// GetStats возвращает статистику работы
func (s *Scheduler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"interval":    s.config.Interval.String(),
		"zabbix_url":  s.config.ZabbixURL,
		"zabbix_host": s.config.ZabbixHost,
		"running":     s.ctx.Err() == nil,
	}
}
