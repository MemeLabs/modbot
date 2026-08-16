package main

import (
	"context"
	"log"
	"slices"
	"time"

	"github.com/MemeLabs/dggchat"
)

// message mirrors dggchat.Message and records how it reached the bot. Parsers
// use isPM to reply on the channel the command arrived on, so that a command
// issued over PM doesn't leak its reply into public chat. It isn't an embedded
// dggchat.Message because the outer Message field would shadow the inner one.
type message struct {
	Sender    dggchat.User
	Timestamp time.Time
	Message   string
	isPM      bool
}

// parser inspects an incoming message and optionally acts on it. The context is
// scoped to handling that single message, so outbound API calls are cancelled
// when the bot shuts down.
type parser func(ctx context.Context, m message, s *dggchat.Session)

type bot struct {
	// Normally a context would be passed as an argument, but dggchat's handler
	// signatures are fixed, so the shutdown context is carried on the bot and
	// handed to parsers from there.
	ctx             context.Context
	log             []dggchat.Message
	maxLogLines     int
	parsers         []parser
	lastNukeVictims []string
	randomizer      int
	authCookie      string
}

func newBot(ctx context.Context, authCookie string, maxLogLines int) *bot {
	if maxLogLines < 0 {
		maxLogLines = 0
	}

	return &bot{
		ctx:         ctx,
		log:         make([]dggchat.Message, 0, maxLogLines),
		maxLogLines: maxLogLines,
		randomizer:  0, // TODO workaround for dup msgs, remove me...
		authCookie:  authCookie,
	}
}

func (b *bot) addParser(p ...parser) {
	b.parsers = append(b.parsers, p...)
}

// dispatch runs every parser against the message.
func (b *bot) dispatch(m message, s *dggchat.Session) {
	for _, p := range b.parsers {
		p(b.ctx, m, s)
	}
}

func (b *bot) onMessage(m dggchat.Message, s *dggchat.Session) {
	// remember maxLogLines messages
	if len(b.log) >= b.maxLogLines {
		b.log = b.log[1:]
	}
	b.log = append(b.log, m)

	log.Printf("%s: %s\n", m.Sender.Nick, m.Message)

	setConnected(true)
	recordMessage()

	b.dispatch(message{Sender: m.Sender, Timestamp: m.Timestamp, Message: m.Message}, s)
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
	setConnected(false)
	// dggchat starts its own reconnect loop right after this handler, and
	// exposes no callback for a successful one -- setConnected(true) comes from
	// the pinger and inbound messages instead.
	wsReconnects.Inc()
}

func (b *bot) onPMHandler(m dggchat.PrivateMessage, s *dggchat.Session) {
	log.Printf("[#] PM: %s: %s\n", m.User.Nick, m.Message)
	setConnected(true)

	if !isMod(m.User) {
		return
	}

	// handle PM as command, TODO: rules shouldn't be handled here...
	b.dispatch(message{
		Sender:    m.User,
		Timestamp: m.Timestamp,
		Message:   m.Message,
		isPM:      true,
	}, s)
}

// return last n messsages for given user from log
func (b *bot) getLastMessages(nick string, n int) []dggchat.Message {
	var output []dggchat.Message
	for _, msg := range slices.Backward(b.log) {
		if len(output) >= n {
			return output
		}

		if msg.Sender.Nick == nick {
			output = append(output, msg)
		}
	}
	return output
}
