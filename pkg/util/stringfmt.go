package util

import "strings"

func NormalizeModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

func MetadataToName(ns, name string) string {
	return strings.Join([]string{ns, name}, "-")
}
