package main

import (
	"fmt"
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

	for i := 0; i < n; i++ {
		com, err := cIter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next commit log: %v", err)
		}

		if strings.Contains(com.Message, "CHANGELOG:") {
			msgs = append(msgs,
				strings.TrimSpace(strings.Split(com.Message, "CHANGELOG:")[1]))
		}
	}

	return msgs, nil
}
