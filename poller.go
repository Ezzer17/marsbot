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

type gameState struct {
	isFinished   bool
	activePlayer string
}

func (p *Poller) getGameState(marsGame *MarsGame) (*gameState, error) {
	type Player struct {
		ID              string `json:"id"`
		IsActive        bool   `json:"isActive"`
		NeedsToDraft    bool   `json:"needsToDraft"`
		NeedsToResearch bool   `json:"needsToResearch"`
		Name            string `json:"name"`
	}
	type Game struct {
		Phase string `json:"phase"`
	}
	type GameState struct {
		ActivePlayer string   `json:"activePlayer"`
		Players      []Player `json:"players"`
		Game         Game     `json:"game"`
	}
	res, err := p.client.Get(marsGame.SpectatorAPIURL())

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get active player name: %s", res.Status)
	}

	var game GameState
	if err := json.NewDecoder(res.Body).Decode(&game); err != nil {
		return nil, err
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
	if game.Game.Phase == "end" {
		return &gameState{
			isFinished:   true,
			activePlayer: "",
		}, nil
	}

	if len(draftingPlayers) == 1 {
		return &gameState{
			activePlayer: draftingPlayers[0],
			isFinished:   false,
		}, nil
	}
	if len(researchingPlayers) == 1 {
		return &gameState{
			activePlayer: researchingPlayers[0],
			isFinished:   false,
		}, nil
	}
	if len(activePlayers) != 0 {
		return &gameState{
			activePlayer: activePlayers[0],
			isFinished:   false,
		}, nil
	}
	return nil, fmt.Errorf("failed to find active player")

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

func (p *Poller) Reply(chatId int64, msg string) {
	_, err := p.bot.Send(tele.ChatID(chatId), msg, &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
	if err != nil {
		log.Printf("Message send failed; %v", err)
	}
}

func (p *Poller) WatchUrl(game *MarsGame) {
	log.Printf("Watching %s", game.SpectatorAPIURL())
	gameState, err := p.getGameState(game)
	if err != nil {
		log.Printf("Faied to get initial active player: %s", err)
		p.Reply(game.ChatID, fmt.Sprintf("Failed to watch url `%s`: %s", game.SpectatorAPIURL(), err))
		return
	}
	activePlayer := gameState.activePlayer
	res := p.db.Save(game)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		log.Printf("Faied to save game: %s", res.Error)
		p.Reply(game.ChatID, fmt.Sprintf("Failed to save game `%s`: %s", game.SpectatorAPIURL(), err))
	}
	p.Reply(game.ChatID, fmt.Sprintf("Started watching game `%d`", game.ID))

	for range time.Tick(time.Second) {
		newGameState, err := p.getGameState(game)
		if err != nil {
			log.Printf("Faied to get new active player: %s", err)
			continue
		}
		if activePlayer != newGameState.activePlayer {
			log.Printf("Active player changed to %s", newGameState.activePlayer)
			messageTextBuf := bytes.NewBuffer([]byte{})
			login, ok := loginConfig[strings.TrimSpace(strings.ToLower(newGameState.activePlayer))]
			if !ok {
				log.Printf("Error: unknown name %s", newGameState.activePlayer)
			}
			messageTemplate.Execute(
				messageTextBuf,
				struct {
					TelegramLogin string
					Name          string
				}{
					TelegramLogin: login,
					Name:          newGameState.activePlayer,
				})
			_, err = p.bot.Send(tele.ChatID(game.ChatID), messageTextBuf.String(), &tele.SendOptions{
				ParseMode: tele.ModeHTML,
			})
			if err != nil {
				log.Printf("Message send failed; %v", err)
			}
			activePlayer = newGameState.activePlayer
		}
		if gameState.isFinished {
			log.Printf("Finished %d", game.ID)
			p.Reply(game.ChatID, fmt.Sprintf("Game `%d` finished!!", game.ID))
			p.db.Delete(game)
			break
		}

	}

}
