package util

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	SetLoggerLevel(slog.LevelDebug)
}

func SetLogger(l *slog.Logger) {
	logger = l
}

func SetLoggerLevel(level slog.Level) {
	programLevel := new(slog.LevelVar)
	programLevel.Set(level) // 默认debug级别
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})
	SetLogger(slog.New(h))
}

func Debugw(msg string, kv ...interface{}) {
	if logger == nil {
		return
	}
	logger.Debug(msg, kv...)
}

func Infow(msg string, kv ...interface{}) {
	if logger == nil {
		return
	}
	logger.Info(msg, kv...)
}

func Warnw(msg string, kv ...interface{}) {
	if logger == nil {
		return
	}
	logger.Warn(msg, kv...)
}

func Errorw(msg string, kv ...interface{}) {
	if logger == nil {
		return
	}
	logger.Error(msg, kv...)
}

func Panicw(msg string, kv ...interface{}) {
	if logger == nil {
		panic(msg)
	}
	logger.Error(msg, kv...)
	panic(msg)
}
