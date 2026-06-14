package logger

import (
    "fmt"
    "log"
    "os"
    "strings"
)

type Logger struct {
    level  Level
    format string
}

type Level int

const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
    FATAL
)

func NewLogger(level string, format string) *Logger {
    return &Logger{
        level:  parseLevel(level),
        format: format,
    }
}

func (l *Logger) Debug(msg string, args ...interface{}) {
    if l.level <= DEBUG {
        l.log("DEBUG", msg, args...)
    }
}

func (l *Logger) Info(msg string, args ...interface{}) {
    if l.level <= INFO {
        l.log("INFO", msg, args...)
    }
}

func (l *Logger) Warn(msg string, args ...interface{}) {
    if l.level <= WARN {
        l.log("WARN", msg, args...)
    }
}

func (l *Logger) Error(msg string, args ...interface{}) {
    if l.level <= ERROR {
        l.log("ERROR", msg, args...)
    }
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
    l.log("FATAL", msg, args...)
    os.Exit(1)
}

func (l *Logger) log(level string, msg string, args ...interface{}) {
    formatted := msg
    if len(args) > 0 {
        formatted = fmt.Sprintf(msg, args...)
    }
    if l.format == "json" {
        log.Printf(`{"level":"%s","message":"%s"}`, level, formatted)
    } else {
        log.Printf("[%s] %s", level, formatted)
    }
}

func parseLevel(level string) Level {
    switch strings.ToLower(level) {
    case "debug":
        return DEBUG
    case "info":
        return INFO
    case "warn":
        return WARN
    case "error":
        return ERROR
    case "fatal":
        return FATAL
    default:
        return INFO
    }
}
