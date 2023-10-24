package kubewatcher

import (
	"kubewatcher/util"
	"log/slog"
)

// 设置包内日志
// nil 则不输出日志
func SetLog(logger *slog.Logger) {
	util.SetLogger(logger)
}

func SetLogLevel(level slog.Level) {
	util.SetLoggerLevel(level)
}
