package marsapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Player struct {
	IsActive        bool   `json:"isActive"`
	NeedsToDraft    bool   `json:"needsToDraft"`
	NeedsToResearch bool   `json:"needsToResearch"`
	Name            string `json:"name"`
}
type Game struct {
	Phase       string `json:"phase"`
	SpectatorID string `json:"spectatorId"`
}
type GameState struct {
	Players    []Player `json:"players"`
	Game       Game     `json:"game"`
	ThisPlayer Player   `json:"thisPlayer"`
}

type Client struct {
	c *http.Client
}

func NewClient() *Client {
	return &Client{
		c: http.DefaultClient,
	}
}
func (c *Client) FetchGameState(url string) (*GameState, error) {
	resp, err := c.c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get active player name: %s", resp.Status)
	}

	var game GameState
	if err := json.NewDecoder(resp.Body).Decode(&game); err != nil {
		return nil, err
	}
	return &game, nil
}
