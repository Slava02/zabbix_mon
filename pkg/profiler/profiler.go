package profiler

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"
)

// Config представляет конфигурацию профилировщика
type Config struct {
	Enable      bool   // включить профилирование
	HTTPPort    int    // порт для HTTP сервера pprof
	CPUProfile  string // путь к файлу CPU профиля
	MemProfile  string // путь к файлу профиля памяти
	ProfileTime int    // время записи CPU профиля в секундах
}

// Profiler управляет профилированием приложения
type Profiler struct {
	config     Config
	logger     *zap.Logger
	httpServer *http.Server
	cpuFile    *os.File
}

// New создает новый профилировщик
func New(config Config, logger *zap.Logger) *Profiler {
	return &Profiler{
		config: config,
		logger: logger,
	}
}

// Start запускает профилирование
func (p *Profiler) Start(ctx context.Context) error {
	if !p.config.Enable {
		p.logger.Info("Profiling disabled")
		return nil
	}

	p.logger.Info("Starting profiler",
		zap.Int("http_port", p.config.HTTPPort),
		zap.String("cpu_profile", p.config.CPUProfile),
		zap.String("mem_profile", p.config.MemProfile))

	// Запускаем HTTP сервер для pprof
	if err := p.startHTTPServer(ctx); err != nil {
		return fmt.Errorf("failed to start pprof HTTP server: %w", err)
	}

	// Запускаем CPU профилирование в файл
	if p.config.CPUProfile != "" {
		if err := p.startCPUProfile(); err != nil {
			return fmt.Errorf("failed to start CPU profiling: %w", err)
		}
	}

	return nil
}

// Stop останавливает профилирование
func (p *Profiler) Stop() error {
	var errors []error

	// Останавливаем CPU профилирование
	if err := p.stopCPUProfile(); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop CPU profiling: %w", err))
	}

	// Сохраняем профиль памяти
	if p.config.MemProfile != "" {
		if err := p.writeMemProfile(); err != nil {
			errors = append(errors, fmt.Errorf("failed to write memory profile: %w", err))
		}
	}

	// Останавливаем HTTP сервер
	if p.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.httpServer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown HTTP server: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("profiler shutdown errors: %v", errors)
	}

	p.logger.Info("Profiler stopped")
	return nil
}

// startHTTPServer запускает HTTP сервер для pprof endpoints
func (p *Profiler) startHTTPServer(ctx context.Context) error {
	if p.config.HTTPPort <= 0 {
		return nil
	}

	mux := http.NewServeMux()

	// Добавляем pprof endpoints
	mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	// Добавляем информационный endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
Zabbix Monitor Profiler

Available endpoints:
- /debug/pprof/          - pprof index
- /debug/pprof/cmdline   - command line
- /debug/pprof/profile   - CPU profile (30s)
- /debug/pprof/symbol    - symbol lookup
- /debug/pprof/trace     - execution trace (1s)
- /debug/pprof/heap      - heap profile
- /debug/pprof/goroutine - goroutine profile
- /debug/pprof/block     - block profile
- /debug/pprof/mutex     - mutex profile

Usage examples:
go tool pprof http://localhost:%d/debug/pprof/profile
go tool pprof http://localhost:%d/debug/pprof/heap
`, p.config.HTTPPort, p.config.HTTPPort)
	})

	p.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.HTTPPort),
		Handler: mux,
	}

	go func() {
		p.logger.Info("Starting pprof HTTP server",
			zap.String("addr", p.httpServer.Addr))

		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error("pprof HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// startCPUProfile начинает CPU профилирование в файл
func (p *Profiler) startCPUProfile() error {
	file, err := os.Create(p.config.CPUProfile)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	p.cpuFile = file

	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	p.logger.Info("Started CPU profiling", zap.String("file", p.config.CPUProfile))

	// Автоматически останавливаем через заданное время
	if p.config.ProfileTime > 0 {
		go func() {
			time.Sleep(time.Duration(p.config.ProfileTime) * time.Second)
			p.stopCPUProfile()
		}()
	}

	return nil
}

// stopCPUProfile останавливает CPU профилирование
func (p *Profiler) stopCPUProfile() error {
	if p.cpuFile == nil {
		return nil
	}

	pprof.StopCPUProfile()

	if err := p.cpuFile.Close(); err != nil {
		return fmt.Errorf("failed to close CPU profile file: %w", err)
	}

	p.logger.Info("Stopped CPU profiling", zap.String("file", p.config.CPUProfile))
	p.cpuFile = nil
	return nil
}

// writeMemProfile записывает профиль памяти в файл
func (p *Profiler) writeMemProfile() error {
	file, err := os.Create(p.config.MemProfile)
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer file.Close()

	// Принудительно запускаем GC для точного профиля памяти
	runtime.GC()

	if err := pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	p.logger.Info("Written memory profile", zap.String("file", p.config.MemProfile))
	return nil
}

// GetMemStats возвращает статистику памяти
func (p *Profiler) GetMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// LogMemStats логирует статистику памяти
func (p *Profiler) LogMemStats() {
	if !p.config.Enable {
		return
	}

	m := p.GetMemStats()
	p.logger.Info("Memory statistics",
		zap.Uint64("alloc_mb", m.Alloc/1024/1024),
		zap.Uint64("total_alloc_mb", m.TotalAlloc/1024/1024),
		zap.Uint64("sys_mb", m.Sys/1024/1024),
		zap.Uint32("num_gc", m.NumGC),
		zap.Int("goroutines", runtime.NumGoroutine()),
	)
}
