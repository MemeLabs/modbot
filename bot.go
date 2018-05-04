package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

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

type userInfo struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
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

func (b *bot) onSocketError(err error, s *dggchat.Session) {
	log.Printf("socket error: '%s'\n", err.Error())
}

func (b *bot) onPMHandler(m dggchat.PrivateMessage, s *dggchat.Session) {
	log.Printf("PM from '%s': '%s'\n", m.User.Nick, m.Message)
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

// get basic user info - to check if we are logged in and have correct rights
func (b *bot) getProfileInfo() (userInfo, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/profile", backendURL), nil)
	c := fmt.Sprintf("jwt=%s", b.authCookie)
	req.Header.Set("X-Bot", "botnet")
	req.Header.Set("Cookie", c)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return userInfo{}, err
	}

	var ui userInfo
	err = json.NewDecoder(resp.Body).Decode(&ui)
	if err != nil {
		return userInfo{}, err
	}

	return ui, nil
}

// interact with backend...
func (b *bot) renameRequest(oldName string, newName string) error {

	var jsonStr = []byte(fmt.Sprintf(`{"username":"%s"}`, newName))
	path := fmt.Sprintf("%s/admin/profiles/%s/username", backendURL, oldName)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonStr))
	c := fmt.Sprintf("jwt=%s", b.authCookie)
	req.Header.Set("Cookie", c)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot", "botnet")

	client := &http.Client{Timeout: 2 * time.Second}
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
