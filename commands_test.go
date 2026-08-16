package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseModifiers(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]bool
		wantErr bool
	}{
		{
			name:  "set two",
			input: []string{"nsfw", "hidden"},
			want:  map[string]bool{"nsfw": true, "hidden": true},
		},
		{
			name:  "invert",
			input: []string{"!nsfw", "promoted", "!afk"},
			want:  map[string]bool{"nsfw": false, "promoted": true, "afk": false},
		},
		{
			name:  "empty sends nothing",
			input: nil,
			want:  map[string]bool{},
		},
		{
			name:    "unknown modifier",
			input:   []string{"nsfw", "hide"},
			wantErr: true,
		},
		{
			name:    "inverted unknown modifier",
			input:   []string{"!hide"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseModifiers(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseModifiers(%v) = %+v, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseModifiers(%v): %v", tc.input, err)
			}

			// unset modifiers must be omitted entirely, explicit false must be sent
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var marshaled map[string]bool
			if err := json.Unmarshal(b, &marshaled); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if len(marshaled) != len(tc.want) {
				t.Fatalf("marshaled %s, want keys %v", b, tc.want)
			}
			for k, v := range tc.want {
				if marshaled[k] != v {
					t.Errorf("%s = %v, want %v (payload %s)", k, marshaled[k], v, b)
				}
			}
		})
	}
}

func TestComputeRoll(t *testing.T) {
	tests := []struct {
		input string
		err   error
	}{
		{input: "!roll 2d2+100 foo biz baz"},
		{input: "!roll 2d2 + 100"},
		{input: "!roll 2d2 +100"},
		{input: "!roll 2d2+ 100"},
		{input: "!roll 2d2-100"},
		{input: "!roll 2d2 - 100"},
		{input: "!roll 2d2 -100"},
		{input: "!roll 2d2- 100"},
		{input: "!roll 2d2- 100 foo biz baz"},
		{input: "!roll 2d20"},
		{input: "!roll 20"},
		{input: "!roll 20+10"},
		{input: "!roll 1d9223372036854775807"},
		{input: "!roll 1d9223372036854775807+1", err: errResultRangeBounds},
		{input: "!roll 9223372036854775807d1", err: errInputBounds},
		{input: "!roll 0d6", err: errInputBounds},
		{input: "!roll 1d0", err: errInputBounds},
		{input: "!roll 1001d6", err: errInputBounds},
		{input: "!roll", err: errInputFormat},
		{input: "!roll d20", err: errInputFormat},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if _, err := computeRoll(tc.input); !errors.Is(err, tc.err) {
				t.Fatalf("computeRoll(%q) error = %v, want %v", tc.input, err, tc.err)
			}
		})
	}
}

// A modifier separated from the dice by whitespace must still be applied.
func TestComputeRollModifierSpacing(t *testing.T) {
	for _, input := range []string{"!roll 1d1+100", "!roll 1d1 + 100", "!roll 1d1+ 100", "!roll 1d1 +100"} {
		got, err := computeRoll(input)
		if err != nil {
			t.Fatalf("computeRoll(%q): %v", input, err)
		}
		if got != 101 {
			t.Errorf("computeRoll(%q) = %d, want 101", input, got)
		}
	}
}

func TestComputeRollBounds(t *testing.T) {
	for range 100 {
		got, err := computeRoll("!roll 3d6")
		if err != nil {
			t.Fatalf("computeRoll: %v", err)
		}
		if got < 3 || got > 18 {
			t.Fatalf("computeRoll(!roll 3d6) = %d, want 3..18", got)
		}
	}
}

func TestIsCommunityStream(t *testing.T) {
	tests := map[string]bool{
		"/memer":               true,
		"/twitch/test":         false,
		"/angelthump/streamer": true,
		"/youtube/6n3pFFPSlW4": false,
	}
	for path, want := range tests {
		if got := isCommunityStream(path); got != want {
			t.Errorf("isCommunityStream(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestFormatIMDbInfo(t *testing.T) {
	got := formatIMDbInfo(omdbResp{Title: "Dune", Year: "2021", ImdbID: "tt1160419", ImdbRating: "8.0"})
	want := "Dune (2021) - 8.0 - https://www.imdb.com/title/tt1160419"
	if got != want {
		t.Errorf("formatIMDbInfo() = %q, want %q", got, want)
	}

	got = formatIMDbInfo(omdbResp{Title: "Dune", Year: "2021", ImdbID: "tt1160419", ImdbRating: "N/A"})
	if !strings.Contains(got, "no rating") {
		t.Errorf("formatIMDbInfo() = %q, want it to report no rating", got)
	}
}

func TestStaticCommandStore(t *testing.T) {
	path := t.TempDir() + "/commands.json"
	s := newStaticCommandStore(path)

	// a missing file is created rather than fatal
	if err := s.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := s.set("!test", "i like tests"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if resp, ok := s.lookup("!test with trailing words"); !ok || resp != "i like tests" {
		t.Errorf("lookup = %q, %v; want %q, true", resp, ok, "i like tests")
	}
	if _, ok := s.lookup("!nope"); ok {
		t.Error("lookup(!nope) matched")
	}

	// longest prefix wins, regardless of map ordering
	if err := s.set("!testlonger", "more specific"); err != nil {
		t.Fatalf("set: %v", err)
	}
	for range 20 {
		if resp, _ := s.lookup("!testlonger"); resp != "more specific" {
			t.Fatalf("lookup(!testlonger) = %q, want %q", resp, "more specific")
		}
	}

	// changes survive a reload from disk
	reloaded := newStaticCommandStore(path)
	if err := reloaded.load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if resp, ok := reloaded.lookup("!test"); !ok || resp != "i like tests" {
		t.Errorf("after reload lookup = %q, %v", resp, ok)
	}

	if err := s.delete("!test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	reloaded = newStaticCommandStore(path)
	if err := reloaded.load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := reloaded.lookup("!test"); ok {
		t.Error("deleted command survived reload")
	}
	if _, ok := reloaded.lookup("!testlonger"); !ok {
		t.Error("delete removed an unrelated command")
	}
}

func TestHumanizeDuration(t *testing.T) {
	tests := map[string]string{
		"1h30m": "1hour 30mins",
		"25h":   "1day 1hour",
		"90s":   "1min",
		"0s":    "",
	}
	for in, want := range tests {
		d, err := time.ParseDuration(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		if got := humanizeDuration(d); got != want {
			t.Errorf("humanizeDuration(%s) = %q, want %q", in, got, want)
		}
	}
}
