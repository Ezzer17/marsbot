package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

//go:embed message_template.tpl
var messageTemplateString string

type Poller struct {
	client  *http.Client
	gameURL string
}

type Config struct {
	Token       string            `yaml:"token"`
	ChatID      int64             `yaml:"chat_id"`
	GameURL     string            `yaml:"game_url"`
	LoginConfig map[string]string `yaml:"login_config"`
}

func (p *Poller) GetActivePlayerName() (string, error) {
	type Player struct {
		ID       string `json:"id"`
		IsActive bool   `json:"isActive"`
		Name     string `json:"name"`
	}
	type Game struct {
		ActivePlayer string   `json:"activePlayer"`
		Players      []Player `json:"players"`
	}
	res, err := p.client.Get(p.gameURL)

	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get active player name: %s", res.Status)
	}

	var game Game
	if err := json.NewDecoder(res.Body).Decode(&game); err != nil {
		return "", err
	}

	for _, player := range game.Players {
		if player.IsActive {
			return player.Name, nil
		}
	}
	return "", fmt.Errorf("failed to find player by id")

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

	p := Poller{
		client:  &http.Client{Timeout: 10 * time.Second},
		gameURL: config.GameURL,
	}

	messageTemplate, err := template.New("").Parse(messageTemplateString)
	if err != nil {
		log.Fatal(err)
	}

	tgbot, err := tele.NewBot(tele.Settings{
		Token: config.Token,
	})
	if err != nil {
		log.Fatal(err)
	}
	chat := tele.ChatID(config.ChatID)

	activePlayer := ""

	for range time.Tick(time.Second) {
		newActivePlayer, err := p.GetActivePlayerName()
		if err != nil {
			log.Print(err)
		}
		if newActivePlayer != activePlayer  {
			log.Printf("Active player changed to %s", newActivePlayer)
			messageTextBuf := bytes.NewBuffer([]byte{})
			login, ok := config.LoginConfig[strings.TrimSpace(strings.ToLower(newActivePlayer))]
			if !ok {
				log.Printf("Unknown name %s", newActivePlayer)
			}
			err := messageTemplate.Execute(
				messageTextBuf,
				struct {
					TelegramLogin string
					Name          string
				}{
					TelegramLogin: login,
					Name:          newActivePlayer,
				})
			if err != nil {
				log.Print(err)
			}
			_, err = tgbot.Send(chat, messageTextBuf.String(), &tele.SendOptions{
				ParseMode: tele.ModeHTML,
			})
			if err != nil {
				log.Printf("Message send failed; %v", err)
			}
			activePlayer = newActivePlayer
		}

	}

}
