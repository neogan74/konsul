package logger

import (
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ParseLevel parses string to zapcore.Level
func ParseLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
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

// Field is an alias for zap.Field for interface compatibility
type Field = zap.Field

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithRequest(requestID string) Logger
	WithFields(fields ...Field) Logger
}

// zapLogger wraps zap.Logger to implement our Logger interface
type zapLogger struct {
	logger *zap.Logger
}

// New creates a new logger with zap
func New(level zapcore.Level, format string) Logger {
	var config zap.Config

	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	}

	config.Level = zap.NewAtomicLevelAt(level)

	logger, err := config.Build()
	if err != nil {
		// Fallback to default logger if build fails
		logger = zap.NewNop()
	}

	return &zapLogger{logger: logger}
}

// NewFromConfig creates a logger from string configuration
func NewFromConfig(level, format string) Logger {
	return New(ParseLevel(level), format)
}

// Debug logs a debug message
func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fields...)
}

// WithRequest returns a new logger with request ID field
func (l *zapLogger) WithRequest(requestID string) Logger {
	return l.WithFields(zap.String("request_id", requestID))
}

// WithFields returns a new logger with additional fields
func (l *zapLogger) WithFields(fields ...Field) Logger {
	return &zapLogger{logger: l.logger.With(fields...)}
}

// Helper functions for creating fields - now using Zap functions
func String(key, value string) Field {
	return zap.String(key, value)
}

func Int(key string, value int) Field {
	return zap.Int(key, value)
}

func Duration(key string, value time.Duration) Field {
	return zap.Duration(key, value)
}

func Error(err error) Field {
	return zap.Error(err)
}

// Default logger instance
var defaultLogger Logger = NewFromConfig("info", "text")

// SetDefault sets the default logger
func SetDefault(l Logger) {
	defaultLogger = l
}

// GetDefault returns the default logger instance
func GetDefault() Logger {
	return defaultLogger
}

// Global logging functions using default logger
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

func ErrorLog(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}
