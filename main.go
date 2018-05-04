package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/voloshink/dggchat"
)

var (
	authCookie string
	chatURL    string
	backendURL string
)

func init() {
	flag.StringVar(&authCookie, "cookie", "", "Cookie used for chat authentication and API access")
	flag.StringVar(&chatURL, "chat", "wss://chat.strims.gg/ws", "ws(s)-url for chat")
	flag.StringVar(&backendURL, "api", "https://strims.gg/api", "basic backend api path")
	flag.Parse()
}

func main() {

	// TODO dggchat lib isn't flexible with the cookie name, workaround...
	dgg, err := dggchat.New(";jwt=" + authCookie)
	if err != nil {
		log.Fatalln(err)
	}

	// TODO load url from config
	//u, err := url.Parse("wss://chat.strims.gg/ws")
	u, err := url.Parse(chatURL)
	if err != nil {
		log.Fatalln(err)
	}
	dgg.SetURL(*u)

	err = dgg.Open()
	if err != nil {
		log.Fatalln(err)
	}
	debuglogger.Println("chat connection ok")
	defer dgg.Close()

	// init ...
	b := newBot(authCookie, 250)
	b.addParser(b.staticMessage)
	b.addParser(b.nuke)
	b.addParser(b.aegis)
	b.addParser(b.antiSingleCharSpam)
	b.addParser(b.rename)

	dgg.AddMessageHandler(b.onMessage)
	dgg.AddErrorHandler(b.onError)
	dgg.AddMuteHandler(b.onMute)
	dgg.AddUnmuteHandler(b.onUnmute)
	dgg.AddBanHandler(b.onBan)
	dgg.AddUnbanHandler(b.onUnban)
	debuglogger.Println("init done")

	info, err := b.getProfileInfo()
	if err != nil {
		debuglogger.Printf("userinfo: %s", err.Error())
	} else {
		debuglogger.Printf("userinfo: '%+v'\n", info)
	}

	// Wait for ctr-C to shut down
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc
}
