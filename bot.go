package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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
	log.Printf("error %s\n", e)
}

func (b *bot) onMute(m dggchat.Mute, s *dggchat.Session) {
	log.Printf("mute: '%s' by '%s'\n", m.Sender.Nick, m.Target.Nick)
}

func (b *bot) onUnmute(m dggchat.Mute, s *dggchat.Session) {
	log.Printf("unmute: '%s' by '%s'\n", m.Sender.Nick, m.Target.Nick)
}

func (b *bot) onBan(m dggchat.Ban, s *dggchat.Session) {
	log.Printf("ban: '%s' by '%s'\n", m.Sender.Nick, m.Target.Nick)
}

func (b *bot) onUnban(m dggchat.Ban, s *dggchat.Session) {
	log.Printf("unban: '%s' by '%s'\n", m.Sender.Nick, m.Target.Nick)
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

// interact with backend...
func (b *bot) renameRequest(oldName string, newName string) error {

	var jsonStr = []byte(fmt.Sprintf(`{"username":"%s"}`, newName))
	path := fmt.Sprintf("%s/admin/profiles/%s/username", backendURL, oldName)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonStr))
	adminCookie := fmt.Sprintf("jwt=%s", b.authCookie)
	req.Header.Set("Cookie", adminCookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot", "botnet")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Status code %d, %s", resp.StatusCode, body)
	}
	return nil
}
