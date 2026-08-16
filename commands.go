package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MemeLabs/dggchat"
)

func isMod(user dggchat.User) bool {
	return user.HasFeature("moderator") || user.HasFeature("admin")
}

// sendMessageDedupe replies on the channel src arrived on: privately to the
// sender for a command issued over PM, in public chat otherwise.
func (b *bot) sendMessageDedupe(src message, reply string, s *dggchat.Session) {
	if !src.isPM {
		b.sendPublicDedupe(reply, s)
		return
	}

	if logOnly {
		log.Printf("[##] LOGONLY PM reply to %s: %s\n", src.Sender.Nick, reply)
		return
	}

	// no dedupe suffix needed, the duplicate filter only applies to public chat
	if err := s.SendPrivateMessage(src.Sender.Nick, reply); err != nil {
		log.Printf("[##] send error: %s\n", err.Error())
	}
}

// sendPublicDedupe always replies in public chat, even for commands issued over
// PM. Use it for commands whose whole purpose is to say something publicly.
// TODO
func (b *bot) sendPublicDedupe(reply string, s *dggchat.Session) {
	if logOnly {
		log.Printf("[##] LOGONLY reply: %s\n", reply)
		return
	}

	b.randomizer++
	rnd := " " + strings.Repeat(".", b.randomizer%2)
	if err := s.SendMessage(reply + rnd); err != nil {
		log.Printf("[##] send error: %s\n", err.Error())
	}
}

func (b *bot) staticMessage(_ context.Context, m message, s *dggchat.Session) {
	if resp, ok := staticCommands.lookup(m.Message); ok {
		b.sendMessageDedupe(m, resp, s)
	}
}

// !nuke str, !nukeregex regexp
func (b *bot) nuke(_ context.Context, m message, s *dggchat.Session) {
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
		b.sendMessageDedupe(m, "regexp error", s)
		return
	}

	// find anyone saying badstr
	// TODO limit by time, not amout of messages...
	victimNames := []string{}
	// the command itself will be last in the log and caught, exclude that one.
	// TODO: except if the command was issued via PM...
	history := b.log
	if len(history) > 0 {
		history = history[:len(history)-1]
	}
	for _, m := range history {
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

	// combine slices so we are able to undo all past nukes at once, if necessary
	b.lastNukeVictims = append(b.lastNukeVictims, victimNames...)
}

func (b *bot) sudoku(_ context.Context, m message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!sudoku") {
		return
	}
	// TODO duration, -1 means server default
	s.SendMute(m.Sender.Nick, -1)
}

// !aegis - undo (all) past nukes
func (b *bot) aegis(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!aegis") {
		return
	}

	for _, nick := range b.lastNukeVictims {
		s.SendUnmute(nick)
	}
	b.lastNukeVictims = nil
}

// !rename - change a chatter's username
func (b *bot) rename(ctx context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!rename") {
		return
	}

	parts := strings.Fields(m.Message)
	if len(parts) < 3 {
		return
	}

	oldName, newName := parts[1], parts[2]
	if err := b.renameUser(ctx, oldName, newName); err != nil {
		msg := fmt.Sprintf("'%s' to '%s' by %s failed with '%s'",
			oldName, newName, m.Sender.Nick, err.Error())
		log.Printf("[##] rename: %s\n", msg)

		// the detailed failure always goes to the sender privately, so only
		// add the public "check logs" note when the command wasn't a PM
		s.SendPrivateMessage(m.Sender.Nick, msg)
		if !m.isPM {
			b.sendMessageDedupe(m, "rename error, check logs", s)
		}
		return
	}
	log.Printf("[##] rename: '%s' to '%s' by '%s' success!\n",
		oldName, newName, m.Sender.Nick)
	b.sendMessageDedupe(m, fmt.Sprintf("name changed, %s please reconnect", oldName), s)
}

// !say - say a message
func (b *bot) say(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!say") {
		return
	}

	// message itself can contain spaces
	parts := strings.SplitN(m.Message, " ", 2)
	if len(parts) != 2 {
		return
	}
	b.sendPublicDedupe(parts[1], s)
}

// !mute - mute a chatter for a given time
func (b *bot) mute(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!mute") {
		return
	}
	parts := strings.Fields(m.Message)
	if len(parts) < 2 {
		return
	}

	duration := time.Duration(-1)
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
func (b *bot) unmute(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!unmute") {
		return
	}
	parts := strings.Fields(m.Message)
	if len(parts) < 2 {
		return
	}
	s.SendUnmute(parts[1])
}

// normalizeCommandName splits "!addcommand foo bar baz" style messages into
// the command prefix, the command name (always "!"-prefixed), and everything
// after it. ok is false if fewer than minParts whitespace-separated fields
// are present (extra whitespace collapses instead of shifting field indices).
func normalizeCommandName(msg string, minParts int) (cmnd string, rest []string, ok bool) {
	parts := strings.Fields(msg)
	if len(parts) < minParts {
		return "", nil, false
	}
	cmnd = parts[1]
	if !strings.HasPrefix(cmnd, "!") {
		cmnd = "!" + cmnd
	}
	return cmnd, parts[2:], true
}

// !addcommand command response
func (b *bot) addCommand(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!addcommand") {
		return
	}

	cmnd, respParts, ok := normalizeCommandName(m.Message, 3)
	if !ok {
		return
	}
	resp := strings.Join(respParts, " ")

	if err := staticCommands.set(cmnd, resp); err != nil {
		log.Printf("[##] failed adding command %s: %v\n", cmnd, err)
		b.sendMessageDedupe(m, "failed saving command, check logs", s)
		return
	}
	b.sendMessageDedupe(m, "added new command "+cmnd, s)
}

// !delcommand command
func (b *bot) delCommand(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!delcommand") {
		return
	}

	cmnd, _, ok := normalizeCommandName(m.Message, 2)
	if !ok {
		return
	}

	if err := staticCommands.delete(cmnd); err != nil {
		log.Printf("[##] failed deleting command %s: %v\n", cmnd, err)
		b.sendMessageDedupe(m, "failed saving command, check logs", s)
		return
	}
	b.sendMessageDedupe(m, fmt.Sprintf("deleted command %s if it existed", cmnd), s)
}

var trailingYearRegexp = regexp.MustCompile(`^(.*\S)\s+(\d{4})$`)

// !imdb [-tv|-s] title [year] -- look up a movie/show and print title (year) - rating - link
// defaults to a direct movie lookup; -tv looks up a series instead; -s searches and lists matches
func (b *bot) imdb(ctx context.Context, m message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!imdb") {
		return
	}

	if omdbAPIKey == "" {
		b.sendMessageDedupe(m, "imdb lookup not configured", s)
		return
	}

	const usage = "usage: !imdb [-tv|-s] <title> [year]"

	parts := strings.SplitN(m.Message, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		b.sendMessageDedupe(m, usage, s)
		return
	}
	rest := strings.TrimSpace(parts[1])

	mediaType := "movie"
	search := false
	switch {
	case strings.HasPrefix(rest, "-tv "):
		mediaType = "series"
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "-tv "))
	case strings.HasPrefix(rest, "-s "):
		search = true
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "-s "))
	}
	if rest == "" {
		b.sendMessageDedupe(m, usage, s)
		return
	}

	if search {
		sr, err := searchIMDb(ctx, rest, mediaType)
		if err != nil {
			b.sendMessageDedupe(m, "imdb: "+err.Error(), s)
			return
		}
		results := sr.Search[:min(len(sr.Search), 5)]
		matches := make([]string, 0, len(results))
		for _, item := range results {
			matches = append(matches, fmt.Sprintf("%s (%s)", item.Title, item.Year))
		}
		b.sendMessageDedupe(m, strings.Join(matches, ", "), s)
		return
	}

	title, year := rest, ""
	if match := trailingYearRegexp.FindStringSubmatch(rest); match != nil {
		title, year = match[1], match[2]
	}

	info, err := getIMDbInfo(ctx, title, year, mediaType)
	if err != nil {
		b.sendMessageDedupe(m, "imdb: "+err.Error(), s)
		return
	}
	b.sendMessageDedupe(m, formatIMDbInfo(info), s)
}

func formatIMDbInfo(info omdbResp) string {
	rating := info.ImdbRating
	if rating == "" || rating == "N/A" {
		rating = "no rating"
	}
	link := "https://www.imdb.com/title/" + info.ImdbID
	return fmt.Sprintf("%s (%s) - %s - %s", info.Title, info.Year, rating, link)
}

// TODO clean up...
func isCommunityStream(path string) bool {
	// "/twitch/test" it not. "/memer" is.
	return strings.Count(path, "/") == 1 || strings.Contains(path, "angelthump")
}

// !stream or !strim(s) -- show top streams in chat
func (b *bot) printTopStreams(ctx context.Context, m message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!stream") && !strings.HasPrefix(m.Message, "!strim") {
		return
	}

	sd, err := b.getStreamList(ctx)
	if err != nil {
		log.Printf("%v\n", err)
		b.sendMessageDedupe(m, "error getting api data", s)
		return
	}

	// - assumption: API gives json data sorted by "rustlers".
	// - community streams get preference, the rest fills up the remaining slots
	// - data.URL has leading slash
	var community, rest []string
	for _, v := range sd.StreamList {
		if v.Hidden {
			continue
		}
		nsfw := ""
		if v.Nsfw {
			nsfw = " [nsfw]"
		}
		out := fmt.Sprintf("%d %s%s%s", v.Rustlers, websiteURL, v.URL, nsfw)
		if isCommunityStream(v.URL) {
			community = append(community, out)
		} else {
			rest = append(rest, out)
		}
	}

	top := slices.Concat(community, rest)
	if len(top) == 0 {
		b.sendMessageDedupe(m, "no streams are being watched", s)
		return
	}

	for _, out := range top[:min(len(top), 3)] {
		b.sendMessageDedupe(m, out, s)
	}
}

func parseModifiers(mods []string) (streamModifier, error) {
	var sm streamModifier

	for _, part := range mods {
		// a leading "!" inverts the modifier: "!nsfw" clears the nsfw flag
		name, negated := strings.CutPrefix(part, "!")
		value := !negated

		switch name {
		case "nsfw":
			sm.Nsfw = &value
		case "hidden":
			sm.Hidden = &value
		case "afk":
			sm.Afk = &value
		case "promoted":
			sm.Promoted = &value
		default:
			return streamModifier{}, fmt.Errorf("invalid modifier: '%s'", part)
		}
	}

	return sm, nil
}

func (b *bot) modifyStream(ctx context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || !strings.HasPrefix(m.Message, "!modify") {
		return
	}

	//                       parts[2:], ...
	// !modify youtube/memes nsfw !hidden ...
	parts := strings.Fields(m.Message)
	if len(parts) < 3 {
		return
	}

	sm, err := parseModifiers(parts[2:])
	if err != nil {
		b.sendMessageDedupe(m, fmt.Sprintf("%s %s", err.Error(), ominousEmote), s)
		return
	}

	identifier := parts[1]

	if err := b.setStreamAttributes(ctx, identifier, sm); err != nil {
		log.Printf("[##] modify: '%s' with modifier '%+v' by '%s' failed with '%s'\n",
			identifier, sm, m.Sender.Nick, err.Error())

		// TODO chat message less verbose
		b.sendMessageDedupe(m, fmt.Sprintf("modify: %s %s", err, ominousEmote), s)
		return
	}
	log.Printf("[##] modify: '%s' with modifier '%+v' by '%s' success!\n",
		identifier, sm, m.Sender.Nick)
	b.sendMessageDedupe(m, "modify success "+ominousEmote, s)
}

// !check ATusername
func (b *bot) checkAT(ctx context.Context, m message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!check") {
		return
	}

	parts := strings.Fields(m.Message)
	if len(parts) != 2 {
		return
	}
	username := parts[1]

	atd, err := b.getATUserData(ctx, username)
	if err != nil {
		log.Printf("[##] checkAT error1: '%s'\n", err.Error())

		if errors.Is(err, errNotFound) {
			log.Printf("[##] check: not found\n")
			return
		}

		b.sendMessageDedupe(m, "error getting api data", s)
		return
	}

	// additionally check strim data
	sd, err := b.getStreamList(ctx)
	if err != nil {
		log.Printf("[##] checkAT error2: '%s'\n", err.Error())
		b.sendMessageDedupe(m, "error getting api data", s)
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

	b.sendMessageDedupe(m, output, s)
}

// !(un)drop atUser
func (b *bot) dropAT(ctx context.Context, m message, s *dggchat.Session) {
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
			m.Sender.Nick+" - please provide a ban reason")
		return
	}
	if doBan {
		reason = parts[2]
	}

	reply, err := b.banATuser(ctx, username, reason, doBan)
	if err != nil {
		log.Printf("[##] drop error: '%s'\n", err.Error())
		return
	}

	//	b.sendMessageDedupe(m, reply, s)
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
		// {"sec", seconds},
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
func (b *bot) ban(_ context.Context, m message, s *dggchat.Session) {
	if !isMod(m.Sender) || (!strings.HasPrefix(m.Message, "!ban") && !strings.HasPrefix(m.Message, "!unban")) {
		return
	}

	parts := strings.Fields(m.Message)
	if len(parts) < 2 {
		return
	}

	switch parts[0] {
	case "!ban":
		reason := ""
		if len(parts) == 3 {
			reason = parts[2]
		}
		s.SendBan(parts[1], reason, 0, false)
	case "!unban":
		s.SendUnban(parts[1])
	}
}

var (
	errInputFormat       = errors.New("invalid input format")
	errInputBounds       = errors.New("input out of bounds")
	errResultRangeBounds = errors.New("result range out of bounds")
)

// !roll [count]d<sides>[+/-modifier]
var rollRegexp = regexp.MustCompile(`^!rolls?\s+(\d+)(?:d(\d+))?\s*([+\-]\s*\d+)?`)

func computeRoll(input string) (int, error) {
	matches := rollRegexp.FindStringSubmatch(input)
	if matches == nil {
		return 0, fmt.Errorf("%w: %s", errInputFormat, input)
	}

	numDice, _ := strconv.Atoi(matches[1])
	numSides, _ := strconv.Atoi(matches[2])

	// bare "!roll 20" means one d20
	if matches[2] == "" {
		numSides = numDice
		numDice = 1
	}

	// the regexp tolerates whitespace around the sign ("2d2 + 100"), Atoi does not
	modifier, _ := strconv.Atoi(strings.ReplaceAll(matches[3], " ", ""))

	if numSides <= 0 || numDice <= 0 || numDice > 1000 {
		return 0, errInputBounds
	}

	if math.MaxInt64/numSides < numDice ||
		(modifier > 0 && math.MaxInt64-numSides*numDice < modifier) ||
		(modifier < 0 && math.MinInt64+numSides*numDice > modifier) {
		return 0, errResultRangeBounds
	}

	result := 0
	for range numDice {
		result += rand.IntN(numSides) + 1
	}

	return result + modifier, nil
}

// !roll sides [count] - roll dice
func (b *bot) roll(_ context.Context, m message, s *dggchat.Session) {
	if !strings.HasPrefix(m.Message, "!roll") {
		return
	}

	sum, err := computeRoll(m.Message)
	if err != nil {
		return
	}

	b.sendMessageDedupe(m, fmt.Sprintf("%s rolled %d", m.Sender.Nick, sum), s)
}
