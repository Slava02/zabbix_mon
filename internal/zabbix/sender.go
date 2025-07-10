package zabbix

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

const (
	// Zabbix Sender протокол
	senderHeader  = "ZBXD\x01"
	senderDataLen = 8
)

// SenderData представляет данные для отправки через Zabbix Sender
type SenderData struct {
	Host  string      `json:"host"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Clock int64       `json:"clock,omitempty"`
}

// SenderRequest представляет запрос Zabbix Sender
type SenderRequest struct {
	Request string       `json:"request"`
	Data    []SenderData `json:"data"`
	Clock   int64        `json:"clock,omitempty"`
}

// SenderResponse представляет ответ Zabbix Sender
type SenderResponse struct {
	Response string `json:"response"`
	Info     string `json:"info,omitempty"`
}

// Sender реализует Zabbix Sender протокол
type Sender struct {
	serverHost string
	serverPort int
	timeout    time.Duration
	logger     *zap.Logger
}

// NewSender создает новый Zabbix Sender
func NewSender(serverHost string, serverPort int, timeout time.Duration, logger *zap.Logger) *Sender {
	return &Sender{
		serverHost: serverHost,
		serverPort: serverPort,
		timeout:    timeout,
		logger:     logger,
	}
}

// SendData отправляет данные через Zabbix Sender протокол
func (s *Sender) SendData(ctx context.Context, data []SenderData) error {
	if len(data) == 0 {
		return nil
	}

	s.logger.Debug("Sending data via Zabbix Sender",
		zap.String("host", s.serverHost),
		zap.Int("port", s.serverPort),
		zap.Int("items", len(data)))

	// Создаем запрос
	request := SenderRequest{
		Request: "sender data",
		Data:    data,
		Clock:   time.Now().Unix(),
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal sender request: %w", err)
	}

	// Формируем пакет согласно протоколу Zabbix Sender
	packet := s.buildPacket(jsonData)

	// Отправляем данные
	response, err := s.sendPacket(ctx, packet)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	// Парсим ответ
	var senderResp SenderResponse
	if err := json.Unmarshal(response, &senderResp); err != nil {
		return fmt.Errorf("failed to parse sender response: %w", err)
	}

	if senderResp.Response != "success" {
		return fmt.Errorf("zabbix sender error: %s", senderResp.Info)
	}

	s.logger.Debug("Successfully sent data via Zabbix Sender",
		zap.String("info", senderResp.Info))

	return nil
}

// buildPacket создает пакет согласно протоколу Zabbix Sender
func (s *Sender) buildPacket(data []byte) []byte {
	dataLen := uint64(len(data))

	// Заголовок протокола + длина данных + данные
	packet := make([]byte, 0, len(senderHeader)+senderDataLen+len(data))

	// Добавляем заголовок
	packet = append(packet, []byte(senderHeader)...)

	// Добавляем длину данных (little-endian uint64)
	lenBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBytes, dataLen)
	packet = append(packet, lenBytes...)

	// Добавляем данные
	packet = append(packet, data...)

	return packet
}

// sendPacket отправляет пакет на Zabbix сервер и возвращает ответ
func (s *Sender) sendPacket(ctx context.Context, packet []byte) ([]byte, error) {
	// Подключаемся к серверу
	dialer := &net.Dialer{
		Timeout: s.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", s.serverHost, s.serverPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zabbix server: %w", err)
	}
	defer conn.Close()

	// Устанавливаем таймаут для операций
	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	// Отправляем пакет
	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	// Читаем ответ
	response, err := s.readResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

// readResponse читает ответ от Zabbix сервера
func (s *Sender) readResponse(conn net.Conn) ([]byte, error) {
	// Читаем заголовок (5 байт)
	header := make([]byte, 5)
	if _, err := conn.Read(header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if !bytes.Equal(header, []byte(senderHeader)) {
		return nil, fmt.Errorf("invalid response header: %v", header)
	}

	// Читаем длину данных (8 байт)
	lenBytes := make([]byte, 8)
	if _, err := conn.Read(lenBytes); err != nil {
		return nil, fmt.Errorf("failed to read data length: %w", err)
	}

	dataLen := binary.LittleEndian.Uint64(lenBytes)
	if dataLen > 1024*1024 { // 1MB максимум
		return nil, fmt.Errorf("response data too large: %d bytes", dataLen)
	}

	// Читаем данные
	data := make([]byte, dataLen)
	if _, err := conn.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	return data, nil
}

// ConvertMetricsToSenderData конвертирует метрики в формат Zabbix Sender
func ConvertMetricsToSenderData(hostName string, items map[string]string, metrics interface{}) []SenderData {
	// Эта функция будет реализована в client.go
	// Здесь просто заглушка
	return nil
}
