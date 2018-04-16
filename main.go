package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/voloshink/dggchat"
)

func main() {

	// TODO load token from config
	dgg, err := dggchat.New("")
	if err != nil {
		log.Fatalln(err)
	}

	// TODO load url from config
	u, err := url.Parse("wss://chat.strims.gg/ws")
	if err != nil {
		log.Fatalln(err)
	}
	dgg.SetURL(*u)

	err = dgg.Open()
	if err != nil {
		log.Fatalln(err)
	}
	defer dgg.Close()

	// init ...
	b := newBot(250)
	b.addParser(staticMessage)
	b.addParser(b.nuke)
	b.addParser(b.aegis)
	b.addParser(b.antiSingleCharSpam)

	dgg.AddMessageHandler(b.onMessage)
	dgg.AddErrorHandler(b.onError)

	// Wait for ctr-C to shut down
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc
}
