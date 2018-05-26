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
	authCookie      string
}

func newBot(authCookie string, maxLogLines int) *bot {

	if maxLogLines < 0 {
		maxLogLines = 0
	}

	b := bot{
		log:         make([]dggchat.Message, maxLogLines),
		maxLogLines: maxLogLines,
		randomizer:  0, // TODO workaround for dup msgs, remove me...
		authCookie:  authCookie,
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
	log.Printf("[#] error: '%s'\n", e)
}

func (b *bot) onMute(m dggchat.Mute, s *dggchat.Session) {
	log.Printf("[#] mute: '%s' by '%s'\n", m.Target.Nick, m.Sender.Nick)
}

func (b *bot) onUnmute(m dggchat.Mute, s *dggchat.Session) {
	log.Printf("[#] unmute: '%s' by '%s'\n", m.Target.Nick, m.Sender.Nick)
}

func (b *bot) onBan(m dggchat.Ban, s *dggchat.Session) {
	log.Printf("[#] ban: '%s' by '%s'\n", m.Target.Nick, m.Sender.Nick)
}

func (b *bot) onUnban(m dggchat.Ban, s *dggchat.Session) {
	log.Printf("[#] unban: '%s' by '%s'\n", m.Target.Nick, m.Sender.Nick)
}

func (b *bot) onSocketError(err error, s *dggchat.Session) {
	log.Printf("[#] socket error: '%s'\n", err.Error())
}

func (b *bot) onPMHandler(m dggchat.PrivateMessage, s *dggchat.Session) {
	log.Printf("[#] PM: %s: %s\n", m.User.Nick, m.Message)

	if isMod(m.User) {
		// handle PM as command, TODO: rules shouldn't be handled here...
		msg := dggchat.Message{
			Sender:    m.User,
			Timestamp: m.Timestamp,
			Message:   m.Message,
		}

		for _, p := range b.parsers {
			p(msg, s)
		}
	}
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
