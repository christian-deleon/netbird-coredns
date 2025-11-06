package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel Level = LevelInfo
	logger       *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

// SetLevel sets the global log level
func SetLevel(level string) error {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn", "warning":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	default:
		return fmt.Errorf("invalid log level: %s. Must be one of: debug, info, warn, error", level)
	}
	return nil
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if currentLevel <= LevelDebug {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if currentLevel <= LevelInfo {
		logger.Printf("[INFO] "+format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if currentLevel <= LevelWarn {
		logger.Printf("[WARN] "+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if currentLevel <= LevelError {
		logger.Printf("[ERROR] "+format, v...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	logger.Fatalf("[FATAL] "+format, v...)
}

// Printf logs a message at info level (for backward compatibility)
func Printf(format string, v ...interface{}) {
	Info(format, v...)
}

// Println logs a message at info level (for backward compatibility)
func Println(v ...interface{}) {
	if currentLevel <= LevelInfo {
		logger.Println(v...)
	}
}

// Print logs a message at info level (for backward compatibility)
func Print(v ...interface{}) {
	if currentLevel <= LevelInfo {
		logger.Print(v...)
	}
}
