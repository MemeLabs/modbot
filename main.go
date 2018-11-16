package main

import (
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voloshink/dggchat"
)

var (
	debuglogger = log.New(os.Stdout, "[d] ", log.Ldate|log.Ltime|log.Lshortfile)
	logFile     *os.File
)

const (
	websiteURL   = "strims.gg"
	pollTime     = time.Second * 2
	ominousEmote = "BOGGED"
)

func main() {
	// init bot
	b := newBot(250)
	b.loadConfig()
	b.newDatabase()
	b.LoadStaticCommands()

	// TODO dggchat lib isn't flexible with the cookie name, workaround...
	dgg, err := dggchat.New(";jwt=" + b.config.AuthCookie)
	if err != nil {
		log.Fatalln(err)
	}

	b.addParser(
		b.staticMessage,
		b.nuke,
		b.aegis,
		b.noShortMsgSpam,
		b.rename,
		b.say,
		b.addCommand,
		b.deleteCommand,
		b.mute,
		b.printTopStreams,
		b.modifyStream,
		b.checkAT,
		b.embedLink,
		b.dropAT,
	)
	dgg.AddMessageHandler(b.onMessage)
	dgg.AddErrorHandler(b.onError)
	dgg.AddMuteHandler(b.onMute)
	dgg.AddUnmuteHandler(b.onUnmute)
	dgg.AddBanHandler(b.onBan)
	dgg.AddUnbanHandler(b.onUnban)
	dgg.AddSocketErrorHandler(b.onSocketError)
	dgg.AddPMHandler(b.onPMHandler)

	u, err := url.Parse(b.config.ChatWebsocket)
	if err != nil {
		log.Fatalln(err)
	}
	dgg.SetURL(*u)

	err = dgg.Open()
	if err != nil {
		log.Fatalln(err)
	}
	debuglogger.Println("[##] connected...")
	defer dgg.Close()

	info, err := b.getProfileInfo()
	if err != nil {
		debuglogger.Printf("userinfo: %s", err.Error())
	} else {
		debuglogger.Printf("userinfo: '%+v'", info)
	}

	// log to file and stdout
	logFile = b.reOpenLog()
	log.Println("[##] Restart")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if b.config.LogOnly {
		debuglogger.Println("[##] started in logonly mode.")
	}
	debuglogger.Println("[##] waiting for signals...")
	for {
		sig := <-signals
		switch sig {

		// handle logrotate request from daemon
		case syscall.SIGHUP:
			log.Println("[##] signal: handling SIGHUP")
			err := logFile.Close()
			if err != nil {
				panic(err)
			}
			logFile = b.reOpenLog()

		// exit on interrupt
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			log.Println("[##] signal: handling SIGINT/SIGTERM")
			err = logFile.Close()
			if err != nil {
				log.Printf("[##] error in cleanup: %s\n", err.Error())
			}
			os.Exit(1)
		}
	}
}

func (b *bot) reOpenLog() *os.File {

	f, err := os.OpenFile(b.config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	return f
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
