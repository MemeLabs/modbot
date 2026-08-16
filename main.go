package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/MemeLabs/dggchat"
)

var (
	debuglogger  = log.New(os.Stdout, "[d] ", log.Ldate|log.Ltime|log.Lshortfile)
	authCookie   string
	chatPath     string
	chatURL      string
	backendURL   string
	logFileName  string
	commandJSON  string
	atAdminToken string
	omdbAPIKey   string
	logOnly      bool
	showVersion  bool
	metricsAddr  string
	pingInterval time.Duration
	healthCheck  bool

	logFile        *os.File
	staticCommands *staticCommandStore
)

const (
	websiteURL   = "strims.gg"
	ominousEmote = "BOGGED"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

// run holds the real body of main so that deferred cleanup still executes on
// the error paths; log.Fatal in main would skip it.
func run() error {
	flag.StringVar(&authCookie, "cookie", "", "Cookie used for chat authentication and API access")
	// unused, kept so existing deploy invocations passing -path still start
	flag.StringVar(&chatPath, "path", "", "path to chat-gui (unused)")
	flag.StringVar(&chatURL, "chat", "wss://chat.strims.gg/ws", "ws(s)-url for chat")
	flag.StringVar(&backendURL, "api", "https://strims.gg/api", "basic backend api path")
	flag.StringVar(&logFileName, "log", "/tmp/chatlog/chatlog.log", "file to write messages to")
	flag.StringVar(&commandJSON, "commands", "commands.json", "static commands file")
	flag.StringVar(&atAdminToken, "attoken", "", "angelthump admin token (optional)")
	flag.StringVar(&omdbAPIKey, "omdbkey", "", "OMDb API key for !imdb command (optional)")
	flag.BoolVar(&logOnly, "logonly", false, "only 'reply' to logfile, not chat (for debugging)")
	flag.StringVar(&metricsAddr, "metrics", ":9090", "listen address for /metrics and /healthz")
	flag.DurationVar(&pingInterval, "pinginterval", 30*time.Second, "how often to send a keepalive PING")
	flag.BoolVar(&healthCheck, "healthcheck", false, "probe a running instance's /healthz and exit; used by the container HEALTHCHECK")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(buildVersion())
		return nil
	}

	if healthCheck {
		if err := checkHealth(metricsAddr); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return nil
	}

	staticCommands = newStaticCommandStore(commandJSON)
	if err := staticCommands.load(); err != nil {
		return err
	}

	serveMetrics(metricsAddr)

	// cancelled on SIGINT/SIGTERM; parsers inherit it so in-flight API calls
	// are dropped instead of holding up shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO dggchat lib isn't flexible with the cookie name, workaround...
	dgg, err := dggchat.New(";jwt=" + authCookie)
	if err != nil {
		return err
	}

	// init bot
	b := newBot(ctx, authCookie, 250)
	b.addParser(
		b.staticMessage,
		b.nuke,
		b.aegis,
		b.noShortMsgSpam,
		b.rename,
		b.say,
		b.addCommand,
		b.delCommand,
		b.mute,
		b.unmute,
		b.printTopStreams,
		b.modifyStream,
		b.checkAT,
		b.dropAT,
		b.ban,
		b.sudoku,
		b.roll,
		b.imdb,
	)
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
		return fmt.Errorf("parsing chat url %q: %w", chatURL, err)
	}
	dgg.SetURL(*u)

	if err := dgg.Open(); err != nil {
		return fmt.Errorf("connecting to chat: %w", err)
	}
	setConnected(true)
	debuglogger.Printf("[##] connected... (version %s)\n", buildVersion())
	defer dgg.Close()

	go pinger(dgg, pingInterval)

	info, err := b.getProfileInfo(ctx)
	if err != nil {
		debuglogger.Printf("userinfo: %s\n", err.Error())
	} else {
		debuglogger.Printf("userinfo: '%+v'\n", info)
	}

	// log to file and stdout
	logFile, err = reOpenLog()
	if err != nil {
		return err
	}
	log.Println("[##] Restart")

	// logrotate signals us out-of-band; SIGINT/SIGTERM land on ctx instead.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	if logOnly {
		debuglogger.Println("[##] started in logonly mode.")
	}
	debuglogger.Println("[##] waiting for signals...")

	for {
		select {
		// handle logrotate request from daemon
		case <-hup:
			log.Println("[##] signal: handling SIGHUP")
			if err := logFile.Close(); err != nil {
				log.Printf("[##] error closing logfile: %s\n", err)
			}
			if logFile, err = reOpenLog(); err != nil {
				return err
			}

		// exit on interrupt
		case <-ctx.Done():
			log.Println("[##] signal: handling SIGINT/SIGTERM")
			if err := logFile.Close(); err != nil {
				log.Printf("[##] error in cleanup: %s\n", err)
			}
			return nil
		}
	}
}

func reOpenLog() (*os.File, error) {
	if dir := filepath.Dir(logFileName); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating log directory %s: %w", dir, err)
		}
	}

	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", logFileName, err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	return f, nil
}

// buildVersion reports the VCS revision stamped into the binary by the Go
// toolchain, so a running bot can be traced back to a commit.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision == "" {
		return info.GoVersion
	}
	return fmt.Sprintf("%s%s %s", revision[:min(len(revision), 12)], dirty, info.GoVersion)
}
