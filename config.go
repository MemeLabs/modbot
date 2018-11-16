package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type config struct {
	AuthCookie           string `json:"authCookie"`
	ChatWebsocket        string `json:"chatWebsocket"`
	APIUrl               string `json:"apiUrl"`
	LogFile              string `json:"logFile"`
	LogOnly              bool   `json:"logOnly"`
	AngelthumpAdminToken string `json:"angelthumpAdminToken"`
	Database             string `json:"database"`
}

func (b *bot) loadConfig() {
	d, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("failed loading config file: %v", err)
	}
	err = json.Unmarshal(d, &b.config)
	if err != nil {
		log.Fatalf("failed unmarshaling config file: %v", err)
	}
}
