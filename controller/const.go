package controller

import "strings"

type WatcherKeyPrefix string // 用于标记该次k8s推送事件的操作类型

const (
	WatcherKeyPrefixUpdate WatcherKeyPrefix = "Update" // add 也是一种 update
	WatcherKeyPrefixDelete WatcherKeyPrefix = "Delete"
)

func BuildWatcherKeyFunc(method WatcherKeyPrefix, key string) string {
	return string(method) + "/" + key
}

func SplitWatcherKeyFunc(methodKey string) (WatcherKeyPrefix, string) {
	i := strings.Index(methodKey, "/")
	if i > 0 {
		return WatcherKeyPrefix(methodKey[0:i]), methodKey[i+1:]
	}
	return "", ""
}

func (r WatcherKeyPrefix) IsDelete() bool {
	return r == WatcherKeyPrefixDelete
}

func (r WatcherKeyPrefix) IsUpdate() bool {
	return r == WatcherKeyPrefixUpdate
}
