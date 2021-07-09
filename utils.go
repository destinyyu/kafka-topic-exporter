package main

import (
	"crypto/md5"
	"fmt"
	"strings"
)

func validMetricName(metric string) (string, bool) {
	switch {
	case strings.Contains(metric, "."):
		metric = strings.ReplaceAll(metric, ".", "_")
	default:
		return metric, false
	}

	return strings.ToLower(metric), true
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func UID(args ...string) string {
	s := strings.Join(args, "-")
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}
