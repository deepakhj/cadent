package main

import (
	"testing"
)

func TestDecode(t *testing.T) {
	_, err := parseConfigFile("../../config.toml")
	if err != nil {
		t.Fatal(err)
	}
}
