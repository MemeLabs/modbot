package main

import (
	"log"

	"github.com/voloshink/dggchat"
)

type bot struct {
	log             []dggchat.Message
	maxLogLines     int
	parsers         []func(m dggchat.Message, s *dggchat.Session)
	lastNukeVictims []string
	randomizer      int
}

func newBot(maxLogLines int) *bot {

	if maxLogLines < 0 {
		maxLogLines = 0
	}

	b := bot{
		log:         make([]dggchat.Message, maxLogLines),
		maxLogLines: maxLogLines,
		randomizer:  0, // TODO workaround for dup msgs, remove me...
	}
	return &b

}

func (b *bot) addParser(p func(m dggchat.Message, s *dggchat.Session)) {
	b.parsers = append(b.parsers, p)
}

func (b *bot) onMessage(m dggchat.Message, s *dggchat.Session) {

	// remember maxLogLines messages
	if len(b.log) >= b.maxLogLines {
		b.log = b.log[1:]
	}
	b.log = append(b.log, m)

	log.Printf("%s: %s\n", m.Sender.Nick, m.Message)

	for _, p := range b.parsers {
		p(m, s)
	}
}

func (b *bot) onError(e string, s *dggchat.Session) {
	log.Printf("error %s\n", e)
}

// return last n messsages for given user from log
func (b *bot) getLastMessages(nick string, n int) []string {

	output := []string{}
	for i := len(b.log) - 1; i >= 0; i-- {

		if len(output) >= n {
			return output
		}

		msg := b.log[i]
		if msg.Sender.Nick == nick {
			output = append(output, msg.Message)
		}
	}
	return output
}
