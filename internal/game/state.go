package game

import "math/rand"

// TurnPhase represents the current step in the Go-Stop state machine.
type TurnPhase string

const (
	PhaseGwangPalgi      TurnPhase = "gwang_palgi" // 4P: players go around choosing Play or Fold/Sell Gwang
	PhasePlayHand        TurnPhase = "play_hand"
	PhaseChooseHandMatch TurnPhase = "choose_hand_match"
	PhaseChooseFlipMatch TurnPhase = "choose_flip_match"
	PhaseGoOrStop        TurnPhase = "go_or_stop"
	PhaseRoundOver       TurnPhase = "round_over"
)

// ActionType enumerates valid player actions.
type ActionType string

const (
	ActionPlayCard    ActionType = "play_card"
	ActionBomb        ActionType = "bomb"
	ActionShake       ActionType = "shake"
	ActionBombPass    ActionType = "bomb_pass"
	ActionChooseMatch ActionType = "choose_match"
	ActionCallGo      ActionType = "call_go"
	ActionCallStop    ActionType = "call_stop"
	ActionFold        ActionType = "fold"         // 4P: forfeit early & sell Gwang/Double-Pi
	ActionStayIn      ActionType = "stay_in"      // 4P: stay in and play this round
)

// Action is a player's intent submitted to the engine.
type Action struct {
	Type   ActionType `json:"type"`
	CardID string     `json:"cardId,omitempty"`
}

// Event represents a discrete domain event emitted by a state transition.
type Event struct {
	Type      string   `json:"type"`
	PlayerIdx int      `json:"playerIdx"`
	Cards     []Card   `json:"cards,omitempty"`
	Message   string   `json:"message,omitempty"`
	Points    int      `json:"points,omitempty"`
}

// PlayerState holds a single player's hand, captured piles, and multipliers.
type PlayerState struct {
	Name        string `json:"name"`
	Hand        []Card `json:"hand"`
	Captured    []Card `json:"captured"`
	GoCount     int    `json:"goCount"`
	LastGoScore int    `json:"lastGoScore"`
	ShakeCount  int    `json:"shakeCount"`
	BombPasses  int    `json:"bombPasses"` // Free draw-only turns earned from playing a bomb
	PpeokCount  int    `json:"ppeokCount"`
	Folded      bool   `json:"folded"`      // 4P: sat out via Gwang-palgi
	FoldBonus   int    `json:"foldBonus"`   // 4P: points earned from selling Gwang/Double-Pi
}

// PendingTurn tracks mid-turn state while waiting for player disambiguation.
type PendingTurn struct {
	PlayedCard    *Card  `json:"playedCard,omitempty"`
	FlippedCard   *Card  `json:"flippedCard,omitempty"`
	HandMatched   []Card `json:"handMatched,omitempty"`   // Cards matched by hand play
	ChoiceOptions []Card `json:"choiceOptions,omitempty"` // 2 field cards to choose from
	WasHandEmpty  bool   `json:"wasHandEmpty,omitempty"`  // True if played card matched 0 on field
	IsBombPass    bool   `json:"isBombPass,omitempty"`
}

// RoundResult summarizes the outcome of a finished round.
type RoundResult struct {
	IsDraw      bool           `json:"isDraw"` // Nagari
	WinnerIdx   int            `json:"winnerIdx"`
	Reason      string         `json:"reason"` // "stop", "chongtong", "nagari"
	Breakdown   ScoreBreakdown `json:"breakdown"`
	Settlements []int          `json:"settlements"`         // Net points per player
	Penalties   [][]string     `json:"penalties,omitempty"` // Per-player tags e.g. "Pi-bak (x2)", "Gwang-bak (x2)", "Go-bak", "Folded (+2)"
}

// GameState is the authoritative state of a single Go-Stop round.
type GameState struct {
	Config      RuleConfig     `json:"config"`
	Players     []*PlayerState `json:"players"`
	Field       []Card         `json:"field"`
	DrawPile    []Card         `json:"drawPile"`
	CurrentTurn int            `json:"currentTurn"`
	Phase       TurnPhase      `json:"phase"`
	Pending     *PendingTurn   `json:"pending,omitempty"`
	Result      *RoundResult   `json:"result,omitempty"`
	rng         *rand.Rand
}

// PublicPlayerView is what every player can see about another player.
type PublicPlayerView struct {
	Index      int            `json:"index"`
	Name       string         `json:"name"`
	HandCount  int            `json:"handCount"`
	BombPasses int            `json:"bombPasses"`
	Captured   []Card         `json:"captured"`
	GoCount    int            `json:"goCount"`
	ShakeCount int            `json:"shakeCount"`
	Folded     bool           `json:"folded"`
	FoldBonus  int            `json:"foldBonus"`
	Score      ScoreBreakdown `json:"score"`
}

// PlayerView is the sanitized game state sent to a specific player.
type PlayerView struct {
	MyIndex        int                `json:"myIndex"`
	MyHand         []Card             `json:"myHand"`
	Config         RuleConfig         `json:"config"`
	Players        []PublicPlayerView `json:"players"`
	Field          []Card             `json:"field"`
	DrawCount      int                `json:"drawCount"`
	CurrentTurn    int                `json:"currentTurn"`
	Phase          TurnPhase          `json:"phase"`
	ChoiceCards    []Card             `json:"choiceCards,omitempty"`
	ShakeMonths    []int              `json:"shakeMonths,omitempty"`
	BombMonths     []int              `json:"bombMonths,omitempty"`
	FoldSellValue  int                `json:"foldSellValue,omitempty"` // 4P: how many pts you get if you fold now
	MustFold       bool               `json:"mustFold,omitempty"`      // 4P: 4th player must fold if first 3 stayed in
	Result         *RoundResult       `json:"result,omitempty"`
}

// Clone creates a deep copy of GameState (preserving rng reference) for Undo snapshots.
func (s *GameState) Clone() *GameState {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Field = append([]Card(nil), s.Field...)
	cp.DrawPile = append([]Card(nil), s.DrawPile...)
	cp.Players = make([]*PlayerState, len(s.Players))
	for i, p := range s.Players {
		pCopy := *p
		pCopy.Hand = append([]Card(nil), p.Hand...)
		pCopy.Captured = append([]Card(nil), p.Captured...)
		cp.Players[i] = &pCopy
	}
	if s.Pending != nil {
		pend := *s.Pending
		if s.Pending.PlayedCard != nil {
			c := *s.Pending.PlayedCard
			pend.PlayedCard = &c
		}
		if s.Pending.FlippedCard != nil {
			c := *s.Pending.FlippedCard
			pend.FlippedCard = &c
		}
		pend.HandMatched = append([]Card(nil), s.Pending.HandMatched...)
		pend.ChoiceOptions = append([]Card(nil), s.Pending.ChoiceOptions...)
		cp.Pending = &pend
	}
	return &cp
}

// ViewFor builds a sanitized PlayerView for seat playerIdx (or -1 for spectator).
func (s *GameState) ViewFor(playerIdx int) PlayerView {
	pubPlayers := make([]PublicPlayerView, len(s.Players))
	for i, p := range s.Players {
		pubPlayers[i] = PublicPlayerView{
			Index:      i,
			Name:       p.Name,
			HandCount:  len(p.Hand),
			BombPasses: p.BombPasses,
			Captured:   append([]Card(nil), p.Captured...),
			GoCount:    p.GoCount,
			ShakeCount: p.ShakeCount,
			Folded:     p.Folded,
			FoldBonus:  p.FoldBonus,
			Score:      CalculateScore(p, s.Config),
		}
	}

	view := PlayerView{
		MyIndex:     playerIdx,
		Config:      s.Config,
		Players:     pubPlayers,
		Field:       append([]Card(nil), s.Field...),
		DrawCount:   len(s.DrawPile),
		CurrentTurn: s.CurrentTurn,
		Phase:       s.Phase,
		Result:      s.Result,
	}

	if playerIdx >= 0 && playerIdx < len(s.Players) {
		view.MyHand = append([]Card(nil), s.Players[playerIdx].Hand...)
		if s.CurrentTurn == playerIdx && s.Phase == PhasePlayHand && s.Config.EnableShakeBomb {
			view.ShakeMonths, view.BombMonths = s.availableThreeOfAKind(playerIdx)
		}
		if s.CurrentTurn == playerIdx && s.Phase == PhaseGwangPalgi {
			view.FoldSellValue = CountSellableCards(s.Players[playerIdx].Hand)
			view.MustFold = playerIdx == len(s.Players)-1
		}
	}

	if s.Pending != nil && (s.Phase == PhaseChooseHandMatch || s.Phase == PhaseChooseFlipMatch) && s.CurrentTurn == playerIdx {
		view.ChoiceCards = append([]Card(nil), s.Pending.ChoiceOptions...)
	}

	return view
}

func (s *GameState) availableThreeOfAKind(playerIdx int) (shakeMonths []int, bombMonths []int) {
	counts := map[int]int{}
	for _, c := range s.Players[playerIdx].Hand {
		if c.Month > 0 {
			counts[c.Month]++
		}
	}
	fieldCounts := map[int]int{}
	for _, c := range s.Field {
		if c.Month > 0 {
			fieldCounts[c.Month]++
		}
	}
	for m, cnt := range counts {
		if cnt == 3 {
			if fieldCounts[m] == 1 {
				bombMonths = append(bombMonths, m)
			} else if fieldCounts[m] == 0 {
				shakeMonths = append(shakeMonths, m)
			}
		}
	}
	return shakeMonths, bombMonths
}
