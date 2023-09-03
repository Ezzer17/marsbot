package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	tele "gopkg.in/telebot.v3"
	"gorm.io/gorm"

	"github.com/ezzer17/marsbot/marsapi"
)

type Watcher struct {
	client *marsapi.Client
	bot    *tele.Bot
	db     *gorm.DB
}

type gameState struct {
	isFinished    bool
	waitedPlayers map[string]struct{}
}

func (p *Watcher) GetSubscribers(chatID int64) ([]*Subscriber, error) {
	subscribers := []*Subscriber{}
	res := p.db.Where(&Subscriber{ChatID: chatID}).Find(&subscribers)
	return subscribers, res.Error
}
func (p *Watcher) RemoveSubscription(chatID int64, playerID string) error {
	sub := &Subscriber{ChatID: chatID, PlayerID: playerID}
	res := p.db.Unscoped().Where(sub).Delete(sub)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("Subscription not found")
	}
	return nil
}
func (p *Watcher) AddSubscription(chatId int64, marsUrl marsapi.MarsPlayerURL) (*Subscriber, error) {
	gameState, err := p.client.FetchGameState(marsUrl.AsString())
	if err != nil {
		return nil, err
	}
	gamePoll := &MarsGame{
		Proto:       marsUrl.Proto,
		Domain:      marsUrl.MarsDomain,
		SpectatorID: gameState.Game.SpectatorID,
	}
	res := p.db.Where(&gamePoll).First(&gamePoll)
	if res.Error == gorm.ErrRecordNotFound {
		if saveRes := p.db.Save(&gamePoll); saveRes.Error != nil {
			return nil, saveRes.Error
		}
		go p.WatchGame(*gamePoll)
	} else if res.Error != nil {
		return nil, res.Error
	}
	if gamePoll.IsFinished {
		return nil, fmt.Errorf("Game %d finished", gamePoll.ID)
	}
	subscription := &Subscriber{
		PlayerID: marsUrl.ParticipantID,
		ChatID:   chatId,
		Name:     gameState.ThisPlayer.Name,
		MarsGame: gamePoll,
	}
	res = p.db.Save(&subscription)
	if res.Error != nil {
		return nil, res.Error
	}
	return subscription, nil
}
func (p *Watcher) getGameState(marsGame MarsGame) (*gameState, error) {
	spectatorURL := marsGame.SpectatorURL()
	game, err := p.client.FetchGameState(spectatorURL.AsString())
	if err != nil {
		return nil, err
	}
	draftingPlayers := map[string]struct{}{}
	activePlayers := map[string]struct{}{}
	researchingPlayers := map[string]struct{}{}
	for _, player := range game.Players {
		if player.IsActive {
			activePlayers[player.Name] = struct{}{}
		}
		if player.NeedsToDraft {
			draftingPlayers[player.Name] = struct{}{}
		}
		if player.NeedsToResearch {
			researchingPlayers[player.Name] = struct{}{}
		}
	}
	state := &gameState{
		isFinished:    false,
		waitedPlayers: map[string]struct{}{},
	}
	if game.Game.Phase == "end" {
		state.isFinished = true
		return state, nil
	}

	if game.Game.Phase == "drafting" {
		state.waitedPlayers = draftingPlayers
		return state, nil
	}
	if game.Game.Phase == "research" {
		state.waitedPlayers = researchingPlayers
		return state, nil
	}
	state.waitedPlayers = activePlayers
	return state, nil

}

func (p *Watcher) WatchAll() (int, error) {
	games := []MarsGame{}
	res := p.db.Where("is_finished = false").Find(&games)

	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return 0, res.Error
	}
	for _, game := range games {
		go p.WatchGame(game)
	}
	return len(games), nil
}

func (p *Watcher) reply(chatId int64, msg string) {
	_, err := p.bot.Send(tele.ChatID(chatId), msg, &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
	if err != nil {
		log.Printf("Message send failed; %v", err)
	}
}

func (p *Watcher) WatchGame(game MarsGame) {
	log.Printf("Watching game %d", game.ID)
	waitedPlayers := map[string]struct{}{}

	for range time.Tick(time.Second) {
		newGameState, err := p.getGameState(game)
		if err != nil {
			log.Printf("Faied to get new active player: %s", err)
			continue
		}

		if !reflect.DeepEqual(waitedPlayers, newGameState.waitedPlayers) && len(newGameState.waitedPlayers) > 0 && len(waitedPlayers) > 0 {
			log.Printf("Active players changed to %v", newGameState.waitedPlayers)
			subscribers := []Subscriber{}
			for player := range newGameState.waitedPlayers {
				if _, ok := waitedPlayers[player]; !ok {
					p.db.Where(&Subscriber{MarsGameID: game.ID, Name: player}).Joins("MarsGame").Find(&subscribers)
					for _, subscriber := range subscribers {
						playerURL := subscriber.PlayerURL()
						p.reply(subscriber.ChatID, fmt.Sprintf("%s, your turn in game %d!\n%s", subscriber.Name, game.ID, playerURL.AsHumanLink()))
					}
				}
			}
		}
		waitedPlayers = newGameState.waitedPlayers
		if newGameState.isFinished {
			log.Printf("Game %d finished", game.ID)
			subscribers := []Subscriber{}
			p.db.Where(&Subscriber{MarsGameID: game.ID}).Find(&subscribers)
			for _, subscriber := range subscribers {
				p.reply(subscriber.ChatID, fmt.Sprintf("Game %d finished!!", game.ID))
			}
			game.IsFinished = true
			if res := p.db.Save(&game); res.Error != nil {
				log.Printf("Faied to save finished game: %v", res.Error)
			}
			break
		}

	}

}
