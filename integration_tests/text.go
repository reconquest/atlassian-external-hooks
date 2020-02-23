package main

import "strings"

func text(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}
