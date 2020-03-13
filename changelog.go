package main

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

func getLatestChanges(repo string, n int) ([]string, error) {
	r, err := git.PlainOpen(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to open the repo %q: %v", repo, err)
	}
	var msgs []string
	cIter, err := r.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed 'git log': %v", err)
	}

	re := regexp.MustCompile(`\r?\n`)
	for i := 0; i < n; i++ {
		com, err := cIter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next commit log: %v", err)
		}
		// strip all new lines, replace with spaces
		// trim and extra leading or trailing spaces
		msgs = append(msgs, strings.TrimSpace(re.ReplaceAllString(com.Message, " ")))
	}

	return msgs, nil
}
