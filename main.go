package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/voloshink/dggchat"
)

var (
	debuglogger = log.New(os.Stdout, "[d] ", log.Ldate|log.Ltime|log.Lshortfile)
	authCookie  string
	chatURL     string
	backendURL  string
	logFileName string
	commandJson string
	logFile     *os.File
)

func init() {
	flag.StringVar(&authCookie, "cookie", "", "Cookie used for chat authentication and API access")
	flag.StringVar(&chatURL, "chat", "wss://chat.strims.gg/ws", "ws(s)-url for chat")
	flag.StringVar(&backendURL, "api", "https://strims.gg/api", "basic backend api path")
	flag.StringVar(&logFileName, "log", "/tmp/chatlog/chatlog.log", "file to write messages to")
	flag.StringVar(&commandJson, "commands", "commands.json", "static commands file")
	flag.Parse()
}

func main() {

	loadStaticCommands()

	// TODO dggchat lib isn't flexible with the cookie name, workaround...
	dgg, err := dggchat.New(";jwt=" + authCookie)
	if err != nil {
		log.Fatalln(err)
	}

	// init bot
	b := newBot(authCookie, 250)
	b.addParser(b.staticMessage)
	b.addParser(b.nuke)
	b.addParser(b.aegis)
	b.addParser(b.antiSingleCharSpam)
	b.addParser(b.rename)
	b.addParser(b.say)
	b.addParser(b.addCommand)
	b.addParser(b.mute)
	dgg.AddMessageHandler(b.onMessage)
	dgg.AddErrorHandler(b.onError)
	dgg.AddMuteHandler(b.onMute)
	dgg.AddUnmuteHandler(b.onUnmute)
	dgg.AddBanHandler(b.onBan)
	dgg.AddUnbanHandler(b.onUnban)
	dgg.AddSocketErrorHandler(b.onSocketError)
	dgg.AddPMHandler(b.onPMHandler)

	u, err := url.Parse(chatURL)
	if err != nil {
		log.Fatalln(err)
	}
	dgg.SetURL(*u)

	err = dgg.Open()
	if err != nil {
		log.Fatalln(err)
	}
	debuglogger.Println("connected...")
	defer dgg.Close()

	info, err := b.getProfileInfo()
	if err != nil {
		debuglogger.Printf("userinfo: %s", err.Error())
	} else {
		debuglogger.Printf("userinfo: '%+v'\n", info)
	}

	// log to file and stdout
	logFile = reOpenLog()
	log.Println("[##] Restart")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	debuglogger.Println("waiting for signals...")
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
			logFile = reOpenLog()

		// exit on interrupt
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			log.Println("[##] signal: handling SIGINT/SIGTERM")
			err = logFile.Close()
			if err != nil {
				log.Printf("[##] error in cleanup: %s", err.Error())
			}
			os.Exit(1)
		}
	}
}

func reOpenLog() *os.File {
	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	return f
}

func loadStaticCommands() {
	b, err := ioutil.ReadFile(commandJson)
	if err != nil {
		panic(err)
	}
	var cmnd map[string]string
	err = json.Unmarshal(b, &cmnd)
	if err != nil {
		panic(err)
	}
	commands = cmnd
}

func saveStaticCommands() bool {
	s, err := json.MarshalIndent(commands, "", "\t")
	if err != nil {
		log.Printf("failed marshaling commands, error: %v", err)
		return false
	}
	err = ioutil.WriteFile(commandJson, s, 0755)
	if err != nil {
		log.Printf("failed saving commands, error: %v", err)
		return false
	}
	return true
}
