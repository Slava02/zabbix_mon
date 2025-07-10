package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// Collector отвечает за сбор системных метрик
type Collector struct {
	logger *zap.Logger
}

// New создает новый экземпляр сборщика метрик
func New(logger *zap.Logger) *Collector {
	return &Collector{
		logger: logger,
	}
}

// Collect собирает все системные метрики
func (c *Collector) Collect(ctx context.Context) (*MetricSet, error) {
	c.logger.Debug("Starting metrics collection")

	metrics := &MetricSet{
		Timestamp: time.Now(),
	}

	// Используем каналы для параллельного сбора метрик
	type result struct {
		name string
		err  error
	}

	results := make(chan result, 4)

	// Собираем CPU метрики
	go func() {
		cpuMetrics, err := c.collectCPU(ctx)
		if err != nil {
			results <- result{name: "CPU", err: err}
			return
		}
		metrics.CPU = *cpuMetrics
		results <- result{name: "CPU", err: nil}
	}()

	// Собираем метрики памяти
	go func() {
		memMetrics, err := c.collectMemory(ctx)
		if err != nil {
			results <- result{name: "Memory", err: err}
			return
		}
		metrics.Memory = *memMetrics
		results <- result{name: "Memory", err: nil}
	}()

	// Собираем метрики диска
	go func() {
		diskMetrics, err := c.collectDisk(ctx)
		if err != nil {
			results <- result{name: "Disk", err: err}
			return
		}
		metrics.Disk = *diskMetrics
		results <- result{name: "Disk", err: nil}
	}()

	// Собираем метрики сети
	go func() {
		netMetrics, err := c.collectNetwork(ctx)
		if err != nil {
			results <- result{name: "Network", err: err}
			return
		}
		metrics.Network = *netMetrics
		results <- result{name: "Network", err: nil}
	}()

	// Ждем завершения всех горутин
	var errors []string
	for i := 0; i < 4; i++ {
		select {
		case res := <-results:
			if res.err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", res.name, res.err))
				c.logger.Warn("Failed to collect metrics",
					zap.String("component", res.name),
					zap.Error(res.err))
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if len(errors) == 4 {
		return nil, fmt.Errorf("failed to collect all metrics: %v", errors)
	}

	c.logger.Debug("Metrics collection completed",
		zap.Int("errors", len(errors)),
		zap.Time("timestamp", metrics.Timestamp))

	return metrics, nil
}

// collectCPU собирает метрики процессора
func (c *Collector) collectCPU(ctx context.Context) (*CPUMetrics, error) {
	// CPU Usage
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percentage: %w", err)
	}

	var cpuUsage float64
	if len(percentages) > 0 {
		cpuUsage = percentages[0]
	}

	// Load Average
	loadAvg, err := load.AvgWithContext(ctx)
	if err != nil {
		c.logger.Warn("Failed to get load average", zap.Error(err))
		// Load average не критично, продолжаем без него
	}

	metrics := &CPUMetrics{
		UsagePercent: cpuUsage,
	}

	if loadAvg != nil {
		metrics.LoadAvg1 = loadAvg.Load1
		metrics.LoadAvg5 = loadAvg.Load5
		metrics.LoadAvg15 = loadAvg.Load15
	}

	return metrics, nil
}

// collectMemory собирает метрики памяти
func (c *Collector) collectMemory(ctx context.Context) (*MemoryMetrics, error) {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory statistics: %w", err)
	}

	return &MemoryMetrics{
		TotalBytes:     vmStat.Total,
		UsedBytes:      vmStat.Used,
		FreeBytes:      vmStat.Free,
		UsagePercent:   vmStat.UsedPercent,
		AvailableBytes: vmStat.Available,
	}, nil
}

// collectDisk собирает метрики диска для корневого раздела
func (c *Collector) collectDisk(ctx context.Context) (*DiskMetrics, error) {
	diskStat, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk statistics: %w", err)
	}

	return &DiskMetrics{
		TotalBytes:        diskStat.Total,
		UsedBytes:         diskStat.Used,
		FreeBytes:         diskStat.Free,
		UsagePercent:      diskStat.UsedPercent,
		InodesTotal:       diskStat.InodesTotal,
		InodesUsed:        diskStat.InodesUsed,
		InodesFree:        diskStat.InodesFree,
		InodesUsedPercent: diskStat.InodesUsedPercent,
	}, nil
}

// collectNetwork собирает метрики сети
func (c *Collector) collectNetwork(ctx context.Context) (*NetworkMetrics, error) {
	netStats, err := net.IOCountersWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get network statistics: %w", err)
	}

	if len(netStats) == 0 {
		return &NetworkMetrics{}, nil
	}

	// Суммируем статистику по всем интерфейсам
	metrics := &NetworkMetrics{}
	for _, stat := range netStats {
		metrics.BytesSent += stat.BytesSent
		metrics.BytesRecv += stat.BytesRecv
		metrics.PacketsSent += stat.PacketsSent
		metrics.PacketsRecv += stat.PacketsRecv
		metrics.ErrorsIn += stat.Errin
		metrics.ErrorsOut += stat.Errout
		metrics.DropsIn += stat.Dropin
		metrics.DropsOut += stat.Dropout
	}

	return metrics, nil
}
