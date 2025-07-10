package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// Initialize инициализирует глобальный логгер
func Initialize(level string) error {
	var config zap.Config

	switch level {
	case "debug":
		config = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(parseLevel(level))
	default:
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Настраиваем формат времени
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	Logger, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// parseLevel конвертирует строку в zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Cleanup корректно закрывает логгер
func Cleanup() {
	if Logger != nil {
		Logger.Sync()
	}
}
