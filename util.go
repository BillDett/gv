package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

/*

Utility functions that don't belong anyplace special

*/
func generateFilename(prefix string, suffix string) string {
	filename := strings.ToLower(prefix)
	filename = strings.Replace(filename, " ", "_", -1)
	maxLen := 10 // Cap prefix at this length
	if maxLen > len(prefix) {
		maxLen = len(prefix)
	}
	filename = filename[0:maxLen]
	filename = fmt.Sprintf("%s%s%s", filename, randSeq(5), suffix)
	return filename
}

func randSeq(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().Unix())
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
