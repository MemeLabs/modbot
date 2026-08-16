package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authCookieName    = "jwt"
	apiRequestTimeout = 2 * time.Second
	atRequestTimeout  = 4 * time.Second
)

// One client for the whole process: http.Client is safe for concurrent use and
// pools connections, which a fresh client per call cannot do. Deadlines come
// from the request context rather than Client.Timeout so callers can also be
// cancelled on shutdown.
var httpClient = &http.Client{}

// errNotFound is returned when the upstream has no record of the requested user.
var errNotFound = errors.New("user not found")

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

// newRequest builds a backend request carrying the bot's auth cookie.
func (b *bot) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", authCookieName, b.authCookie))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot", "botnet")
	return req, nil
}

// getJSON issues a GET against the backend and decodes the response into out.
func (b *bot) getJSON(ctx context.Context, path string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, apiRequestTimeout)
	defer cancel()

	req, err := b.newRequest(ctx, http.MethodGet, backendURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(out)
}

// Send rename request to backend.
func (b *bot) renameUser(ctx context.Context, oldName, newName string) error {
	payload, err := json.Marshal(map[string]string{"username": newName})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, apiRequestTimeout)
	defer cancel()

	path := fmt.Sprintf("%s/admin/profiles/%s/username", backendURL, url.PathEscape(oldName))
	req, err := b.newRequest(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var msg struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &msg); err != nil || msg.Message == "" {
		return fmt.Errorf("status code %d, %s", resp.StatusCode, body)
	}
	return errors.New(msg.Message)
}

// Pointers so an unset modifier is omitted while an explicit false is still sent.
type streamModifier struct {
	Nsfw     *bool `json:"nsfw,omitempty"`
	Hidden   *bool `json:"hidden,omitempty"`
	Afk      *bool `json:"afk,omitempty"`
	Promoted *bool `json:"promoted,omitempty"`
}

// Modify stream attributes (nsfw/hidden/...)
// identifier can be a stream_path (simple string) or "{service}/{channel}"
func (b *bot) setStreamAttributes(ctx context.Context, identifier string, modifier streamModifier) error {
	payload, err := json.Marshal(&modifier)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, apiRequestTimeout)
	defer cancel()

	path := fmt.Sprintf("%s/admin/streams/%s", backendURL, identifier)
	req, err := b.newRequest(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// backend tells us a custom error message
	var e errorResp
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return fmt.Errorf("status code %d: %w", resp.StatusCode, err)
	}

	return fmt.Errorf("error: %s", e.Error)
}

// get basic user info - to check if we are logged in and have correct rights
func (b *bot) getProfileInfo(ctx context.Context) (userInfo, error) {
	var ui userInfo
	if err := b.getJSON(ctx, "/profile", &ui); err != nil {
		return userInfo{}, err
	}
	return ui, nil
}

// Get list of current streams.
func (b *bot) getStreamList(ctx context.Context) (streamData, error) {
	// empty path (/api) holds stream data...
	var sd streamData
	if err := b.getJSON(ctx, "", &sd); err != nil {
		return streamData{}, err
	}
	return sd, nil
}

const omdbURL = "https://www.omdbapi.com/"

type omdbResp struct {
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	ImdbID     string `json:"imdbID"`
	ImdbRating string `json:"imdbRating"`
	Response   string `json:"Response"`
	Error      string `json:"Error"`
}

type omdbSearchResp struct {
	Search []struct {
		Title  string `json:"Title"`
		Year   string `json:"Year"`
		ImdbID string `json:"imdbID"`
		Type   string `json:"Type"`
	} `json:"Search"`
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

// omdbGet issues a GET against OMDb with the given query params and decodes the
// result into out (apikey is added automatically).
func omdbGet(ctx context.Context, params map[string]string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, apiRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, omdbURL, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("apikey", omdbAPIKey)
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(out)
}

// direct title lookup, scoped to a media type (movie/series) and optionally a release year
func getIMDbInfo(ctx context.Context, query, year, mediaType string) (omdbResp, error) {
	params := map[string]string{"t": query, "type": mediaType}
	if year != "" {
		params["y"] = year
	}

	var or omdbResp
	if err := omdbGet(ctx, params, &or); err != nil {
		return omdbResp{}, err
	}
	if or.Response == "False" {
		return omdbResp{}, errors.New(or.Error)
	}
	return or, nil
}

// search titles matching the free-text query, scoped to a media type (movie/series)
func searchIMDb(ctx context.Context, query, mediaType string) (omdbSearchResp, error) {
	var sr omdbSearchResp
	if err := omdbGet(ctx, map[string]string{"s": query, "type": mediaType}, &sr); err != nil {
		return omdbSearchResp{}, err
	}
	if sr.Response == "False" {
		return omdbSearchResp{}, errors.New(sr.Error)
	}
	return sr, nil
}

// at api data
type atData struct {
	ViewerCount int `json:"viewer_count"`
	User        struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Title           string `json:"title"`
		Angel           bool   `json:"angel"`
		Nsfw            bool   `json:"nsfw"`
		Banned          bool   `json:"banned"`
		PasswordProtect bool   `json:"password_protect"`
	} `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// interact with at backend
func (b *bot) getATUserData(ctx context.Context, username string) (atData, error) {
	ctx, cancel := context.WithTimeout(ctx, atRequestTimeout)
	defer cancel()

	path := "https://api.angelthump.com/v3/streams/?username=" + url.QueryEscape(strings.ToLower(username))
	req, err := b.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return atData{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return atData{}, err
	}
	defer resp.Body.Close()

	// don't check status code, the backend doesn't report it correctly.
	// if user does not exist, content type is text/html.
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return atData{}, errNotFound
	}

	var atds []atData
	if err := json.NewDecoder(resp.Body).Decode(&atds); err != nil {
		return atData{}, err
	}
	if len(atds) == 0 {
		return atData{}, errNotFound
	}

	return atds[0], nil
}

// (un)ban AT user
func (b *bot) banATuser(ctx context.Context, username, reason string, ban bool) (string, error) {
	if reason == "" {
		reason = "no reason provided"
	}

	action := "unban"
	if ban {
		action = "ban"
	}

	ctx, cancel := context.WithTimeout(ctx, atRequestTimeout)
	defer cancel()

	form := url.Values{"username": {username}, "reason": {reason}}
	path := "https://streams.angelthump.com/v3/admin/" + action

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Bot", "botnet")
	req.Header.Set("Authorization", "key "+atAdminToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Error    bool   `json:"error"`
		ErrorMSG string `json:"errorMSG"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error {
		return "", fmt.Errorf("failed to %s with: %q", action, result.ErrorMSG)
	}

	return "success", nil
}
