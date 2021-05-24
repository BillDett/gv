package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

/*

Utility functions that don't belong anyplace special

*/

var illegalName = regexp.MustCompile(`[^[:alnum:]-.]`)

func generateFilename(prefix string, suffix string) string {
	filename := strings.ToLower(prefix)
	filename = illegalName.ReplaceAllString(filename, "") // 'clean' the filename of any invalid chars
	maxLen := 10                                          // Cap prefix at this length
	if maxLen > len(filename) {
		maxLen = len(filename)
	}
	filename = filename[0:maxLen]
	filename = fmt.Sprintf("%s%s%s", filename, randSeq(5), suffix)
	return filename
}

func randSeq(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
