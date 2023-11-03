package main

import (
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

func TestCompute(t *testing.T) {
	testInputs := [10]string{
		"!roll 2d2+100 foo biz baz",
		"!roll 2d2 + 100",
		"!roll 2d2 +100",
		"!roll 2d2+ 100",
		"!roll 2d2-100",
		"!roll 2d2 - 100",
		"!roll 2d2 -100",
		"!roll 2d2- 100",
		"!roll 2d2- 100 foo biz baz",
		"!roll 23904823904823904823490d20 +1"}

	for _, input := range testInputs {
		result, err := compute(input)
		errorMessage := fmt.Sprintf("%v", err)
		if err != nil {
			if errorMessage != "Sides or count too large" {
				t.Error(fmt.Sprintf("Error: %v\n %d", err, result))
			}
		}
	}

}
