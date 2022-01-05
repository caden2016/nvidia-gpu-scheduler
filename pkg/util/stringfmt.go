package util

import "strings"

func NormalizeModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}
