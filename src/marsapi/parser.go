package marsapi

import (
	"fmt"
	"net/url"
	"regexp"
)

var spectatorIdRegex = regexp.MustCompile(`^s[a-f0-9]+$`)
var playerIdRegex = regexp.MustCompile(`^p[a-f0-9]+$`)

type Parser struct {
	allowedDomains []string
}

func IsPlayerID(id string) bool {
	return playerIdRegex.MatchString(id)
}

func IsSpectatorID(id string) bool {
	return spectatorIdRegex.MatchString(id)
}
func NewParser(allowedDomains []string) *Parser {
	return &Parser{
		allowedDomains: allowedDomains,
	}
}

func (p *Parser) parseMarsURLWithID(rawURL string) (*MarsUrl, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	IDs, ok := parsedURL.Query()["id"]
	if !ok || len(IDs) == 0 {
		return nil, fmt.Errorf("Player ID is missing in URL")
	}

	for _, domain := range p.allowedDomains {
		if domain == parsedURL.Host {
			return &MarsUrl{
				Proto:         parsedURL.Scheme,
				MarsDomain:    parsedURL.Host,
				ParticipantID: IDs[0],
			}, nil
		}
	}
	return nil, fmt.Errorf("This game URL is not allowed!")
}
func (p *Parser) ParsePlayerURL(rawURL string) (*MarsPlayerURL, error) {
	parsedURL, err := p.parseMarsURLWithID(rawURL)
	if err != nil {
		return nil, err
	}
	if !IsPlayerID(parsedURL.ParticipantID) {
		return nil, fmt.Errorf("ID has invalid format, please provide spectator link!")
	}
	playerURL := MarsPlayerURL(*parsedURL)
	return &playerURL, nil
}
