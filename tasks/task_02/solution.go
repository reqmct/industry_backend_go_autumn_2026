package main

import "slices"

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	size := len(runes)
	if size == 0 {
		return ""
	}

	shift = shift % size
	if shift < 0 {
		shift += size
	}

	return string(slices.Concat(runes[shift:], runes[:shift]))
}
