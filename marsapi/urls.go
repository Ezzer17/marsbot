package marsapi

import (
	"net/url"
)

type MarsUrl struct {
	Proto         string
	MarsDomain    string
	ParticipantID string
}
type MarsPlayerURL MarsUrl

func (m *MarsPlayerURL) AsString() string {
	url := url.URL{
		Scheme:   m.Proto,
		Host:     m.MarsDomain,
		Path:     "/api/player",
		RawQuery: "id=" + m.ParticipantID,
	}
	return url.String()
}

type MarsSpectatorURL MarsUrl

func (m *MarsSpectatorURL) AsString() string {
	url := url.URL{
		Scheme:   m.Proto,
		Host:     m.MarsDomain,
		Path:     "/api/spectator",
		RawQuery: "id=" + m.ParticipantID,
	}
	return url.String()
}
