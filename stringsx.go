package main

import "strings"

// Small wrappers (aid readability + tests)
func stringsTrimSpace(value string) string           { return strings.TrimSpace(value) }
func stringsSplit(value string, sep string) []string { return strings.Split(value, sep) }
func stringsEqualFold(a, b string) bool              { return strings.EqualFold(a, b) }
