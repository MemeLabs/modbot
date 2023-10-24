package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MemeLabs/dggchat"
)

var (
	mutex    sync.Mutex
	commands = map[string]string{}
)

func isMod(user dggchat.User) bool {
	return user.HasFeature("moderator") || user.HasFeature("admin")
}

// TODO
func (b *bot) sendMessageDedupe(m string, s *dggchat.Session) {
	if logOnly {
		log.Printf("[##] LOGONLY reply: %s\n", m)
		return
	}

	b.randomizer++
	rnd := " " + strings.Repeat(".", b.randomizer%2)
	err := s.SendMessage(m + rnd)
	if err != nil {
		log.Printf("[##] send error: %s\n", err.Error())
	}
}

func (b *bot) staticMessage(m dggchat.Message, s *dggchat.Session) {
	for command, response := range commands {
		if strings.HasPrefix(m.Message, command) {

			b.sendMessageDedupe(response, s)
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
	badregexp, err := regexp.Compile(badstr) // TODO when is error not nil??
	if isRegexNuke && err != nil {
		b.sendMessageDedupe("regexp error", s)
		return
	}

	// find anyone saying badstr
	// TODO limit by time, not amout of messages...
	victimNames := []string{}
	// the command itself will be last in the log and caught, exclude that one.
	// TODO: except if the command was issued via PM...
	for _, m := range b.log[:len(b.log)-1] {
		// don't nuke mods.
		if isMod(m.Sender) {
			continue
		}

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

			log.Printf("[##] Nuking '%s' because of message '%s' with nuke '%s'\n",
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
	b.lastNukeVictims = append(b.lastNukeVictims, victimNames...)
}

func (b *bot) sudoku(m dggchat.Message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!sudoku") {
		return
	}
	// TODO duration, -1 means server default
	s.SendMute(m.Sender.Nick, -1)
}

// !aegis - undo (all) past nukes
func (b *bot) aegis(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!aegis") || b.lastNukeVictims == nil {
		return
	}

	for _, nick := range b.lastNukeVictims {
		s.SendUnmute(nick)
	}
	b.lastNukeVictims = nil
}

// !rename - change a chatter's username
func (b *bot) rename(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!rename") {
		return
	}

	parts := strings.Split(m.Message, " ")
	if len(parts) < 3 {
		return
	}

	oldName := parts[1]
	newName := parts[2]
	err := b.renameUser(oldName, newName)
	if err != nil {
		msg := fmt.Sprintf("'%s' to '%s' by %s failed with '%s'",
			oldName, newName, m.Sender.Nick, err.Error())
		log.Printf("[##] rename: %s\n", msg)

		s.SendPrivateMessage(m.Sender.Nick, msg)
		b.sendMessageDedupe("rename error, check logs", s)
		return
	}
	log.Printf("[##] rename: '%s' to '%s' by '%s' success!\n",
		oldName, newName, m.Sender.Nick)
	b.sendMessageDedupe(fmt.Sprintf("name changed, %s please reconnect", oldName), s)
}

// !say - say a message
func (b *bot) say(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!say") {
		return
	}

	// message itself can contain spaces
	parts := strings.SplitN(m.Message, " ", 2)
	if len(parts) != 2 {
		return
	}
	b.sendMessageDedupe(parts[1], s)
}

// !mute - mute a chatter for a given time
func (b *bot) mute(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!mute") {
		return
	}
	parts := strings.Split(m.Message, " ")
	if len(parts) < 2 {
		return
	}

	var duration time.Duration = -1
	if len(parts) >= 3 {
		dur, err := time.ParseDuration(parts[2])
		if err != nil {
			log.Printf("failed to parse duration %q: %v. Using default time", parts[2], err)
		} else {
			duration = dur
		}
	}
	s.SendMute(parts[1], duration)
}

// !unmute - unmute a chatter
func (b *bot) unmute(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!unmute") {
		return
	}
	parts := strings.Split(m.Message, " ")
	if len(parts) < 2 {
		return
	}
	s.SendUnmute(parts[1])
}

// !addcommand command response
func (b *bot) addCommand(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!addcommand") {
		return
	}

	// message itself can contain spaces
	parts := strings.Split(m.Message, " ")
	if len(parts) < 3 {
		return
	}

	cmnd := parts[1]
	if !strings.HasPrefix(cmnd, "!") {
		cmnd = "!" + cmnd
	}
	resp := strings.Join(parts[2:], " ")
	mutex.Lock()
	defer mutex.Unlock()
	// TODO workaround to enable deletion
	if resp == "_" {
		delete(commands, cmnd)
		b.sendMessageDedupe("deleted commands if it existed", s)
	} else {
		commands[cmnd] = resp
		success := saveStaticCommands()
		if success {
			b.sendMessageDedupe(fmt.Sprintf("added new command %s", cmnd), s)
			return
		}
		b.sendMessageDedupe("failed saving command, check logs", s)
	}
}

// TOOD clean up...
func isCommunityStream(path string) bool {
	// "/twitch/test" it not. "/memer" is.
	return strings.Count(path, "/") == 1 || strings.Contains(path, "angelthump")
}

// !stream or !strim(s) -- show top streams in chat
func (b *bot) printTopStreams(m dggchat.Message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!stream") && !strings.HasPrefix(m.Message, "!strim") {
		return
	}

	sd, err := b.getStreamList()
	if err != nil {
		log.Printf("%v\n", err)
		b.sendMessageDedupe("error getting api data", s)
		return
	}

	// filter hidden streams
	allStreams := sd.StreamList
	filteredStreams := streamData{}
	for _, v := range allStreams {
		if !v.Hidden {
			filteredStreams.StreamList = append(filteredStreams.StreamList, v)
		}
	}

	// handle case that less than 3 streams are being watched...
	maxlen := len(filteredStreams.StreamList)
	if maxlen == 0 {
		b.sendMessageDedupe("no streams are being watched", s)
		return
	}
	if maxlen > 3 {
		maxlen = 3
	}

	alreadyPrinted := 0
	// - assumption: API gives json data sorted by "rustlers".
	// - first pass: give community streams preference
	// - data.URL has leading slash
	for i := 0; i < len(filteredStreams.StreamList) && alreadyPrinted < maxlen; i++ {
		data := filteredStreams.StreamList[i]
		if isCommunityStream(data.URL) {
			nsfw := ""
			if data.Nsfw {
				nsfw = " [nsfw]"
			}
			out := fmt.Sprintf("%d %s%s%s", data.Rustlers, websiteURL, data.URL, nsfw)
			b.sendMessageDedupe(out, s)
			alreadyPrinted++
		}
	}

	// TODO clean me up...
	for i := 0; alreadyPrinted < maxlen; i++ {
		data := filteredStreams.StreamList[i]
		if !isCommunityStream(data.URL) {
			nsfw := ""
			if data.Nsfw {
				nsfw = " [nsfw]"
			}
			data := filteredStreams.StreamList[i]
			out := fmt.Sprintf("%d %s%s%s", data.Rustlers, websiteURL, data.URL, nsfw)
			b.sendMessageDedupe(out, s)
			alreadyPrinted++
		}
	}
}

func parseModifiers(s []string) (streamModifier, error) {
	var sm streamModifier

	for _, part := range s {
		switch part {
		case "nsfw":
			sm.Nsfw = "true"
		case "!nsfw":
			sm.Nsfw = "false"
		case "hidden":
			sm.Hidden = "true"
		case "!hidden":
			sm.Hidden = "false"
		case "afk":
			sm.Afk = "true"
		case "!afk":
			sm.Afk = "false"
		case "promoted":
			sm.Promoted = "true"
		case "!promoted":
			sm.Promoted = "false"
		default:
			return streamModifier{}, fmt.Errorf("invalid modifier: '%s'", part)
		}
	}

	return sm, nil
}

func (b *bot) modifyStream(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!modify") {
		return
	}

	//                       parts[2:], ...
	// !modify youtube/memes nsfw !hidden ...
	parts := strings.Split(m.Message, " ")
	if len(parts) < 3 {
		return
	}

	sm, err := parseModifiers(parts[2:])
	if err != nil {
		b.sendMessageDedupe(fmt.Sprintf("%s %s", err.Error(), ominousEmote), s)
		return
	}

	identifier := parts[1]

	err = b.setStreamAttributes(identifier, sm)
	if err != nil {
		log.Printf("[##] modify: '%s' with modifier '%+v' by '%s' failed with '%s'\n",
			identifier, sm, m.Sender.Nick, err.Error())

		// TODO chat message less verbose
		b.sendMessageDedupe(fmt.Sprintf("modify: %s %s", err, ominousEmote), s)
		return
	}
	log.Printf("[##] modify: '%s' with modifier '%+v' by '%s' success!\n",
		identifier, sm, m.Sender.Nick)
	b.sendMessageDedupe(fmt.Sprintf("modify success %s", ominousEmote), s)
}

// !check ATusername
func (b *bot) checkAT(m dggchat.Message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!check") {
		return
	}

	parts := strings.Split(m.Message, " ")
	if len(parts) != 2 {
		return
	}
	username := parts[1]

	atd, err := b.getATUserData(username)
	if err != nil {
		log.Printf("[##] checkAT error1: '%s'\n",
			err.Error())

		// workaround... depends on content of error message
		if strings.Contains(err.Error(), "404") {
			log.Printf("[##] check: not found\n")
			return
		}

		b.sendMessageDedupe("error getting api data", s)
		return
	}

	// additionally check strim data
	sd, err := b.getStreamList()
	if err != nil {
		log.Printf("[##] checkAT error2: '%s'\n",
			err.Error())
		b.sendMessageDedupe("error getting api data", s)
		return
	}

	var url string
	viewerCount := 0
	for _, strim := range sd.StreamList {
		if strim.Service == "angelthump" && strings.EqualFold(strim.Channel, username) {
			viewerCount = strim.Rustlers
			url = fmt.Sprintf("%s%s", websiteURL, strim.URL)
			if strim.Hidden {
				log.Printf("[##] check: not found\n")
				return
			}
		}
	}

	// might be live on AT, but no rustlers: disregard.
	if viewerCount == 0 {
		log.Printf("[##] check: not found\n")
		return
	}

	output := fmt.Sprintf("%s is live for %s with %d rustlers and %d viewers at %s",
		atd.User.Username, humanizeDuration(time.Since(atd.CreatedAt)),
		viewerCount, atd.ViewerCount, url)

	if atd.User.Nsfw {
		output += " nsfw"
	}

	b.sendMessageDedupe(output, s)
}

// !(un)drop atUser
func (b *bot) dropAT(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || (!strings.HasPrefix(m.Message, "!drop") && !strings.HasPrefix(m.Message, "!undrop")) {
		return
	}

	parts := strings.SplitN(m.Message, " ", 3)
	if len(parts) < 2 {
		return
	}

	doBan := parts[0] == "!drop"
	username := parts[1]
	reason := ""

	if doBan && len(parts) < 3 {
		s.SendPrivateMessage(m.Sender.Nick,
			fmt.Sprintf("%s - please provide a ban reason", m.Sender.Nick))
		return
	}
	if doBan {
		reason = parts[2]
	}

	reply, err := b.banATuser(username, reason, doBan)
	if err != nil {
		log.Println(fmt.Sprintf("[##] drop error: '%s'", err.Error()))
		return
	}

	//	b.sendMessageDedupe(reply, s)
	s.SendPrivateMessage(m.Sender.Nick, reply)
}

// https://gist.github.com/harshavardhana/327e0577c4fed9211f65
func humanizeDuration(duration time.Duration) string {
	days := int64(duration.Hours() / 24)
	hours := int64(math.Mod(duration.Hours(), 24))
	minutes := int64(math.Mod(duration.Minutes(), 60))
	// seconds := int64(math.Mod(duration.Seconds(), 60))

	chunks := []struct {
		singularName string
		amount       int64
	}{
		{"day", days},
		{"hour", hours},
		{"min", minutes},
		//{"sec", seconds},
	}

	parts := []string{}

	for _, chunk := range chunks {
		switch chunk.amount {
		case 0:
			continue
		case 1:
			parts = append(parts, fmt.Sprintf("%d%s", chunk.amount, chunk.singularName))
		default:
			parts = append(parts, fmt.Sprintf("%d%ss", chunk.amount, chunk.singularName))
		}
	}

	return strings.Join(parts, " ")
}

// !(un)ban -- ban a user
func (b *bot) ban(m dggchat.Message, s *dggchat.Session) {
	if !isMod(m.Sender) || (!strings.HasPrefix(m.Message, "!ban") && !strings.HasPrefix(m.Message, "!unban")) {
		return
	}

	parts := strings.Split(m.Message, " ")
	if len(parts) < 2 {
		return
	}

	if parts[0] == "!ban" {
		reason := ""
		if len(parts) == 3 {
			reason = parts[2]
		}
		s.SendBan(parts[1], reason, 0, false)
	} else if parts[0] == "!unban" {
		s.SendUnban(parts[1])
	}
}

// !roll sides [count] - roll dice
func (b *bot) roll(m dggchat.Message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!roll") {
		return
	}

	parts := strings.Split(m.Message, " ")
	if len(parts) < 2 {
		return
	}

	args := parts[1:]

	// parse XdY
	if strings.Contains(parts[1], "d") {
		parts := strings.Split(parts[1], "d")
		args = []string{parts[1], parts[0]}
	}

	sides, _ := strconv.ParseUint(args[0], 10, 64)
	if sides < 2 {
		return
	}

	count := uint64(1)
	if len(args) > 1 {
		c, _ := strconv.ParseUint(args[1], 10, 64)
		if c != 0 {
			count = c
		}
	}

	if math.MaxInt64/count <= sides {
		return
	}

	res := rand.Int63n(int64((sides-1)*count)) + int64(count)
	b.sendMessageDedupe(fmt.Sprintf("%s rolled %d", m.Sender.Nick, res), s)
}
