package main

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
)

func TestUtil(t *testing.T) {

	fmt.Println("Generate simple filename")
	f := generateFilename("", "")
	fmt.Printf("Simple filename: %s\n", f)

	fmt.Println("Generate simple filename with prefix")
	f = generateFilename("foo", "")
	fmt.Printf("Prefix filename: %s\n", f)
	if !strings.HasPrefix(f, "foo") {
		t.Errorf("Fail: Generated filename >%s< missing prefix 'foo'\n", f)
	}

	fmt.Println("Generate simple filename with suffix")
	f = generateFilename("", "bar")
	fmt.Printf("Sufffix filename: %s\n", f)
	if !strings.HasSuffix(f, "bar") {
		t.Errorf("Fail: Generated filename >%s< missing suffix 'bar'\n", f)
	}

	fmt.Println("Generate filename with prefix containing invalid chars")
	f = generateFilename("*foo,$(&", "")
	fmt.Printf("Prefix filename: %s\n", f)
	if !strings.HasPrefix(f, "foo") {
		t.Errorf("Fail: Generated filename >%s< missing sanitized prefix 'foo'\n", f)
	}

}
