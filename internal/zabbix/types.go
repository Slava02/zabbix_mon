package zabbix

import "encoding/json"

// JSONRPCRequest представляет JSON-RPC запрос к Zabbix API
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Auth    string      `json:"auth,omitempty"`
	ID      int         `json:"id"`
}

// JSONRPCResponse представляет JSON-RPC ответ от Zabbix API
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// JSONRPCError представляет ошибку JSON-RPC
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// LoginParams параметры для авторизации
type LoginParams struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// HostGetParams параметры для получения хоста
type HostGetParams struct {
	Output []string          `json:"output"`
	Filter map[string]string `json:"filter"`
}

// Host представляет хост в Zabbix
type Host struct {
	HostID string `json:"hostid"`
	Host   string `json:"host"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ItemGetParams параметры для получения элементов данных
type ItemGetParams struct {
	Output  []string          `json:"output"`
	HostIDs []string          `json:"hostids"`
	Filter  map[string]string `json:"filter,omitempty"`
}

// Item представляет элемент данных в Zabbix
type Item struct {
	ItemID      string `json:"itemid"`
	Name        string `json:"name"`
	Key         string `json:"key_"`
	HostID      string `json:"hostid"`
	Status      string `json:"status"`
	ValueType   string `json:"value_type"`
	DataType    string `json:"data_type"`
	Description string `json:"description"`
}

// ItemCreateParams параметры для создания элемента данных
type ItemCreateParams struct {
	Name        string `json:"name"`
	Key         string `json:"key_"`
	HostID      string `json:"hostid"`
	Type        int    `json:"type"`       // 2 - Zabbix trapper
	ValueType   int    `json:"value_type"` // 0 - float, 3 - unsigned int
	DataType    int    `json:"data_type"`  // 0 - decimal
	Description string `json:"description,omitempty"`
	Status      int    `json:"status"` // 0 - enabled
}

// HistoryData представляет исторические данные для отправки
type HistoryData struct {
	ItemID string      `json:"itemid"`
	Clock  int64       `json:"clock"`
	Value  interface{} `json:"value"`
	NS     int64       `json:"ns,omitempty"`
}

// HistoryCreateParams параметры для создания исторических данных
type HistoryCreateParams struct {
	Items []HistoryData `json:"items"`
}

// ZabbixMetricItem представляет элемент данных для Zabbix
type ZabbixMetricItem struct {
	Key         string
	Name        string
	ValueType   int // 0 - float, 3 - unsigned int
	Description string
}

// GetZabbixItems возвращает список всех метрик, которые должны быть созданы в Zabbix
func GetZabbixItems() []ZabbixMetricItem {
	return []ZabbixMetricItem{
		// CPU метрики
		{
			Key:         "system.cpu.util[,idle]",
			Name:        "CPU utilization",
			ValueType:   0, // float
			Description: "CPU usage percentage",
		},
		{
			Key:         "system.cpu.load[percpu,avg1]",
			Name:        "Processor load (1 min average per core)",
			ValueType:   0, // float
			Description: "1 minute load average",
		},
		{
			Key:         "system.cpu.load[percpu,avg5]",
			Name:        "Processor load (5 min average per core)",
			ValueType:   0, // float
			Description: "5 minute load average",
		},
		{
			Key:         "system.cpu.load[percpu,avg15]",
			Name:        "Processor load (15 min average per core)",
			ValueType:   0, // float
			Description: "15 minute load average",
		},

		// Memory метрики
		{
			Key:         "vm.memory.size[total]",
			Name:        "Total memory",
			ValueType:   3, // unsigned int
			Description: "Total memory in bytes",
		},
		{
			Key:         "vm.memory.size[used]",
			Name:        "Used memory",
			ValueType:   3, // unsigned int
			Description: "Used memory in bytes",
		},
		{
			Key:         "vm.memory.size[available]",
			Name:        "Available memory",
			ValueType:   3, // unsigned int
			Description: "Available memory in bytes",
		},
		{
			Key:         "vm.memory.util",
			Name:        "Memory utilization",
			ValueType:   0, // float
			Description: "Memory usage percentage",
		},

		// Disk метрики
		{
			Key:         "vfs.fs.size[/,total]",
			Name:        "Free disk space on / (total)",
			ValueType:   3, // unsigned int
			Description: "Total disk space in bytes",
		},
		{
			Key:         "vfs.fs.size[/,used]",
			Name:        "Used disk space on /",
			ValueType:   3, // unsigned int
			Description: "Used disk space in bytes",
		},
		{
			Key:         "vfs.fs.size[/,free]",
			Name:        "Free disk space on /",
			ValueType:   3, // unsigned int
			Description: "Free disk space in bytes",
		},
		{
			Key:         "vfs.fs.pused[/]",
			Name:        "Free disk space on / (percentage used)",
			ValueType:   0, // float
			Description: "Disk usage percentage",
		},

		// Network метрики
		{
			Key:         "net.if.in[all]",
			Name:        "Incoming network traffic on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Bytes received on all network interfaces",
		},
		{
			Key:         "net.if.out[all]",
			Name:        "Outgoing network traffic on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Bytes sent on all network interfaces",
		},
		{
			Key:         "net.if.in[all,packets]",
			Name:        "Incoming packets on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Packets received on all network interfaces",
		},
		{
			Key:         "net.if.out[all,packets]",
			Name:        "Outgoing packets on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Packets sent on all network interfaces",
		},
		{
			Key:         "net.if.in[all,errors]",
			Name:        "Incoming errors on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Input errors on all network interfaces",
		},
		{
			Key:         "net.if.out[all,errors]",
			Name:        "Outgoing errors on all interfaces",
			ValueType:   3, // unsigned int
			Description: "Output errors on all network interfaces",
		},
	}
}
