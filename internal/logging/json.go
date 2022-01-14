package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	golog "github.com/fclairamb/go-log"
)

type logMessage struct {
	Time  int64       `json:"time"`
	Level string      `json:"level"`
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type JSONLogger struct {
	verbosity int
}

func NewJSONLogger(verbosity int) *JSONLogger {
	return &JSONLogger{
		verbosity: verbosity,
	}
}

func printJSON(level string, event string, keyvals interface{}) {
	line, _ := json.Marshal(&logMessage{
		Time:  time.Now().Unix(),
		Level: level,
		Event: event,
		Data:  keyvals,
	})

	_, _ = fmt.Fprintln(os.Stderr, string(line))
}

func NewJSONLoggerWriter(verbosity int, event, key string) io.Writer {
	jsonLogger := NewJSONLogger(verbosity)

	reader, writer := io.Pipe()
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			jsonLogger.Trace(event, map[string]interface{}{
				key: scanner.Text(),
			})
		}
	}()

	return writer
}

func (l JSONLogger) Trace(event string, keyvals ...interface{}) {
	if l.verbosity >= 4 {
		printJSON("TRACE", event, keyvals)
	}
}

func (l JSONLogger) Debug(event string, keyvals ...interface{}) {
	if l.verbosity >= 3 {
		printJSON("DEBUG", event, keyvals)
	}
}

func (l JSONLogger) Info(event string, keyvals ...interface{}) {
	if l.verbosity >= 2 {
		printJSON("INFO", event, keyvals)
	}
}

func (l JSONLogger) Warn(event string, keyvals ...interface{}) {
	if l.verbosity >= 1 {
		printJSON("WARN", event, keyvals)
	}
}

func (l JSONLogger) Error(event string, keyvals ...interface{}) {
	if l.verbosity >= 0 {
		printJSON("ERROR", event, keyvals)
	}
}

func (l JSONLogger) With(keyvals ...interface{}) golog.Logger {
	return l
}
