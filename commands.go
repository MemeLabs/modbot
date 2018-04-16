package main

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/voloshink/dggchat"
)

var (
	// TODO load from file.
	commands = map[string]string{
		"!faq": "https://github.com/MemeLabs/Rustla2/wiki/Chat-FAQ",
	}
	debuglogger = log.New(os.Stdout, "[d] ", log.Ldate|log.Ltime|log.Lshortfile)
)

func isMod(user dggchat.User) bool {
	return user.HasFeature("moderator") || user.HasFeature("admin")
}

func (b *bot) staticMessage(m dggchat.Message, s *dggchat.Session) {
	b.randomizer++
	rnd := " " + strings.Repeat(".", b.randomizer%2)

	for command, response := range commands {
		if strings.HasPrefix(m.Message, command) {
			s.SendMessage(response + rnd)
			// only handle the first match
			return
		}
	}
}

// !nuke str, !nukeregex regexp
func (b *bot) nuke(m dggchat.Message, s *dggchat.Session) {

	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!nuke") {
		return
	}

	parts := strings.SplitN(m.Message, " ", 2)
	if len(parts) <= 1 {
		return
	}

	isRegexNuke := parts[0] == "!nukeregex"
	badstr := parts[1]
	badregexp, err := regexp.Compile(badstr)
	if isRegexNuke && err != nil {
		b.randomizer++
		rnd := " " + strings.Repeat(".", b.randomizer%2)
		s.SendMessage("regexp error." + rnd)
		return
	}

	// find anyone saying badstr
	// TODO limit by time, not amout of messages...
	victimNames := []string{}
	// the command itself will be last in the log and caught, exclude that one.
	for _, m := range b.log[:len(b.log)-1] {

		var isBad bool
		if isRegexNuke {
			isBad = badregexp.MatchString(m.Message)
		} else {
			isBad = strings.Contains(m.Message, badstr)
		}

		if isBad {
			// TODO dont collect duplicates...
			// collect names in case we want to revert nuke
			victimNames = append(victimNames, m.Sender.Nick)

			debuglogger.Printf("Nuking '%s' because of message '%s' with nuke '%s'\n",
				m.Sender.Nick, m.Message, badstr)

			// TODO duration, -1 means server default
			s.SendMute(m.Sender.Nick, -1)
		}
		// TODO print/send summary?
	}

	if b.lastNukeVictims == nil {
		b.lastNukeVictims = []string{}
	}
	// combine array so we are able to undo all past nukes at once, if necessary
	for _, nick := range victimNames {
		b.lastNukeVictims = append(b.lastNukeVictims, nick)
	}
}

// !aegis - undo last nuke
func (b *bot) aegis(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!aegis") || b.lastNukeVictims == nil {
		return
	}

	for _, nick := range b.lastNukeVictims {
		s.SendUnmute(nick)
	}
	b.lastNukeVictims = nil
}
