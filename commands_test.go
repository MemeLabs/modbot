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
	var testInputs [9]string

	testInputs[0] = "!roll 2d2+100 foo biz baz"
	testInputs[1] = "!roll 2d2 + 100"
	testInputs[2] = "!roll 2d2 +100"
	testInputs[3] = "!roll 2d2+ 100"
	testInputs[4] = "!roll 2d2-100"
	testInputs[5] = "!roll 2d2 - 100"
	testInputs[6] = "!roll 2d2 -100"
	testInputs[7] = "!roll 2d2- 100"
	testInputs[8] = "!roll 2d2- 100 foo biz baz"
	testInputs[8] = "!roll 23904823904823904823490d20 +1"

	for _, input := range testInputs {
		result, err := Compute(input)
		errorMessage := fmt.Sprintf("%v", err)
		if err != nil {
			if errorMessage != "Sides or count too large" {
				t.Error(fmt.Sprintf("Error: %v\n %d", err, result))
			}
		}
	}

}
