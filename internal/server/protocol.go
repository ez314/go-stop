package server

import "gostop/internal/game"

// ClientMsg is the envelope for messages sent from browser clients to the server.
type ClientMsg struct {
	Type         string       `json:"type"`
	Name         string       `json:"name,omitempty"`
	Token        string       `json:"token,omitempty"`
	PlayerCount  int          `json:"playerCount,omitempty"`
	IncludeBonus *bool        `json:"includeBonus,omitempty"`
	Action       *game.Action `json:"action,omitempty"`
	TargetSeat   int          `json:"targetSeat,omitempty"`
}

// LobbyPlayer describes a seated player in the lobby.
type LobbyPlayer struct {
	Seat        int    `json:"seat"`
	Name        string `json:"name"`
	Ready       bool   `json:"ready"`
	Connected   bool   `json:"connected"`
	IsBot       bool   `json:"isBot,omitempty"`
	TotalPoints int    `json:"totalPoints"`
}

// LobbyView is broadcast to all clients when lobby or connection state changes.
type LobbyView struct {
	Players  []LobbyPlayer   `json:"players"`
	Config   game.RuleConfig `json:"config"`
	InGame   bool            `json:"inGame"`
	HostSeat int             `json:"hostSeat"`
	LanURL   string          `json:"lanUrl,omitempty"`
	LanURLs  []string        `json:"lanUrls,omitempty"`
}

// ServerMsg is the envelope for messages sent from the server to browser clients.
type ServerMsg struct {
	Type       string           `json:"type"`
	Token      string           `json:"token,omitempty"`
	Seat       int              `json:"seat"`
	Spectating bool             `json:"spectating,omitempty"`
	CanUndo    bool             `json:"canUndo,omitempty"`
	Lobby      *LobbyView       `json:"lobby,omitempty"`
	Game       *game.PlayerView `json:"game,omitempty"`
	Events     []game.Event     `json:"events,omitempty"`
	Error      string           `json:"error,omitempty"`
}
