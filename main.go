package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	tele "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed message_template.tpl
var messageTemplateString string

var messageTemplate *template.Template
var loginConfig map[string]string
var domainWhitelist []string

type Config struct {
	Token           string            `yaml:"token"`
	Database        string            `yaml:"database"`
	LoginConfig     map[string]string `yaml:"login_config"`
	DomainWhitelist []string          `yaml:"allowed_domains"`
}

var spectatorIdRegex = regexp.MustCompile(`^s[a-f0-9]{12}$`)

func ParseMarsURL(rawURL string) (*MarsUrl, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	spectatorIDs, ok := parsedURL.Query()["id"]
	if !ok || len(spectatorIDs) == 0 {
		return nil, fmt.Errorf("Player ID is missing in URL")
	}
	spectatorID := spectatorIDs[0]

	if !spectatorIdRegex.MatchString(spectatorID) {
		return nil, fmt.Errorf("ID has invalid format, please provide spectator link!")
	}
	for _, domain := range domainWhitelist {
		if domain == parsedURL.Host {
			return &MarsUrl{
				Proto:       parsedURL.Scheme,
				MarsDomain:  parsedURL.Host,
				SpectatorID: spectatorID,
			}, nil
		}
	}
	return nil, fmt.Errorf("This game URL is not allowed!")
}

func main() {

	configFile := flag.String("config", "config.yaml", "config file")
	flag.Parse()

	yamlFile, err := os.Open(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer yamlFile.Close()
	var config Config
	buf := bytes.NewBuffer([]byte{})
	buf.ReadFrom(yamlFile)
	err = yaml.Unmarshal(buf.Bytes(), &config)
	if err != nil {
		log.Fatal(err)
	}
	loginConfig = config.LoginConfig
	domainWhitelist = config.DomainWhitelist

	db, err := gorm.Open(sqlite.Open(config.Database), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&MarsGame{})
	if err != nil {
		log.Fatal(err)
	}

	messageTemplate, _ = template.New("").Parse(messageTemplateString)
	tgbot, err := tele.NewBot(tele.Settings{
		Token: config.Token,
	})
	if err != nil {
		log.Fatal(err)
	}
	p := Poller{
		client: &http.Client{Timeout: 10 * time.Second},
		bot:    tgbot,
		db:     db,
	}

	gamesNumber, err := p.WatchAll()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bot started! Watching %d games", gamesNumber)
	tgbot.Handle("/watch", func(ctx tele.Context) error {
		payload := ctx.Message().Payload
		marsUrl, err := ParseMarsURL(payload)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("Invalid url: %s", payload))
		}
		game := MarsGame{
			Proto:       marsUrl.Proto,
			MarsDomain:  marsUrl.MarsDomain,
			SpectatorID: marsUrl.SpectatorID,
			ChatID:      ctx.Chat().ID,
		}

		go p.WatchUrl(game)
		return ctx.Reply(fmt.Sprintf("Started watching url `%s`!", game.SpectatorAPIURL()), tele.ModeMarkdown)

	})
	tgbot.Start()

}
