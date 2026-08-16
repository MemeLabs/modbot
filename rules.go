package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/MemeLabs/dggchat"
)

const (
	shortMsgLen    = 2
	shortMsgWindow = time.Hour
	shortMsgLimit  = 5
)

// Prevent repeated posting of short messages.
func (b *bot) noShortMsgSpam(_ context.Context, m message, s *dggchat.Session) {
	// only proceed if the current message is "bad"
	if len(m.Message) > shortMsgLen {
		return
	}

	lastmsgs := b.getLastMessages(m.Sender.Nick, 10)
	badmsgs := []string{}
	cutoff := time.Now().Add(-shortMsgWindow)

	// check how many of the last messages were too short and they are within the
	// past hour.
	for _, msg := range lastmsgs {
		if len(msg.Message) <= shortMsgLen && cutoff.Before(msg.Timestamp) {
			badmsgs = append(badmsgs, msg.Message)
		}
	}

	if len(badmsgs) >= shortMsgLimit {
		log.Printf("[##] single char mute with '%s' for '%s'\n", strings.Join(badmsgs, ", "), m.Sender.Nick)
		s.SendMute(m.Sender.Nick, -1)
		if err := s.SendMessage(m.Sender.Nick + " - too many short messages"); err != nil {
			log.Printf("[##] send error: %s\n", err.Error())
		}
	}
}
