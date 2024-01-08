package main

import (
	"errors"
	"fmt"
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

func TestComputeRoll(t *testing.T) {
	testInputs := []struct {
		input string
		err   error
	}{
		{input: "!roll 2d2+100 foo biz baz"},
		{input: "!roll 2d2 + 100"},
		{input: "!roll 2d2 +100"},
		{input: "!roll 2d2+ 100"},
		{input: "!roll 2d2-100"},
		{input: "!roll 2d2 - 100"},
		{input: "!roll 2d2 -100"},
		{input: "!roll 2d2- 100"},
		{input: "!roll 2d2- 100 foo biz baz"},
		{input: "!roll 2d20"},
		{input: "!roll 20"},
		{input: "!roll 20+10"},
		{input: "!roll 1d9223372036854775807"},
		{input: "!roll 1d9223372036854775807+1", err: errResultRangeBounds},
		{input: "!roll 9223372036854775807d1", err: errInputBounds},
	}

	for _, c := range testInputs {
		_, err := computeRoll(c.input)
		if !errors.Is(err, c.err) {
			fmt.Printf(`"%s": unexpected error %s\n`, c.input, err)
			t.FailNow()
		}
	}
}
