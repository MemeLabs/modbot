package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/voloshink/dggchat"
)

// prevent single character spam
func (b *bot) antiSingleCharSpam(m dggchat.Message, s *dggchat.Session) {

	// only proceed if the current message is "bad"
	if len(m.Message) >= 2 {
		return
	}

	lastmsgs := b.getLastMessages(m.Sender.Nick, 10)
	badmsgs := 0

	// check how many of the last messages were too short
	for _, msg := range lastmsgs {
		if len(msg) <= 2 {
			badmsgs++
		}
	}

	if badmsgs >= 5 {
		log.Printf("[##] single char mute with '%s' for '%s'\n", strings.Join(lastmsgs, ", "), m.Sender.Nick)
		s.SendMute(m.Sender.Nick, -1)
		s.SendMessage(fmt.Sprintf("%s - too many short messages", m.Sender.Nick))
	}

}
