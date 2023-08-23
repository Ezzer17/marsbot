package main

import (
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v3"
)

//go:embed message_template.tpl
var messageTemplateString string

var loginsConfig = map[string]string{
	"алиса": "alice8080",
	"егор":  "ivanusernam",
}

func getActivePlayerName(db *sql.DB) (string, error) {
	type Player struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type Game struct {
		ActivePlayer string   `json:"activePlayer"`
		Players      []Player `json:"players"`
	}

	row := db.QueryRow("select game from games order by created_at desc limit 1")
	var gameDesc string
	err := row.Scan(&gameDesc)
	if err != nil {
		return "", err
	}
	var game Game
	if err := json.Unmarshal([]byte(gameDesc), &game); err != nil {
		return "", err
	}
	activePlayerID := game.ActivePlayer
	for _, player := range game.Players {
		if player.ID == activePlayerID {
			return player.Name, nil
		}
	}
	return "", fmt.Errorf("failed to find player by id")

}
func main() {

	database := flag.String("database", "", "database file")
	chatID := flag.Int("chat", 0, "telegram chat id")

	db, err := sql.Open("sqlite3", *database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	messageTemplate, err := template.New("").Parse(messageTemplateString)
	if err != nil {
		log.Fatal(err)
	}

	activePlayer, err := getActivePlayerName(db)
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

	for range time.Tick(time.Second) {
		newActivePlayer, err := getActivePlayerName(db)
		if err != nil {
			log.Fatal(err)
		}
		if newActivePlayer != activePlayer {
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
		}

	}

}
