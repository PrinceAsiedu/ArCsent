package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	format string
	base   *log.Logger
}

func New(format string) *Logger {
	if format == "" {
		format = "json"
	}
	return &Logger{
		format: format,
		base:   log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.write("info", msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.write("warn", msg, fields...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.write("error", msg, fields...)
}

func (l *Logger) write(level, msg string, fields ...Field) {
	if l.format == "text" {
		l.base.Printf("%s level=%s msg=%s %s", time.Now().Format(time.RFC3339), level, msg, formatFields(fields))
		return
	}

	payload := map[string]interface{}{
		"ts":    time.Now().Format(time.RFC3339),
		"level": level,
		"msg":   msg,
	}
	for _, f := range fields {
		payload[f.Key] = f.Value
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		l.base.Printf("%s level=error msg=%s err=%v", time.Now().Format(time.RFC3339), "failed to marshal log entry", err)
		return
	}
	l.base.Println(string(encoded))
}

type Field struct {
	Key   string
	Value interface{}
}

func formatFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}
	out := ""
	for i, f := range fields {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s=%v", f.Key, f.Value)
	}
	return out
}
