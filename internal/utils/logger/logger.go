package logger

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"
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

var levelLabel = map[Level]string{
    DEBUG: "DEBUG",
    INFO:  "INFO",
    WARN:  "WARN",
    ERROR: "ERROR",
    FATAL: "FATAL",
}

var levelColor = map[Level]string{
    DEBUG: "\033[36m", // cyan
    INFO:  "\033[32m", // green
    WARN:  "\033[33m", // yellow
    ERROR: "\033[31m", // red
    FATAL: "\033[35m", // magenta
}

const resetColor = "\033[0m"
const dimColor = "\033[90m"

func NewLogger(level string, format string) *Logger {
    return &Logger{
        level:  parseLevel(level),
        format: format,
    }
}

func (l *Logger) Debug(msg string, args ...interface{}) {
    if l.level <= DEBUG {
        l.log(DEBUG, msg, args...)
    }
}

func (l *Logger) Info(msg string, args ...interface{}) {
    if l.level <= INFO {
        l.log(INFO, msg, args...)
    }
}

func (l *Logger) Warn(msg string, args ...interface{}) {
    if l.level <= WARN {
        l.log(WARN, msg, args...)
    }
}

func (l *Logger) Error(msg string, args ...interface{}) {
    if l.level <= ERROR {
        l.log(ERROR, msg, args...)
    }
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
    l.log(FATAL, msg, args...)
    os.Exit(1)
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
    formatted := msg
    if len(args) > 0 {
        formatted = fmt.Sprintf(msg, args...)
    }
    if l.format == "json" {
        sanitized := strings.ReplaceAll(formatted, `"`, `\"`)
        log.Printf(`{"level":"%s","timestamp":"%s","message":"%s"}`, levelLabel[level], time.Now().Format(time.RFC3339), sanitized)
    } else {
        timestamp := time.Now().Format("01/02/2006, 15:04:05")
        pid := os.Getpid()
        color := levelColor[level]
        label := levelLabel[level]
        log.Printf("%s[Nest] %-6d  - %s    %s%s %s%s %s%s", dimColor, pid, timestamp, color, label, resetColor, dimColor, formatted, resetColor)
    }
}

func (l *Logger) Format() string {
	return l.format
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
