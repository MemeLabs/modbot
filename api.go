package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authCookieName    = "jwt"
	apiRequestTimeout = 2 * time.Second
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
		Hidden    bool   `json:"hidden"`
		Rustlers  int    `json:"rustlers"`
		Service   string `json:"service"`
		Thumbnail string `json:"thumbnail"`
		URL       string `json:"url"`
		Viewers   int    `json:"viewers"`
	} `json:"stream_list"`
}

type errorResp struct {
	Error string `json:"error"`
}

func (b *bot) initHeaders(req *http.Request) *http.Request {

	c := fmt.Sprintf("%s=%s", authCookieName, b.authCookie)
	req.Header.Set("Cookie", c)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot", "botnet")
	return req
}

// Send rename request to backend.
func (b *bot) renameUser(oldName string, newName string) error {

	var jsonStr = []byte(fmt.Sprintf(`{"username":"%s"}`, newName))
	path := fmt.Sprintf("%s/admin/profiles/%s/username", backendURL, oldName)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req = b.initHeaders(req)

	client := &http.Client{Timeout: apiRequestTimeout}
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

// string because we don't want false default bools when marshaling
type streamModifier struct {
	Nsfw     string `json:"nsfw,omitempty"`
	Hidden   string `json:"hidden,omitempty"`
	Afk      string `json:"afk,omitempty"`
	Promoted string `json:"promoted,omitempty"`
}

// Modify stream attributes (nsfw/hidden/...)
// identifier can be a stream_path (simple string) or "{service}/{channel}"
func (b *bot) setStreamAttributes(identifier string, modifier streamModifier) error {

	jsonStr, err := json.Marshal(&modifier)
	if err != nil {
		return err
	}

	// backend does not like string-version of booleans,
	// but we don't like structs with bools because omitempty
	j := string(jsonStr[:])
	j = strings.Replace(j, "\"true\"", "true", -1)
	j = strings.Replace(j, "\"false\"", "false", -1)

	path := fmt.Sprintf("%s/admin/streams/%s", backendURL, identifier)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer([]byte(j)))
	if err != nil {
		return err
	}
	req = b.initHeaders(req)

	client := &http.Client{Timeout: apiRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		return nil
	}

	// backend tells us a custom error message
	var e errorResp
	err = json.NewDecoder(resp.Body).Decode(&e)
	if err != nil {
		return err
	}

	return fmt.Errorf("error: %s", e.Error)
}

// build common get request...
func (b *bot) buildGetRequest(path string) (*http.Request, error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", backendURL, path), nil)
	if err != nil {
		return nil, err
	}
	req = b.initHeaders(req)
	return req, nil
}

// get basic user info - to check if we are logged in and have correct rights
func (b *bot) getProfileInfo() (userInfo, error) {

	req, err := b.buildGetRequest("/profile")
	if err != nil {
		return userInfo{}, err
	}
	client := &http.Client{Timeout: apiRequestTimeout}
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
	client := &http.Client{Timeout: apiRequestTimeout}
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

// at api data
type atData struct {
	Username          string `json:"username"`
	Live              bool   `json:"live"`
	Title             string `json:"title"`
	Viewers           int    `json:"viewers"`
	PasswordProtected bool   `json:"passwordProtected"`
	Banned            bool   `json:"banned"`
	Poster            string `json:"poster"`
	Thumbnail         string `json:"thumbnail"`
	CreatedAt         string `json:"created_at"`
}

// interact with at backend
func (b *bot) getATUserData(username string) (atData, error) {

	path := fmt.Sprintf("https://api.angelthump.com/v1/%s", username)
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return atData{}, err
	}
	req = b.initHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot", "botnet")

	client := &http.Client{Timeout: apiRequestTimeout * 2}
	resp, err := client.Do(req)
	if err != nil {
		return atData{}, err
	}

	// don't check status code, the backend doesn't report it correctly.
	// if user does not exist, content type is text/html.
	if !strings.Contains(resp.Header.Get("content-type"), "application/json") {
		return atData{}, errors.New("User not found - 404")
	}

	var atd atData
	err = json.NewDecoder(resp.Body).Decode(&atd)
	if err != nil {
		return atData{}, err
	}

	return atd, nil
}

// (un)ban AT user
func (b *bot) banATuser(username string, reason string, ban bool) (string, error) {

	if reason == "" {
		reason = "no reason provided"
	}

	action := "unban"
	if ban {
		action = "ban"
	}

	path := fmt.Sprintf("https://api.angelthump.com/admin/v1/%s", action)

	req, err := http.NewRequest(http.MethodPost, path, strings.NewReader(
		fmt.Sprintf("username=%s&reason=%s", username, url.QueryEscape(reason))))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Bot", "botnet")
	req.Header.Set("Authorization", fmt.Sprintf("key %s", atAdminToken))

	client := &http.Client{Timeout: apiRequestTimeout * 2}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("drop: %s", string(responseData)), nil
}
