package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	tele "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ezzer17/marsbot/marsapi"
)

type Config struct {
	Token           string            `yaml:"token"`
	Database        string            `yaml:"database"`
	LoginConfig     map[string]string `yaml:"login_config"`
	DomainWhitelist []string          `yaml:"allowed_domains"`
}

func autoMigrate(db *gorm.DB) (err error) {
	err = db.AutoMigrate(&MarsGame{})
	if err != nil {
		return
	}

	err = db.AutoMigrate(&Subscriber{})
	if err != nil {
		return
	}
	return
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

	parser := marsapi.NewParser(config.DomainWhitelist)

	db, err := gorm.Open(sqlite.Open(config.Database), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = autoMigrate(db)
	if err != nil {
		log.Fatal(err)
	}

	tgbot, err := tele.NewBot(tele.Settings{
		Token: config.Token,
	})
	if err != nil {
		log.Fatal(err)
	}

	client := marsapi.NewClient()
	p := Watcher{
		client: client,
		bot:    tgbot,
		db:     db,
	}

	gamesNumber, err := p.WatchAll()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bot started! Watching %d games", gamesNumber)
	tgbot.Handle("/subscribe", func(ctx tele.Context) error {
		payload := ctx.Message().Payload
		marsUrl, err := parser.ParsePlayerURL(payload)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("Invalid url: `%s`! Expected player ID in URL", payload), tele.ModeMarkdown)
		}
		subscriber, err := p.AddSubscription(ctx.Chat().ID, *marsUrl)

		if err != nil {
			return ctx.Reply(fmt.Sprintf("Failed to subscribe: `%s`", err), tele.ModeMarkdown)
		}
		log.Printf("Successfullty subscribed player %s to game %d!", subscriber.Name, subscriber.MarsGameID)
		return ctx.Reply(fmt.Sprintf("Successfullty subscribed player %s to game %d!", subscriber.Name, subscriber.MarsGameID))
	})
	tgbot.Handle("/subscriptions", func(ctx tele.Context) error {
		subscribers, err := p.GetSubscribers(ctx.Chat().ID)

		if err != nil {
			return ctx.Reply(fmt.Sprintf("Failed to get subscriptions: `%s`", err), tele.ModeMarkdown)
		}
		message := "Your subscriptions:\n"
		for _, subscriber := range subscribers {
			url := subscriber.PlayerURL()
			message += fmt.Sprintf("`%-10s`: game %02d %s\n", subscriber.Name, subscriber.MarsGameID, url.AsHumanLink())
		}
		return ctx.Reply(message, tele.ModeMarkdown)
	})
	tgbot.Handle("/unsubscribe", func(ctx tele.Context) error {
		playerID := ctx.Message().Payload
		if !marsapi.IsPlayerID(playerID) {
			return ctx.Reply(fmt.Sprintf("Invalid player id: `%s`", playerID), tele.ModeMarkdown)
		}
		err := p.RemoveSubscription(ctx.Chat().ID, playerID)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("Failed to delete subscription: `%s`", err), tele.ModeMarkdown)
		}
		log.Printf("Successfullty unsubscribed player %s from chat %d!", playerID, ctx.Chat().ID)
		return ctx.Reply(fmt.Sprintf("Successfully unsubscribed player id `%s` from chat %d!", playerID, ctx.Chat().ID))
	})
	tgbot.Start()

}
