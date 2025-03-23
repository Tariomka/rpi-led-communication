package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97

	timeFormat = "[15:04:05.000]"
)

func colorize(colorCode int, value string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), value, reset)
}

type LogHandler struct {
	handler slog.Handler
	buffer  *bytes.Buffer
	mutex   *sync.Mutex
	print   func(message string)
}

func NewLogHandler(
	printCallback func(message string),
	opts *slog.HandlerOptions,
) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if printCallback == nil {
		printCallback = func(message string) { fmt.Println(message) }
	}
	buffer := &bytes.Buffer{}
	return &LogHandler{
		buffer: buffer,
		handler: slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		mutex: &sync.Mutex{},
		print: printCallback,
	}
}

func (lh *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	level := record.Level.String() + ":"

	switch record.Level {
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	attrs, err := lh.computeAttrs(ctx, record)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("error when marshaling attrs: %w", err)
	}

	payload := strings.Join([]string{
		colorize(lightGray, record.Time.Format(timeFormat)),
		level,
		colorize(white, record.Message),
		colorize(darkGray, string(bytes)),
	}, " ")

	lh.print(payload)
	return nil
}

func (lh *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return lh.handler.Enabled(ctx, level)
}

func (lh *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{
		handler: lh.handler.WithAttrs(attrs),
		buffer:  lh.buffer,
		mutex:   lh.mutex,
		print:   lh.print,
	}
}

func (lh *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{
		handler: lh.handler.WithGroup(name),
		buffer:  lh.buffer,
		mutex:   lh.mutex,
		print:   lh.print,
	}
}

func (lh *LogHandler) computeAttrs(ctx context.Context, record slog.Record) (map[string]any, error) {
	lh.mutex.Lock()
	defer func() {
		lh.buffer.Reset()
		lh.mutex.Unlock()
	}()
	if err := lh.handler.Handle(ctx, record); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(lh.buffer.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func suppressDefaults(next func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey || attr.Key == slog.LevelKey || attr.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return attr
		}
		return next(groups, attr)
	}
}
