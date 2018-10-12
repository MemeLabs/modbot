package main

import (
	"fmt"
	"regexp"
)

var (
	// e.g. youtube ids include "-" and "_".
	commonMatch = "([\\w-]{1,30})"

	// identifier regexp for sanity check
	identifierRegexp = regexp.MustCompile(fmt.Sprintf("^%s/%s$", commonMatch, commonMatch))

	twitchVodRe = regexp.MustCompile(fmt.Sprintf("twitch\\.tv/videos/%s", commonMatch))
	// twitchVodRe collides with twitchRe so additional constraint for string to end...
	twitchRe   = regexp.MustCompile(fmt.Sprintf("twitch\\.tv/%s/?$", commonMatch))
	atRe       = regexp.MustCompile(fmt.Sprintf("angelthump\\.com/embed/%s", commonMatch))
	youtubeRe1 = regexp.MustCompile(fmt.Sprintf("youtube\\.com/watch.*?[&?]v=%s", commonMatch))
	youtubeRe2 = regexp.MustCompile(fmt.Sprintf("youtu\\.be/%s", commonMatch))
	youtubeRe3 = regexp.MustCompile(fmt.Sprintf("youtube\\.com/embed/%s", commonMatch))
	facebookRe = regexp.MustCompile(fmt.Sprintf("facebook\\.com/.*?/videos/%s/?", commonMatch))
	mixerRe    = regexp.MustCompile(fmt.Sprintf("mixer\\.com/(?:embed/player/)?%s$", commonMatch))

	// these are the path mappings on strims
	matchMap = map[*regexp.Regexp]string{
		twitchVodRe: "twitch-vod",
		twitchRe:    "twitch",
		atRe:        "angelthump",
		youtubeRe1:  "youtube",
		youtubeRe2:  "youtube",
		youtubeRe3:  "youtube",
		facebookRe:  "facebook",
		mixerRe:     "mixer",
	}
)

func isValidIdentifier(id string) bool {
	return identifierRegexp.MatchString(id)
}

func parseIdentifier(link string) string {

	for regexp, path := range matchMap {
		// FindStringSubmatch returns the full match and then whatever capture groups...
		// We are interested in the first and (normally) only match after the full (very first) one.
		// E.g. return value could be [twitch.tv/username, username] - we want the latter: "username".
		if match := regexp.FindStringSubmatch(link); len(match) == 2 {
			return path + "/" + match[1]
		}
	}

	return ""
}
