package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

type Poller struct {
	client *http.Client
	bot    *tele.Bot
	db     *gorm.DB
}

func (p *Poller) getActivePlayerName(marsGame *MarsGame) (string, error) {
	type Player struct {
		ID              string `json:"id"`
		IsActive        bool   `json:"isActive"`
		NeedsToDraft    bool   `json:"needsToDraft"`
		NeedsToResearch bool   `json:"needsToResearch"`
		Name            string `json:"name"`
	}
	type Game struct {
		ActivePlayer string   `json:"activePlayer"`
		Players      []Player `json:"players"`
	}
	res, err := p.client.Get(marsGame.SpectatorAPIURL())

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

	draftingPlayers := []string{}
	activePlayers := []string{}
	researchingPlayers := []string{}
	for _, player := range game.Players {
		if player.IsActive {
			activePlayers = append(activePlayers, player.Name)
		}
		if player.NeedsToDraft {
			draftingPlayers = append(draftingPlayers, player.Name)
		}
		if player.NeedsToResearch {
			researchingPlayers = append(researchingPlayers, player.Name)
		}
	}

	if len(draftingPlayers) == 1 {
		return draftingPlayers[0], nil
	}
	if len(researchingPlayers) == 1 {
		return researchingPlayers[0], nil
	}
	if len(activePlayers) != 0 {
		return activePlayers[0], nil
	}
	return "", fmt.Errorf("failed to find active player")

}

func (p *Poller) WatchAll() (int, error) {

	games := []MarsGame{}

	res := p.db.Find(&games)

	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return 0, res.Error
	}
	for _, game := range games {
		go p.WatchUrl(&game)
	}
	return len(games), nil
}
func (p *Poller) WatchUrl(game *MarsGame) {
	log.Printf("Watching %s", game.SpectatorAPIURL())
	activePlayer, err := p.getActivePlayerName(game)
	if err != nil {
		log.Printf("Faied to get initial active player: %s", err)
		_, err = p.bot.Send(tele.ChatID(game.ChatID), fmt.Sprintf("Failed to watch url %s: %s", game.SpectatorAPIURL(), err))
		if err != nil {
			log.Printf("Message send failed; %v", err)
		}
	}
	res := p.db.Save(game)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		log.Printf("Faied to save game: %s", res.Error)
		_, err = p.bot.Send(tele.ChatID(game.ChatID), fmt.Sprintf("Failed to save game %s: %s", game.SpectatorAPIURL(), res.Error))
		if err != nil {
			log.Printf("Message send failed; %v", err)
		}
	}
	_, err = p.bot.Send(tele.ChatID(game.ChatID), fmt.Sprintf("Started watching url %s", game.SpectatorAPIURL()))
	if err != nil {
		log.Printf("Message send failed; %v", err)
	}

	for range time.Tick(time.Second) {
		newActivePlayer, err := p.getActivePlayerName(game)
		if err != nil {
			log.Printf("Faied to get new active player: %s", err)
			continue
		}
		if newActivePlayer != activePlayer {
			log.Printf("Active player changed to %s", newActivePlayer)
			messageTextBuf := bytes.NewBuffer([]byte{})
			login, ok := loginConfig[strings.TrimSpace(strings.ToLower(newActivePlayer))]
			if !ok {
				log.Printf("Error: unknown name %s", newActivePlayer)
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
			_, err = p.bot.Send(tele.ChatID(game.ChatID), messageTextBuf.String(), &tele.SendOptions{
				ParseMode: tele.ModeHTML,
			})
			if err != nil {
				log.Printf("Message send failed; %v", err)
			}
			activePlayer = newActivePlayer
		}

	}

}
