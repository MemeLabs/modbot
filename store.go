package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// staticCommandStore holds the mod-defined "!command -> response" pairs, backed
// by a JSON file on disk. It is read on every chat message and written only by
// !addcommand, hence the RWMutex.
type staticCommandStore struct {
	mu   sync.RWMutex
	path string
	// resolved so deploy logs show exactly which file/volume is in play -- the
	// flag default is a bare relative name and silently follows whatever the
	// process's CWD happens to be on a given deploy.
	absPath string
	cmds    map[string]string
}

func newStaticCommandStore(path string) *staticCommandStore {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return &staticCommandStore{path: path, absPath: absPath, cmds: map[string]string{}}
}

// load reads the backing file, creating an empty one if it does not exist yet.
func (s *staticCommandStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		log.Printf("commands file %s not found, creating empty one\n", s.absPath)
		s.cmds = map[string]string{}
		return s.save()
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", s.absPath, err)
	}

	cmds := map[string]string{}
	if err := json.Unmarshal(b, &cmds); err != nil {
		return fmt.Errorf("parsing %s: %w", s.absPath, err)
	}
	s.cmds = cmds
	log.Printf("loaded %d static command(s) from %s\n", len(s.cmds), s.absPath)
	return nil
}

// save writes the store to disk atomically. Callers must hold the write lock.
// A failure here means an acknowledged !addcommand was lost, so it is counted.
func (s *staticCommandStore) save() error {
	if err := s.writeFile(); err != nil {
		commandPersistFailures.Inc()
		return err
	}
	return nil
}

func (s *staticCommandStore) writeFile() error {
	b, err := json.MarshalIndent(s.cmds, "", "\t")
	if err != nil {
		return fmt.Errorf("marshaling commands: %w", err)
	}

	// Write to a temp file in the same directory and rename, so a crash
	// mid-write cannot leave a truncated commands file behind.
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".commands-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp.Name(), err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return fmt.Errorf("chmod %s: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("renaming into %s: %w", s.path, err)
	}
	return nil
}

// lookup returns the response for the longest command that msg starts with.
// Longest-prefix wins so that "!foo" and "!foobar" resolve deterministically;
// ranging over the map picked an arbitrary one.
func (s *staticCommandStore) lookup(msg string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best, resp string
	for cmd, r := range s.cmds {
		if strings.HasPrefix(msg, cmd) && len(cmd) > len(best) {
			best, resp = cmd, r
		}
	}
	return resp, best != ""
}

func (s *staticCommandStore) set(cmd, resp string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cmds[cmd] = resp
	return s.save()
}

func (s *staticCommandStore) delete(cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cmds, cmd)
	return s.save()
}
