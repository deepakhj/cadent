package main

import (
	"testing"
)

func TestDecode(t *testing.T) {
	_, err := parseConfigFile("testfiles/config.toml")
	if err != nil {
		t.Fatal(err)
	}
}
