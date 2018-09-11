package main

import (
	"fmt"
	"testing"
)

func TestParseIdentifier(t *testing.T) {

	linkIdentifierMap := map[string]string{
		"https://www.youtube.com/watch?v=9wnNW4HyDtg&t=13s":  "youtube/9wnNW4HyDtg",
		"https://www.youtube.com/embed/9wnNW4HyDtg":          "youtube/9wnNW4HyDtg",
		"youtu.be/9wnNW4HyDtg?t=13s":                         "youtube/9wnNW4HyDtg",
		"youtube.com/watch?t=13s&v=9wnNW4HyDtg":              "youtube/9wnNW4HyDtg",
		"twitch.tv/admin":                                    "twitch/admin",
		"twitch.tv/admin/":                                   "twitch/admin",
		"twitch.tv/videos/":                                  "twitch/videos",
		"twitch.tv/videos/a":                                 "twitch-vod/a",
		"angelthump.com/embed/test":                          "angelthump/test",
		"https://www.twitch.tv/videos/777777777?t=01h11m11s": "twitch-vod/777777777",
		"bogus www.twitch.tv/videos/777777777 bogus":         "twitch-vod/777777777",
		"bogus.com/watch?v=aaaaaaaaaaa":                      "",
	}

	for link, expected := range linkIdentifierMap {
		result := parseIdentifier(link)
		if result != expected {
			t.Errorf("Failed with '%s': got: '%s', expected: '%s'\n", link, result, expected)
		}
		if result != "" && !isValidIdentifier(result) {
			t.Errorf("Non-empty result failed sanity check: '%s'\n", result)
		}
	}

}

func TestIsValidIdentifier(t *testing.T) {

	PassCases := []string{
		"twitch/admin",
		"middle/matches",
		"a/b",
	}
	FailCases := []string{
		"twitch/",
		"/test",
		"youtube.com/test",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/b", //too long
		"test middle/matches test",
		"https://www.youtube.com/watch?v=9wnNW4HyDtg&t=13s",
	}

	for _, v := range PassCases {
		if !isValidIdentifier(v) {
			t.Errorf("Supposedly valid identifier failed: '%s'\n", v)
		}
	}

	for _, v := range FailCases {
		if isValidIdentifier(v) {
			t.Errorf("Supposedly invalid identifier passed: '%s'\n", v)
		}
	}
}

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
	fmt.Println(err.Error())

}
