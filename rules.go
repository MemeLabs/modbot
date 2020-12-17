package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/memelabs/dggchat"
)

// Prevent repeated posting of short messages.
func (b *bot) noShortMsgSpam(m dggchat.Message, s *dggchat.Session) {
	// only proceed if the current message is "bad"
	if len(m.Message) > 2 {
		return
	}

	// ["time", "msg"]
	lastmsgs := b.getLastMessages(m.Sender.Nick, 10)
	badmsgs := []string{}
	badmsgcount := 0
	now := time.Now().Add(time.Duration(-60) * time.Minute)

	// check how many of the last messages were too short and they are within the
	// past hour.
	for _, msg := range lastmsgs {
		tm, err := time.Parse(time.Kitchen, msg[0])
		if err != nil {
			continue
		}

		if len(msg[1]) <= 2 && now.Before(tm) {
			badmsgcount++
			badmsgs = append(badmsgs, msg[1])
		}
	}

	if badmsgcount >= 5 {
		log.Printf("[##] single char mute with '%s' for '%s'\n", strings.Join(badmsgs, ", "), m.Sender.Nick)
		s.SendMute(m.Sender.Nick, -1)
		s.SendMessage(fmt.Sprintf("%s - too many short messages", m.Sender.Nick))
	}
}
