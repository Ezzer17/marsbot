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
)

//go:embed message_template.tpl
var messageTemplateString string

var loginsConfig = map[string]string{
	"алиса": "alice8080",
	"егор":  "ivanusernam",
}

type Poller struct {
	client *http.Client
	link   string
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
	res, err := p.client.Get(p.link)

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

	url := flag.String("url", "", "player api url")
	chatID := flag.Int("chat", 0, "telegram chat id")
	flag.Parse()

	p := Poller{
		client: &http.Client{Timeout: 10 * time.Second},
		link:   *url,
	}

	messageTemplate, err := template.New("").Parse(messageTemplateString)
	if err != nil {
		log.Fatal(err)
	}

	tgbot, err := tele.NewBot(tele.Settings{
		Token: os.Getenv("TELEGRAM_TOKEN"),
	})
	if err != nil {
		log.Fatal(err)
	}
	chat := tele.ChatID(*chatID)

	activePlayer := ""

	for range time.Tick(time.Second) {
		newActivePlayer, err := p.GetActivePlayerName()
		if err != nil {
			log.Fatal(err)
		}
		if newActivePlayer != activePlayer && activePlayer != "" {
			log.Printf("Active player changed to %s", newActivePlayer)
			messageTextBuf := bytes.NewBuffer([]byte{})
			login, ok := loginsConfig[strings.TrimSpace(strings.ToLower(newActivePlayer))]
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
				log.Fatal(err)
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
