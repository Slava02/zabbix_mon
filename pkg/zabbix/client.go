package zabbix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"zabbix_mon/internal/collector"

	"go.uber.org/zap"
)

// Client представляет клиент для работы с Zabbix API
type Client struct {
	url        string
	user       string
	password   string
	httpClient *http.Client
	logger     *zap.Logger

	authToken  string
	authMutex  sync.RWMutex
	hostID     string
	hostName   string            // Добавляем имя хоста для sender
	items      map[string]string // key -> itemID mapping
	itemsMutex sync.RWMutex

	requestID int
	idMutex   sync.Mutex

	// Zabbix Sender для отправки данных
	sender *Sender
}

// NewClient создает новый Zabbix клиент
func NewClient(url, user, password string, timeout time.Duration, logger *zap.Logger) *Client {
	return &Client{
		url:      url,
		user:     user,
		password: password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
		items:  make(map[string]string),
	}
}

// getZabbixServerHost извлекает хост сервера из URL API
func (c *Client) getZabbixServerHost() (string, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		return "", fmt.Errorf("failed to parse zabbix URL: %w", err)
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("cannot extract hostname from URL: %s", c.url)
	}

	return host, nil
}

// getNextRequestID возвращает следующий ID для запроса
func (c *Client) getNextRequestID() int {
	c.idMutex.Lock()
	defer c.idMutex.Unlock()
	c.requestID++
	return c.requestID
}

// makeRequest выполняет HTTP запрос к Zabbix API
func (c *Client) makeRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	c.authMutex.RLock()
	authToken := c.authToken
	c.authMutex.RUnlock()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		Auth:    authToken,
		ID:      c.getNextRequestID(),
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Debug("Making Zabbix API request",
		zap.String("method", method),
		zap.String("url", c.url))

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json-rpc")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("zabbix API error: %s (code: %d, data: %s)",
			response.Error.Message, response.Error.Code, response.Error.Data)
	}

	return &response, nil
}

// Login выполняет аутентификацию в Zabbix
func (c *Client) Login(ctx context.Context) error {
	c.logger.Info("Authenticating with Zabbix", zap.String("user", c.user))

	params := LoginParams{
		User:     c.user,
		Password: c.password,
	}

	resp, err := c.makeRequest(ctx, "user.login", params)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	var authToken string
	if err := json.Unmarshal(resp.Result, &authToken); err != nil {
		return fmt.Errorf("failed to parse auth token: %w", err)
	}

	c.authMutex.Lock()
	c.authToken = authToken
	c.authMutex.Unlock()

	c.logger.Info("Successfully authenticated with Zabbix")
	return nil
}

// findHost ищет хост по имени
func (c *Client) findHost(ctx context.Context, hostName string) error {
	c.logger.Info("Finding host in Zabbix", zap.String("host", hostName))

	params := HostGetParams{
		Output: []string{"hostid", "host", "name", "status"},
		Filter: map[string]string{
			"host": hostName,
		},
	}

	resp, err := c.makeRequest(ctx, "host.get", params)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	var hosts []Host
	if err := json.Unmarshal(resp.Result, &hosts); err != nil {
		return fmt.Errorf("failed to parse hosts: %w", err)
	}

	if len(hosts) == 0 {
		return fmt.Errorf("host '%s' not found in Zabbix", hostName)
	}

	c.hostID = hosts[0].HostID
	c.hostName = hostName // Сохраняем имя хоста для sender
	c.logger.Info("Found host",
		zap.String("hostID", c.hostID),
		zap.String("name", hosts[0].Name),
		zap.String("status", hosts[0].Status))

	return nil
}

// loadItems загружает существующие элементы данных для хоста
func (c *Client) loadItems(ctx context.Context) error {
	c.logger.Info("Loading existing items from Zabbix")

	params := ItemGetParams{
		Output:  []string{"itemid", "name", "key_", "status"},
		HostIDs: []string{c.hostID},
	}

	resp, err := c.makeRequest(ctx, "item.get", params)
	if err != nil {
		return fmt.Errorf("failed to get items: %w", err)
	}

	var items []Item
	if err := json.Unmarshal(resp.Result, &items); err != nil {
		return fmt.Errorf("failed to parse items: %w", err)
	}

	c.itemsMutex.Lock()
	defer c.itemsMutex.Unlock()

	c.items = make(map[string]string)
	for _, item := range items {
		c.items[item.Key] = item.ItemID
	}

	c.logger.Info("Loaded items", zap.Int("count", len(items)))
	return nil
}

// createMissingItems создает отсутствующие элементы данных
func (c *Client) createMissingItems(ctx context.Context) error {
	c.logger.Info("Creating missing items")

	zabbixItems := GetZabbixItems()
	var itemsToCreate []ItemCreateParams

	c.itemsMutex.RLock()
	for _, zItem := range zabbixItems {
		if _, exists := c.items[zItem.Key]; !exists {
			itemsToCreate = append(itemsToCreate, ItemCreateParams{
				Name:        zItem.Name,
				Key:         zItem.Key,
				HostID:      c.hostID,
				Type:        2, // Zabbix trapper
				ValueType:   zItem.ValueType,
				DataType:    0, // decimal
				Description: zItem.Description,
				Status:      0, // enabled
			})
		}
	}
	c.itemsMutex.RUnlock()

	if len(itemsToCreate) == 0 {
		c.logger.Info("All items already exist")
		return nil
	}

	c.logger.Info("Creating items", zap.Int("count", len(itemsToCreate)))

	resp, err := c.makeRequest(ctx, "item.create", itemsToCreate)
	if err != nil {
		return fmt.Errorf("failed to create items: %w", err)
	}

	var result map[string][]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse create result: %w", err)
	}

	itemIDs := result["itemids"]
	if len(itemIDs) != len(itemsToCreate) {
		return fmt.Errorf("unexpected number of created items: got %d, expected %d",
			len(itemIDs), len(itemsToCreate))
	}

	// Обновляем локальную карту элементов
	c.itemsMutex.Lock()
	for i, itemID := range itemIDs {
		c.items[itemsToCreate[i].Key] = itemID
	}
	c.itemsMutex.Unlock()

	c.logger.Info("Successfully created items", zap.Int("count", len(itemIDs)))
	return nil
}

// Initialize инициализирует клиент (авторизация, поиск хоста, создание элементов)
func (c *Client) Initialize(ctx context.Context, hostName string) error {
	c.logger.Info("Initializing Zabbix client")

	// Авторизация
	if err := c.Login(ctx); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Поиск хоста
	if err := c.findHost(ctx, hostName); err != nil {
		return fmt.Errorf("failed to find host: %w", err)
	}

	// Загрузка существующих элементов
	if err := c.loadItems(ctx); err != nil {
		return fmt.Errorf("failed to load items: %w", err)
	}

	// Создание недостающих элементов
	if err := c.createMissingItems(ctx); err != nil {
		return fmt.Errorf("failed to create missing items: %w", err)
	}

	// Инициализируем Zabbix Sender
	serverHost, err := c.getZabbixServerHost()
	if err != nil {
		return fmt.Errorf("failed to get zabbix server host: %w", err)
	}

	c.sender = NewSender(serverHost, 10051)
	c.logger.Info("Initialized Zabbix Sender", zap.String("server", serverHost))

	c.logger.Info("Zabbix client initialized successfully")
	return nil
}

// SendMetrics отправляет метрики в Zabbix через Sender протокол
func (c *Client) SendMetrics(ctx context.Context, metrics *collector.MetricSet) error {
	c.logger.Debug("Sending metrics to Zabbix via Sender")

	// Конвертируем метрики в формат Zabbix Sender
	senderMetrics := c.convertMetricsToSenderData(metrics)

	if len(senderMetrics) == 0 {
		c.logger.Warn("No metrics to send")
		return nil
	}

	// Создаем пакет и отправляем данные через Zabbix Sender
	packet := NewPacket(senderMetrics)
	_, err := c.sender.Send(packet)
	if err != nil {
		return fmt.Errorf("failed to send metrics via sender: %w", err)
	}

	c.logger.Debug("Successfully sent metrics", zap.Int("count", len(senderMetrics)))
	return nil
}

// convertMetricsToSenderData конвертирует собранные метрики в формат Zabbix Sender
func (c *Client) convertMetricsToSenderData(metrics *collector.MetricSet) []*Metric {
	c.itemsMutex.RLock()
	defer c.itemsMutex.RUnlock()

	var senderMetrics []*Metric
	timestamp := metrics.Timestamp.Unix()

	// Функция для добавления метрики
	addMetric := func(key string, value interface{}) {
		if _, exists := c.items[key]; exists {
			metric := NewMetric(c.hostName, key, fmt.Sprintf("%v", value), timestamp)
			senderMetrics = append(senderMetrics, metric)
		}
	}

	// CPU метрики
	addMetric("system.cpu.util[,idle]", metrics.CPU.UsagePercent)
	addMetric("system.cpu.load[percpu,avg1]", metrics.CPU.LoadAvg1)
	addMetric("system.cpu.load[percpu,avg5]", metrics.CPU.LoadAvg5)
	addMetric("system.cpu.load[percpu,avg15]", metrics.CPU.LoadAvg15)

	// Memory метрики
	addMetric("vm.memory.size[total]", metrics.Memory.TotalBytes)
	addMetric("vm.memory.size[used]", metrics.Memory.UsedBytes)
	addMetric("vm.memory.size[available]", metrics.Memory.AvailableBytes)
	addMetric("vm.memory.util", metrics.Memory.UsagePercent)

	// Disk метрики
	addMetric("vfs.fs.size[/,total]", metrics.Disk.TotalBytes)
	addMetric("vfs.fs.size[/,used]", metrics.Disk.UsedBytes)
	addMetric("vfs.fs.size[/,free]", metrics.Disk.FreeBytes)
	addMetric("vfs.fs.pused[/]", metrics.Disk.UsagePercent)

	// Network метрики
	addMetric("net.if.in[all]", metrics.Network.BytesRecv)
	addMetric("net.if.out[all]", metrics.Network.BytesSent)
	addMetric("net.if.in[all,packets]", metrics.Network.PacketsRecv)
	addMetric("net.if.out[all,packets]", metrics.Network.PacketsSent)
	addMetric("net.if.in[all,errors]", metrics.Network.ErrorsIn)
	addMetric("net.if.out[all,errors]", metrics.Network.ErrorsOut)

	return senderMetrics
}
