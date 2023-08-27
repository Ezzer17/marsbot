package main

import (
	"net/url"

	"gorm.io/gorm"
)

type MarsUrl struct {
	Proto       string
	MarsDomain  string
	SpectatorID string
}

type MarsGame struct {
	gorm.Model
	Proto       string
	MarsDomain  string
	SpectatorID string
	ChatID      int64
}

func (m *MarsGame) SpectatorAPIURL() string {
	url := url.URL{
		Scheme:   m.Proto,
		Host:     m.MarsDomain,
		Path:     "/api/spectator",
		RawQuery: "id=" + m.SpectatorID,
	}
	return url.String()
}
