package util

import (
	"fmt"
	"strings"
)

func ConcatReason(reason, message string) string {
	return strings.Trim(fmt.Sprintf("%s/%s", reason, message), "/")
}

func ConcatRealKey(nameSpace, resourceName string) string {
	return strings.Trim(fmt.Sprintf("%s/%s", nameSpace, resourceName), "/")
}

func ConcatResourceCacheKey(nameSpace, resourceName string) string {
	return ConcatRealKey(nameSpace, resourceName)
}

func ParseResourceCacheKey(resourceCacheKey string) string {
	return resourceCacheKey
}
