package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"gopkg.in/src-d/go-git.v4"
)

func TestGetLatestChanges(t *testing.T) {
	want := []string{"doc: add setup instructions  Closes https://github.com/MemeLabs/comfybot/issues/2", "deps: @glennsl/bs-json@^3.0.0  Closes https://github.com/MemeLabs/comfybot/issues/1", "remove unused env variables"}

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Errorf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	if _, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/MemeLabs/comfybot.git",
	}); err != nil {
		t.Errorf("failed to clone test repo: %v", err)
	}

	got, err := getLatestChanges(dir, 3)
	if err != nil {
		t.Errorf("getLatestChanges failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
