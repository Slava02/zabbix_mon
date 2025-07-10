package collector

import "time"

// MetricSet содержит все собранные метрики системы
type MetricSet struct {
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Disk      DiskMetrics    `json:"disk"`
	Network   NetworkMetrics `json:"network"`
}

// CPUMetrics содержит метрики процессора
type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	LoadAvg1     float64 `json:"load_avg_1"`
	LoadAvg5     float64 `json:"load_avg_5"`
	LoadAvg15    float64 `json:"load_avg_15"`
}

// MemoryMetrics содержит метрики памяти
type MemoryMetrics struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	AvailableBytes uint64  `json:"available_bytes"`
}

// DiskMetrics содержит метрики диска
type DiskMetrics struct {
	TotalBytes        uint64  `json:"total_bytes"`
	UsedBytes         uint64  `json:"used_bytes"`
	FreeBytes         uint64  `json:"free_bytes"`
	UsagePercent      float64 `json:"usage_percent"`
	InodesTotal       uint64  `json:"inodes_total"`
	InodesUsed        uint64  `json:"inodes_used"`
	InodesFree        uint64  `json:"inodes_free"`
	InodesUsedPercent float64 `json:"inodes_used_percent"`
}

// NetworkMetrics содержит метрики сети
type NetworkMetrics struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrorsIn    uint64 `json:"errors_in"`
	ErrorsOut   uint64 `json:"errors_out"`
	DropsIn     uint64 `json:"drops_in"`
	DropsOut    uint64 `json:"drops_out"`
}
