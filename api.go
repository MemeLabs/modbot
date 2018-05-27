package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	authCookieName = "jwt"
)

type userInfo struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

type streamData struct {
	StreamList []struct {
		Channel   string `json:"channel"`
		Live      bool   `json:"live"`
		Nsfw      bool   `json:"nsfw"`
		Rustlers  int    `json:"rustlers"`
		Service   string `json:"service"`
		Thumbnail string `json:"thumbnail"`
		URL       string `json:"url"`
		Viewers   int    `json:"viewers"`
	} `json:"stream_list"`
}

// Send rename request to backend.
func (b *bot) renameUser(oldName string, newName string) error {

	var jsonStr = []byte(fmt.Sprintf(`{"username":"%s"}`, newName))
	path := fmt.Sprintf("%s/admin/profiles/%s/username", backendURL, oldName)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	c := fmt.Sprintf("%s=%s", authCookieName, b.authCookie)
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

// build common get request...
func (b *bot) buildGetRequest(path string) (*http.Request, error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", backendURL, path), nil)
	if err != nil {
		return nil, err
	}
	c := fmt.Sprintf("%s=%s", authCookieName, b.authCookie)
	req.Header.Set("X-Bot", "botnet")
	req.Header.Set("Cookie", c)
	return req, nil
}

// get basic user info - to check if we are logged in and have correct rights
func (b *bot) getProfileInfo() (userInfo, error) {

	req, err := b.buildGetRequest("/profile")
	if err != nil {
		return userInfo{}, err
	}
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

// Get list of current streams.
func (b *bot) getStreamList() (streamData, error) {

	// empty path (/api) holds stream data...
	req, err := b.buildGetRequest("")
	if err != nil {
		return streamData{}, err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return streamData{}, err
	}

	var sd streamData
	err = json.NewDecoder(resp.Body).Decode(&sd)
	if err != nil {
		return streamData{}, err
	}

	return sd, nil
}
