package main

import (
	"github.com/ezzer17/marsbot/marsapi"
	"gorm.io/gorm"
)

type MarsGame struct {
	gorm.Model
	Proto       string
	Domain      string `gorm:"index:idx_mars_game_spectator_domain,unique"`
	SpectatorID string `gorm:"index:idx_mars_game_spectator_domain,unique"`
	IsFinished  bool

	Subscribers []Subscriber
}

func (g *MarsGame) SpectatorURL() marsapi.MarsSpectatorURL {
	return marsapi.MarsSpectatorURL{
		Proto:         g.Proto,
		MarsDomain:    g.Domain,
		ParticipantID: g.SpectatorID,
	}
}

type Subscriber struct {
	gorm.Model
	PlayerID string `gorm:"index:idx_subscription_chat,unique"`
	ChatID   int64  `gorm:"index:idx_subscription_chat,unique"`
	Name     string

	MarsGameID uint
	MarsGame   *MarsGame
}

func (s *Subscriber) PlayerURL() marsapi.MarsPlayerURL {
	return marsapi.MarsPlayerURL{
		Proto:         s.MarsGame.Proto,
		MarsDomain:    s.MarsGame.Domain,
		ParticipantID: s.PlayerID,
	}
}
