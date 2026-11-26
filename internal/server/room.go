package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"gostop/internal/game"
	"log"
	mrand "math/rand"
	"strings"
	"sync"
	"time"
)

type seatPlayer struct {
	token       string
	name        string
	ready       bool
	isBot       bool
	totalPoints int
	client      *Client
}

// Room manages a single LAN Go-Stop table, player seats, reconnections, bots, undo, and game state.
type Room struct {
	mu          sync.Mutex
	clients     map[*Client]bool
	seats       []*seatPlayer
	specs       map[*Client]bool
	config      game.RuleConfig
	game        *game.GameState
	nextDealer  int // previous round's winner goes first next round
	gameGen     int // incremented per round / undo to cancel stale bot timers
	undoSnap    *game.GameState
	undoSeat    int
	undoUsed    map[int]bool
	botDelay    time.Duration
	rng         *mrand.Rand
	lanURL      string
	lanURLs     []string
}

// NewRoom creates a new LAN Go-Stop room with default 2-player Matgo rules.
func NewRoom(lanURL string, lanURLs []string) *Room {
	return &Room{
		clients:  make(map[*Client]bool),
		specs:    make(map[*Client]bool),
		config:   game.DefaultRules(2),
		undoSeat: -1,
		undoUsed: make(map[int]bool),
		botDelay: 1200 * time.Millisecond,
		rng:      mrand.New(mrand.NewSource(time.Now().UnixNano())),
		lanURL:   lanURL,
		lanURLs:  lanURLs,
	}
}

func generateToken() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HandleDisconnect marks a player's seat as disconnected (preserving their seat for reconnect).
func (r *Room) HandleDisconnect(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.clients, c)
	delete(r.specs, c)
	changed := false
	for _, sp := range r.seats {
		if sp.client == c {
			sp.client = nil
			changed = true
		}
	}
	if changed {
		r.broadcastLobbyLocked()
	}
}

// HandleMessage processes an incoming WebSocket message from a client.
func (r *Room) HandleMessage(c *Client, msg ClientMsg) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch msg.Type {
	case "join":
		r.handleJoinLocked(c, msg.Name, msg.Token, false)
	case "reconnect":
		r.handleJoinLocked(c, msg.Name, msg.Token, true)
	case "set_rules":
		r.handleSetRulesLocked(c, msg)
	case "ready":
		r.handleReadyLocked(c)
	case "start_game":
		r.handleStartGameLocked(c)
	case "next_round":
		r.handleNextRoundLocked(c)
	case "return_lobby":
		r.handleReturnLobbyLocked(c)
	case "restart_round":
		r.handleRestartRoundLocked(c)
	case "add_bot":
		r.handleAddBotLocked(c)
	case "undo":
		r.handleUndoLocked(c)
	case "action":
		if msg.Action == nil {
			c.Send(ServerMsg{Type: "error", Error: "missing action"})
			return
		}
		r.handleActionLocked(c, *msg.Action)
	case "leave":
		r.handleLeaveLocked(c)
	case "kick":
		r.handleKickLocked(c, msg.TargetSeat)
	case "reclaim":
		r.handleReclaimLocked(c, msg.TargetSeat)
	default:
		c.Send(ServerMsg{Type: "error", Error: "unknown message type: " + msg.Type})
	}
}

func (r *Room) findSeatByClient(c *Client) int {
	for i, sp := range r.seats {
		if sp.client == c {
			return i
		}
	}
	return -1
}

func (r *Room) handleJoinLocked(c *Client, rawName, token string, reconnectOnly bool) {
	// 1. Try reconnect by token
	if token != "" {
		for i, sp := range r.seats {
			if sp.token == token {
				if sp.client != nil && sp.client != c {
					sp.client.Close()
				}
				delete(r.specs, c)
				sp.client = c
				c.Send(ServerMsg{Type: "welcome", Token: sp.token, Seat: i})
				r.broadcastLobbyLocked()
				if r.game != nil {
					view := r.game.ViewFor(i)
					c.Send(ServerMsg{Type: "state", Game: &view})
				}
				return
			}
		}
		// Stale/expired token on auto-reconnect: clear it without auto-seating
		if reconnectOnly {
			c.Send(ServerMsg{Type: "welcome", Token: "", Seat: -1})
			return
		}
	}

	name := strings.TrimSpace(rawName)
	if name == "" {
		name = fmt.Sprintf("Player %d", len(r.seats)+1)
	}
	if len(name) > 16 {
		name = name[:16]
	}

	// 2. If already seated, update name
	if idx := r.findSeatByClient(c); idx >= 0 {
		r.seats[idx].name = name
		r.broadcastLobbyLocked()
		return
	}

	// 3. Seat if room has space and game hasn't started
	if r.game == nil && len(r.seats) < r.config.PlayerCount {
		delete(r.specs, c)
		tok := generateToken()
		seat := len(r.seats)
		r.seats = append(r.seats, &seatPlayer{
			token:  tok,
			name:   name,
			client: c,
		})
		c.Send(ServerMsg{Type: "welcome", Token: tok, Seat: seat})
		r.broadcastLobbyLocked()
		return
	}

	// 4. Otherwise join as spectator
	r.specs[c] = true
	c.Send(ServerMsg{Type: "welcome", Seat: -1, Spectating: true})
	c.Send(ServerMsg{Type: "error", Error: fmt.Sprintf("Table is full (%d/%d) — watching as spectator", len(r.seats), r.config.PlayerCount)})
	r.broadcastLobbyLocked()
	if r.game != nil {
		view := r.game.ViewFor(-1)
		c.Send(ServerMsg{Type: "state", Game: &view})
	}
}

func (r *Room) handleKickLocked(c *Client, targetSeat int) {
	if r.findSeatByClient(c) != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can remove a player"})
		return
	}
	if r.game != nil || targetSeat <= 0 || targetSeat >= len(r.seats) {
		return
	}
	kicked := r.seats[targetSeat]
	if kicked.client != nil {
		kicked.client.Send(ServerMsg{Type: "welcome", Token: "", Seat: -1})
	}
	r.seats = append(r.seats[:targetSeat], r.seats[targetSeat+1:]...)
	for i, sp := range r.seats {
		if sp.client != nil {
			sp.client.Send(ServerMsg{Type: "welcome", Token: sp.token, Seat: i})
		}
	}
	r.broadcastLobbyLocked()
}

func (r *Room) handleReclaimLocked(c *Client, targetSeat int) {
	if r.findSeatByClient(c) >= 0 {
		return
	}
	if targetSeat < 0 || targetSeat >= len(r.seats) {
		c.Send(ServerMsg{Type: "error", Error: "seat not found"})
		return
	}
	sp := r.seats[targetSeat]
	if sp.isBot || sp.client != nil {
		c.Send(ServerMsg{Type: "error", Error: "that seat is already active"})
		return
	}
	delete(r.specs, c)
	sp.client = c
	c.Send(ServerMsg{Type: "welcome", Token: sp.token, Seat: targetSeat})
	r.broadcastLobbyLocked()
	if r.game != nil {
		view := r.game.ViewFor(targetSeat)
		canUndo := r.undoSnap != nil && r.undoSeat == targetSeat && !r.undoUsed[targetSeat] && r.game.Phase != game.PhaseRoundOver
		c.Send(ServerMsg{Type: "state", Game: &view, CanUndo: canUndo})
	}
}

func (r *Room) handleLeaveLocked(c *Client) {
	idx := r.findSeatByClient(c)
	if idx < 0 {
		return
	}
	if r.game != nil {
		c.Send(ServerMsg{Type: "error", Error: "cannot leave while a round is in progress"})
		return
	}
	r.seats = append(r.seats[:idx], r.seats[idx+1:]...)
	c.Send(ServerMsg{Type: "welcome", Token: "", Seat: -1})
	for i, sp := range r.seats {
		if sp.client != nil {
			sp.client.Send(ServerMsg{Type: "welcome", Token: sp.token, Seat: i})
		}
	}
	r.broadcastLobbyLocked()
}

func (r *Room) handleSetRulesLocked(c *Client, msg ClientMsg) {
	if r.findSeatByClient(c) != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can change rules"})
		return
	}
	if r.game != nil {
		c.Send(ServerMsg{Type: "error", Error: "cannot change rules during a round"})
		return
	}
	if msg.PlayerCount == 2 || msg.PlayerCount == 3 || msg.PlayerCount == 4 {
		if msg.PlayerCount < len(r.seats) {
			c.Send(ServerMsg{Type: "error", Error: fmt.Sprintf("Remove %d player(s) before switching to %dP", len(r.seats)-msg.PlayerCount, msg.PlayerCount)})
			return
		}
		bonus := r.config.IncludeBonusCards
		r.config = game.DefaultRules(msg.PlayerCount)
		r.config.IncludeBonusCards = bonus
	}
	if msg.IncludeBonus != nil {
		r.config.IncludeBonusCards = *msg.IncludeBonus
	}
	r.broadcastLobbyLocked()
}

func (r *Room) handleReadyLocked(c *Client) {
	idx := r.findSeatByClient(c)
	if idx < 0 || r.game != nil {
		return
	}
	r.seats[idx].ready = !r.seats[idx].ready
	r.broadcastLobbyLocked()
}

func (r *Room) handleStartGameLocked(c *Client) {
	if r.findSeatByClient(c) != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can start"})
		return
	}
	if r.game != nil {
		return
	}
	if len(r.seats) != r.config.PlayerCount {
		c.Send(ServerMsg{Type: "error", Error: fmt.Sprintf("waiting for %d players", r.config.PlayerCount)})
		return
	}
	r.startNewRoundLocked()
}

func (r *Room) handleNextRoundLocked(c *Client) {
	if r.findSeatByClient(c) < 0 {
		return
	}
	if r.game == nil || r.game.Phase != game.PhaseRoundOver {
		return
	}
	r.startNewRoundLocked()
}

func (r *Room) handleReturnLobbyLocked(c *Client) {
	seat := r.findSeatByClient(c)
	if seat < 0 || r.game == nil {
		return
	}
	// Any seated player can return to lobby after round over; host (seat 0) can return mid-game too.
	if r.game.Phase != game.PhaseRoundOver && seat != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can return to lobby mid-round"})
		return
	}
	r.game = nil
	r.gameGen++
	r.undoSnap = nil
	r.undoSeat = -1
	r.broadcastLobbyLocked()
	for cl := range r.clients {
		cl.Send(ServerMsg{Type: "state", Game: nil})
	}
}

func (r *Room) handleRestartRoundLocked(c *Client) {
	if r.findSeatByClient(c) != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can restart the round"})
		return
	}
	if r.game == nil {
		return
	}
	r.startNewRoundLocked()
}

func (r *Room) handleAddBotLocked(c *Client) {
	if r.findSeatByClient(c) != 0 {
		c.Send(ServerMsg{Type: "error", Error: "only the host can add a bot"})
		return
	}
	if r.game != nil {
		c.Send(ServerMsg{Type: "error", Error: "cannot add a bot during a round"})
		return
	}
	if len(r.seats) >= r.config.PlayerCount {
		c.Send(ServerMsg{Type: "error", Error: "table is already full"})
		return
	}
	botNum := 1
	for _, sp := range r.seats {
		if sp.isBot {
			botNum++
		}
	}
	r.seats = append(r.seats, &seatPlayer{
		token: generateToken(),
		name:  fmt.Sprintf("Bot %d", botNum),
		isBot: true,
	})
	r.broadcastLobbyLocked()
}

func (r *Room) handleUndoLocked(c *Client) {
	seat := r.findSeatByClient(c)
	if seat < 0 || r.game == nil || r.game.Phase == game.PhaseRoundOver {
		return
	}
	if r.undoSnap == nil || r.undoSeat != seat || r.undoUsed[seat] {
		c.Send(ServerMsg{Type: "error", Error: "undo is not available right now"})
		return
	}
	r.game = r.undoSnap.Clone()
	r.gameGen++
	r.undoSnap = nil
	r.undoSeat = -1
	r.undoUsed[seat] = true
	events := []game.Event{{
		Type:      "undo",
		PlayerIdx: seat,
		Message:   fmt.Sprintf("%s undid their last move (무르기)", r.seats[seat].name),
	}}
	r.broadcastGameLocked(events)
	r.scheduleBotTurnLocked()
}

func (r *Room) startNewRoundLocked() {
	names := make([]string, len(r.seats))
	for i, sp := range r.seats {
		names[i] = sp.name
		sp.ready = false
	}
	dealer := r.nextDealer
	if dealer < 0 || dealer >= len(r.seats) {
		dealer = 0
	}
	g, events, err := game.NewGameWithDealer(names, r.config, dealer, r.rng)
	if err != nil {
		log.Printf("NewGame error: %v", err)
		return
	}
	r.game = g
	r.gameGen++
	r.undoSnap = nil
	r.undoSeat = -1
	r.undoUsed = make(map[int]bool)
	if r.game.Phase == game.PhaseRoundOver && r.game.Result != nil {
		r.applyRoundSettlementLocked()
	}
	r.broadcastLobbyLocked()
	r.broadcastGameLocked(events)
	r.scheduleBotTurnLocked()
}

func (r *Room) handleActionLocked(c *Client, act game.Action) {
	if r.game == nil {
		c.Send(ServerMsg{Type: "error", Error: "no active game"})
		return
	}
	seat := r.findSeatByClient(c)
	if seat < 0 {
		c.Send(ServerMsg{Type: "error", Error: "spectators cannot act"})
		return
	}
	// Save undo snapshot before a human takes a turn action
	snapBefore := r.game.Clone()
	events, err := game.ApplyAction(r.game, seat, act)
	if err != nil {
		c.Send(ServerMsg{Type: "error", Error: err.Error()})
		return
	}
	if !r.undoUsed[seat] && r.game.Phase != game.PhaseRoundOver {
		r.undoSnap = snapBefore
		r.undoSeat = seat
	} else {
		r.undoSnap = nil
		r.undoSeat = -1
	}
	if r.game.Phase == game.PhaseRoundOver && r.game.Result != nil {
		r.applyRoundSettlementLocked()
		r.broadcastLobbyLocked()
	}
	r.broadcastGameLocked(events)
	r.scheduleBotTurnLocked()
}

func (r *Room) scheduleBotTurnLocked() {
	if r.game == nil || r.game.Phase == game.PhaseRoundOver {
		return
	}
	turn := r.game.CurrentTurn
	if turn < 0 || turn >= len(r.seats) || !r.seats[turn].isBot {
		return
	}
	gen := r.gameGen
	delay := r.botDelay
	go func() {
		time.Sleep(delay)
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.game == nil || r.gameGen != gen || r.game.Phase == game.PhaseRoundOver {
			return
		}
		if r.game.CurrentTurn != turn || turn >= len(r.seats) || !r.seats[turn].isBot {
			return
		}
		act := game.ChooseBotAction(r.game, turn)
		events, err := game.ApplyAction(r.game, turn, act)
		if err != nil {
			log.Printf("bot action error: %v", err)
			return
		}
		if r.game.Phase == game.PhaseRoundOver && r.game.Result != nil {
			r.undoSnap = nil
			r.undoSeat = -1
			r.applyRoundSettlementLocked()
			r.broadcastLobbyLocked()
		}
		r.broadcastGameLocked(events)
		r.scheduleBotTurnLocked()
	}()
}

func (r *Room) applyRoundSettlementLocked() {
	if r.game != nil && r.game.Result != nil && !r.game.Result.IsDraw && r.game.Result.WinnerIdx >= 0 {
		r.nextDealer = r.game.Result.WinnerIdx
	}
	for i, delta := range r.game.Result.Settlements {
		if i < len(r.seats) {
			r.seats[i].totalPoints += delta
		}
	}
}

func (r *Room) broadcastLobbyLocked() {
	msg := ServerMsg{Type: "lobby", Lobby: r.buildLobbyLocked()}
	for c := range r.clients {
		c.Send(msg)
	}
}

// RegisterClient registers a newly connected client and pushes the current lobby/game view.
func (r *Room) RegisterClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[c] = true
	c.Send(ServerMsg{Type: "lobby", Lobby: r.buildLobbyLocked()})
	if r.game != nil {
		r.specs[c] = true
		view := r.game.ViewFor(-1)
		c.Send(ServerMsg{Type: "state", Game: &view})
	}
}

func (r *Room) buildLobbyLocked() *LobbyView {
	players := make([]LobbyPlayer, len(r.seats))
	for i, sp := range r.seats {
		players[i] = LobbyPlayer{
			Seat:        i,
			Name:        sp.name,
			Ready:       sp.ready,
			Connected:   sp.isBot || sp.client != nil,
			IsBot:       sp.isBot,
			TotalPoints: sp.totalPoints,
		}
	}
	return &LobbyView{
		Players:  players,
		Config:   r.config,
		InGame:   r.game != nil,
		HostSeat: 0,
		LanURL:   r.lanURL,
		LanURLs:  r.lanURLs,
	}
}

func (r *Room) broadcastGameLocked(events []game.Event) {
	if r.game == nil {
		return
	}
	for i, sp := range r.seats {
		if sp.client != nil {
			view := r.game.ViewFor(i)
			canUndo := r.undoSnap != nil && r.undoSeat == i && !r.undoUsed[i] && r.game.Phase != game.PhaseRoundOver
			sp.client.Send(ServerMsg{Type: "state", Game: &view, CanUndo: canUndo, Events: events})
		}
	}
	specView := r.game.ViewFor(-1)
	for sc := range r.specs {
		sc.Send(ServerMsg{Type: "state", Game: &specView, Events: events})
	}
}
