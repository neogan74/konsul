package logger

import (
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"WARN", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"invalid", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
	}

	for _, tc := range testCases {
		result := ParseLevel(tc.input)
		if result != tc.expected {
			t.Errorf("ParseLevel(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}


func TestZapJSONLogging(t *testing.T) {
	// Test that we can create loggers and they don't panic
	jsonLogger := NewFromConfig("info", "json")
	textLogger := NewFromConfig("debug", "text")

	// These should not panic
	jsonLogger.Info("test message", String("key1", "value1"), Int("key2", 42))
	textLogger.Debug("debug message", String("component", "test"))
}

func TestZapLogLevels(t *testing.T) {
	// Test that log levels work correctly
	logger := NewFromConfig("warn", "json")

	// These should work without panic (we can't easily capture Zap's output in tests)
	logger.Debug("debug message") // Should be filtered out
	logger.Info("info message")   // Should be filtered out
	logger.Warn("warn message")   // Should be logged
	logger.Error("error message") // Should be logged
}


func TestWithRequest(t *testing.T) {
	logger := NewFromConfig("info", "json")
	requestLogger := logger.WithRequest("req-123")

	// Should not panic
	requestLogger.Info("test message")
}

func TestWithFields(t *testing.T) {
	logger := NewFromConfig("info", "json")
	loggerWithFields := logger.WithFields(String("component", "test"), Int("version", 1))

	// Should not panic
	loggerWithFields.Info("test message", String("extra", "data"))
}

func TestHelperFunctions(t *testing.T) {
	// Test that helper functions create valid zap fields (they return zap.Field now)
	stringField := String("key", "value")
	intField := Int("count", 42)
	durationField := Duration("elapsed", 5*time.Second)
	errorField := Error(&testError{"test error"})

	// Test they can be used with logger (should not panic)
	logger := NewFromConfig("info", "json")
	logger.Info("testing fields", stringField, intField, durationField, errorField)
}

func TestNewFromConfig(t *testing.T) {
	logger := NewFromConfig("debug", "json")

	// Test that it creates a valid logger (non-nil)
	if logger == nil {
		t.Fatal("NewFromConfig returned nil logger")
	}

	// Test logging with the created logger (should not panic)
	logger.Debug("debug message")
	logger.Info("info message")
}

func TestGlobalLogger(t *testing.T) {
	originalDefault := defaultLogger
	defer func() { SetDefault(originalDefault) }()

	testLogger := NewFromConfig("info", "json")
	SetDefault(testLogger)

	// Should not panic
	Info("global message")
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}