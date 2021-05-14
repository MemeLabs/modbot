package main

import (
	"testing"
)

func TestParseModifiers(t *testing.T) {
	// correct modifiers
	s1 := []string{"nsfw", "hidden"}
	result, err := parseModifiers(s1)
	if err != nil || result.Hidden != "true" || result.Nsfw != "true" {
		t.Error("Error in modifier s1")
	}

	// wrong modifier
	s2 := []string{"nsfw", "hide"}
	result, err = parseModifiers(s2)
	if err == nil {
		t.Error("Error in modifier s2")
	}
	// fmt.Println(err.Error())
}
