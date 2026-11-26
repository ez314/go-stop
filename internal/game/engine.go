package game

import (
	"errors"
	"fmt"
	"math/rand"
)

var (
	ErrNotYourTurn     = errors.New("not your turn")
	ErrInvalidPhase    = errors.New("action not valid in current phase")
	ErrCardNotFound    = errors.New("card not found")
	ErrInvalidChoice   = errors.New("invalid card choice")
	ErrCannotGo        = errors.New("cannot call go")
	ErrCannotStop      = errors.New("cannot call stop")
	ErrCannotBombShake = errors.New("no valid 3-of-a-kind for bomb or shake")
)

// NewGame deals a new Go-Stop round with seat 0 as the starting dealer.
func NewGame(playerNames []string, cfg RuleConfig, rng *rand.Rand) (*GameState, []Event, error) {
	return NewGameWithDealer(playerNames, cfg, 0, rng)
}

// NewGameWithDealer deals a new Go-Stop round where dealerIdx (previous winner) goes first.
func NewGameWithDealer(playerNames []string, cfg RuleConfig, dealerIdx int, rng *rand.Rand) (*GameState, []Event, error) {
	if len(playerNames) != cfg.PlayerCount {
		return nil, nil, fmt.Errorf("expected %d players, got %d", cfg.PlayerCount, len(playerNames))
	}
	if dealerIdx < 0 || dealerIdx >= cfg.PlayerCount {
		dealerIdx = 0
	}

	deck := NewDeck(cfg.IncludeBonusCards)
	Shuffle(deck, rng)

	players := make([]*PlayerState, cfg.PlayerCount)
	cursor := 0
	handSize := cfg.HandSize()
	for i, name := range playerNames {
		hand := make([]Card, handSize)
		copy(hand, deck[cursor:cursor+handSize])
		cursor += handSize
		players[i] = &PlayerState{
			Name: name,
			Hand: hand,
		}
	}

	fieldSize := cfg.FieldSize()
	field := make([]Card, fieldSize)
	copy(field, deck[cursor:cursor+fieldSize])
	cursor += fieldSize

	drawPile := make([]Card, len(deck)-cursor)
	copy(drawPile, deck[cursor:])

	initialPhase := PhasePlayHand
	initialTurn := dealerIdx
	if cfg.PlayerCount == 4 {
		// Dealer must stay in; counterclockwise decisions start at (dealerIdx + 1) % 4
		initialPhase = PhaseGwangPalgi
		initialTurn = (dealerIdx + 1) % cfg.PlayerCount
	}

	s := &GameState{
		Config:      cfg,
		Players:     players,
		Field:       field,
		DrawPile:    drawPile,
		DealerIdx:   dealerIdx,
		CurrentTurn: initialTurn,
		Phase:       initialPhase,
		rng:         rng,
	}

	startMsg := fmt.Sprintf("Round started — %s goes first!", players[dealerIdx].Name)
	if cfg.PlayerCount == 4 {
		startMsg = fmt.Sprintf("4P Round: %s is Dealer (stays in). %s decides Play or Fold!", players[dealerIdx].Name, players[initialTurn].Name)
	}
	events := []Event{{Type: "round_started", PlayerIdx: initialTurn, Message: startMsg}}

	// Move any bonus cards dealt to the field directly into Dealer's captured pile
	// and replace them from the draw pile (standard Matgo/Go-Stop rule).
	var remainingField []Card
	for _, c := range s.Field {
		if c.IsBonus {
			s.Players[dealerIdx].Captured = append(s.Players[dealerIdx].Captured, c)
			events = append(events, Event{Type: "captured", PlayerIdx: dealerIdx, Cards: []Card{c}, Message: "Bonus card on field"})
			if len(s.DrawPile) > 0 {
				rep := s.popDraw()
				remainingField = append(remainingField, rep)
			}
		} else {
			remainingField = append(remainingField, c)
		}
	}
	s.Field = remainingField

	// Check Chongtong (4 cards of the same month in a player's starting hand)
	if cfg.EnableChongtong {
		for i, p := range s.Players {
			if hasChongtong(p.Hand) {
				s.Phase = PhaseRoundOver
				settlements := make([]int, len(s.Players))
				for j := range s.Players {
					if j == i {
						settlements[j] = 10 * (len(s.Players) - 1)
					} else {
						settlements[j] = -10
					}
				}
				s.Result = &RoundResult{
					WinnerIdx:   i,
					Reason:      "chongtong",
					Settlements: settlements,
				}
				events = append(events, Event{
					Type:      "round_ended",
					PlayerIdx: i,
					Message:   fmt.Sprintf("%s wins by Chongtong!", p.Name),
				})
				return s, events, nil
			}
		}
	}

	return s, events, nil
}

func hasChongtong(cards []Card) bool {
	counts := map[int]int{}
	for _, c := range cards {
		if c.Month > 0 {
			counts[c.Month]++
			if counts[c.Month] >= 4 {
				return true
			}
		}
	}
	return false
}

// ApplyAction validates and executes a player's action, returning emitted events.
func ApplyAction(s *GameState, playerIdx int, act Action) ([]Event, error) {
	if s.Phase == PhaseRoundOver {
		return nil, ErrInvalidPhase
	}
	if playerIdx != s.CurrentTurn {
		return nil, ErrNotYourTurn
	}

	switch act.Type {
	case ActionPlayCard:
		if s.Phase != PhasePlayHand {
			return nil, ErrInvalidPhase
		}
		return s.handlePlayCard(playerIdx, act.CardID)

	case ActionBomb:
		if s.Phase != PhasePlayHand || !s.Config.EnableShakeBomb {
			return nil, ErrInvalidPhase
		}
		return s.handleBomb(playerIdx, act.CardID)

	case ActionShake:
		if s.Phase != PhasePlayHand || !s.Config.EnableShakeBomb {
			return nil, ErrInvalidPhase
		}
		return s.handleShake(playerIdx, act.CardID)

	case ActionBombPass:
		if s.Phase != PhasePlayHand {
			return nil, ErrInvalidPhase
		}
		return s.handleBombPass(playerIdx)

	case ActionChooseMatch:
		if s.Phase != PhaseChooseHandMatch && s.Phase != PhaseChooseFlipMatch {
			return nil, ErrInvalidPhase
		}
		return s.handleChooseMatch(playerIdx, act.CardID)

	case ActionCallGo:
		if s.Phase != PhaseGoOrStop {
			return nil, ErrInvalidPhase
		}
		return s.handleCallGo(playerIdx)

	case ActionCallStop:
		if s.Phase != PhaseGoOrStop {
			return nil, ErrInvalidPhase
		}
		return s.handleCallStop(playerIdx)

	case ActionFold:
		if s.Phase != PhaseGwangPalgi {
			return nil, ErrInvalidPhase
		}
		return s.handleFold(playerIdx)

	case ActionStayIn:
		if s.Phase != PhaseGwangPalgi {
			return nil, ErrInvalidPhase
		}
		return s.handleStayIn(playerIdx)

	default:
		return nil, fmt.Errorf("unknown action: %s", act.Type)
	}
}

func (s *GameState) handleFold(playerIdx int) ([]Event, error) {
	p := s.Players[playerIdx]
	p.Folded = true
	forced := s.isLastFoldSeat(playerIdx)
	if forced {
		// Only a player forced to fold (after Dealer + 2 players ahead stayed in) can sell Gwang (1 pt each)
		p.FoldBonus = CountSellableCards(p.Hand)
	} else {
		p.FoldBonus = 0
	}

	// Return folded player's cards into the draw pile and shuffle so the 3 remaining players have 21 draw cards.
	s.DrawPile = append(s.DrawPile, p.Hand...)
	p.Hand = nil
	if s.rng != nil {
		Shuffle(s.DrawPile, s.rng)
	}

	// Start normal play with Dealer
	s.Phase = PhasePlayHand
	s.CurrentTurn = s.DealerIdx

	msg := fmt.Sprintf("%s folded for this round", p.Name)
	if forced && p.FoldBonus > 0 {
		msg = fmt.Sprintf("%s was forced to fold & sold %d Gwang (+%d pts/loser)", p.Name, p.FoldBonus, p.FoldBonus)
	} else if forced {
		msg = fmt.Sprintf("%s was forced to fold (0 Gwang to sell)", p.Name)
	}
	return []Event{{Type: "player_folded", PlayerIdx: playerIdx, Points: p.FoldBonus, Message: msg}}, nil
}

func (s *GameState) handleStayIn(playerIdx int) ([]Event, error) {
	// Last player in 4P cannot stay in if Dealer + 2 players ahead already chose to play
	if s.isLastFoldSeat(playerIdx) {
		return s.handleFold(playerIdx)
	}
	p := s.Players[playerIdx]
	s.CurrentTurn = (playerIdx + 1) % len(s.Players)
	return []Event{{Type: "player_stayed", PlayerIdx: playerIdx, Message: fmt.Sprintf("%s is playing!", p.Name)}}, nil
}

func (s *GameState) handlePlayCard(playerIdx int, cardID string) ([]Event, error) {
	p := s.Players[playerIdx]
	card, ok := removeCardByID(&p.Hand, cardID)
	if !ok {
		return nil, ErrCardNotFound
	}

	var events []Event
	events = append(events, Event{Type: "card_played", PlayerIdx: playerIdx, Cards: []Card{card}})

	// Bonus card: capture immediately, draw replacement into hand, stay in PhasePlayHand
	if card.IsBonus {
		p.Captured = append(p.Captured, card)
		events = append(events, Event{Type: "captured", PlayerIdx: playerIdx, Cards: []Card{card}, Message: "Bonus card captured"})
		if len(s.DrawPile) > 0 {
			rep := s.popDraw()
			p.Hand = append(p.Hand, rep)
		}
		if len(p.Hand) == 0 && p.BombPasses == 0 {
			events = append(events, s.finalizeTurn(playerIdx, nil)...)
		}
		return events, nil
	}

	matches := s.fieldMatches(card.Month)
	s.Pending = &PendingTurn{PlayedCard: &card}

	switch len(matches) {
	case 0:
		s.Field = append(s.Field, card)
		s.Pending.WasHandEmpty = true
	case 1:
		s.Pending.HandMatched = []Card{card, matches[0]}
	case 2:
		s.Pending.ChoiceOptions = matches
		s.Phase = PhaseChooseHandMatch
		return events, nil
	default: // 3+ on field: clear all 4 + steal 1 Pi!
		captured := []Card{card}
		for _, m := range matches {
			removeCardByID(&s.Field, m.ID)
			captured = append(captured, m)
		}
		s.Pending.HandMatched = captured
		events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)
	}

	flipEvents := s.resolveFlip(playerIdx)
	events = append(events, flipEvents...)
	return events, nil
}

func (s *GameState) handleShake(playerIdx int, cardID string) ([]Event, error) {
	p := s.Players[playerIdx]
	var target *Card
	for i := range p.Hand {
		if p.Hand[i].ID == cardID {
			c := p.Hand[i]
			target = &c
			break
		}
	}
	if target == nil {
		return nil, ErrCardNotFound
	}

	shakeMonths, _ := s.availableThreeOfAKind(playerIdx)
	valid := false
	for _, m := range shakeMonths {
		if m == target.Month {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrCannotBombShake
	}

	var shakenCards []Card
	for _, c := range p.Hand {
		if c.Month == target.Month {
			shakenCards = append(shakenCards, c)
		}
	}

	p.ShakeCount++
	p.ShakenMonths = append(p.ShakenMonths, target.Month)
	// Shake reveals the 3 cards and adds x2 multiplier, but lets the player choose which card to play!
	return []Event{{
		Type:      "shake",
		PlayerIdx: playerIdx,
		Cards:     shakenCards,
		Message:   fmt.Sprintf("%s declared Shake (흔들기) on Month %d! (×2)", p.Name, target.Month),
	}}, nil
}

func (s *GameState) handleBomb(playerIdx int, cardID string) ([]Event, error) {
	p := s.Players[playerIdx]
	var month int
	for _, c := range p.Hand {
		if c.ID == cardID {
			month = c.Month
			break
		}
	}
	if month == 0 {
		return nil, ErrCardNotFound
	}

	_, bombMonths := s.availableThreeOfAKind(playerIdx)
	valid := false
	for _, m := range bombMonths {
		if m == month {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrCannotBombShake
	}

	var bombCards []Card
	var remaining []Card
	for _, c := range p.Hand {
		if c.Month == month {
			bombCards = append(bombCards, c)
		} else {
			remaining = append(remaining, c)
		}
	}
	p.Hand = remaining
	alreadyShaken := false
	for _, sm := range p.ShakenMonths {
		if sm == month {
			alreadyShaken = true
			break
		}
	}
	if !alreadyShaken {
		p.ShakeCount++
		p.ShakenMonths = append(p.ShakenMonths, month)
	}
	p.BombPasses += 2

	matches := s.fieldMatches(month)
	for _, m := range matches {
		removeCardByID(&s.Field, m.ID)
		bombCards = append(bombCards, m)
	}

	p.Captured = append(p.Captured, bombCards...)
	events := []Event{
		{Type: "bomb", PlayerIdx: playerIdx, Cards: bombCards, Message: fmt.Sprintf("%s dropped a BOMB! (x2)", p.Name)},
	}
	events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)

	s.Pending = &PendingTurn{IsBombPass: true}
	events = append(events, s.resolveFlip(playerIdx)...)
	return events, nil
}

func (s *GameState) handleBombPass(playerIdx int) ([]Event, error) {
	p := s.Players[playerIdx]
	if p.BombPasses <= 0 {
		return nil, ErrInvalidPhase
	}
	p.BombPasses--
	s.Pending = &PendingTurn{IsBombPass: true}
	events := []Event{{Type: "bomb_pass", PlayerIdx: playerIdx, Message: fmt.Sprintf("%s used a Bomb Pass", p.Name)}}
	events = append(events, s.resolveFlip(playerIdx)...)
	return events, nil
}

func (s *GameState) handleChooseMatch(playerIdx int, cardID string) ([]Event, error) {
	if s.Pending == nil {
		return nil, ErrInvalidPhase
	}
	var chosen *Card
	for _, c := range s.Pending.ChoiceOptions {
		if c.ID == cardID {
			cp := c
			chosen = &cp
			break
		}
	}
	if chosen == nil {
		return nil, ErrInvalidChoice
	}

	var events []Event
	if s.Phase == PhaseChooseHandMatch {
		s.Pending.HandMatched = []Card{*s.Pending.PlayedCard, *chosen}
		s.Pending.ChoiceOptions = nil
		events = append(events, s.resolveFlip(playerIdx)...)
		return events, nil
	}

	// PhaseChooseFlipMatch
	removeCardByID(&s.Field, chosen.ID)
	flipCaptured := []Card{*s.Pending.FlippedCard, *chosen}
	s.Pending.ChoiceOptions = nil
	events = append(events, s.finalizeTurn(playerIdx, flipCaptured)...)
	return events, nil
}

func (s *GameState) resolveFlip(playerIdx int) []Event {
	var events []Event
	p := s.Players[playerIdx]

	for len(s.DrawPile) > 0 {
		flipped := s.popDraw()
		events = append(events, Event{Type: "card_flipped", PlayerIdx: playerIdx, Cards: []Card{flipped}})

		if flipped.IsBonus {
			p.Captured = append(p.Captured, flipped)
			events = append(events, Event{Type: "captured", PlayerIdx: playerIdx, Cards: []Card{flipped}, Message: "Flipped Bonus Card!"})
			continue
		}

		s.Pending.FlippedCard = &flipped
		matches := s.fieldMatches(flipped.Month)

		// Check Ddak: hand play didn't match anything, and flip matched that exact played card
		if s.Pending.WasHandEmpty && s.Pending.PlayedCard != nil && flipped.Month == s.Pending.PlayedCard.Month {
			removeCardByID(&s.Field, s.Pending.PlayedCard.ID)
			captured := []Card{*s.Pending.PlayedCard, flipped}
			events = append(events, Event{Type: "jjok", PlayerIdx: playerIdx, Cards: captured, Message: "Ddak! (딱)"})
			events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)
			events = append(events, s.finalizeTurn(playerIdx, captured)...)
			return events
		}

		// Check Ssa or Tdadak when hand play already matched on the same month
		if len(s.Pending.HandMatched) > 0 && flipped.Month == s.Pending.HandMatched[0].Month {
			if len(s.Pending.HandMatched) == 2 && len(matches) == 1 {
				// 1 was on field, played 2nd, flipped 3rd -> Ssa!
				// Unless it's the last card of the game, all 3 stay on field.
				if len(p.Hand) > 0 || len(s.DrawPile) > 0 {
					s.Field = append(s.Field, *s.Pending.PlayedCard, flipped)
					s.Pending.HandMatched = nil
					p.PpeokCount++
					events = append(events, Event{Type: "ppeok", PlayerIdx: playerIdx, Message: "Ssa! (쌌다)"})
					events = append(events, s.finalizeTurn(playerIdx, nil)...)
					return events
				}
			}
			// Tdadak: 2 were on field (one matched by hand, one remaining on field) and flipped 4th!
			if len(matches) == 2 && len(s.Pending.HandMatched) == 2 {
				captured := append([]Card{}, s.Pending.HandMatched...)
				removeCardByID(&s.Field, s.Pending.HandMatched[1].ID)
				other := s.fieldMatches(flipped.Month)
				if len(other) > 0 {
					removeCardByID(&s.Field, other[0].ID)
					captured = append(captured, other[0], flipped)
				} else {
					captured = append(captured, flipped)
				}
				s.Pending.HandMatched = nil
				events = append(events, Event{Type: "tdadak", PlayerIdx: playerIdx, Cards: captured, Message: "Tdadak! (따닥)"})
				events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)
				events = append(events, s.finalizeTurn(playerIdx, captured)...)
				return events
			}
		}

		// Remove hand-matched field cards now before evaluating flip matches
		var handCaptured []Card
		if len(s.Pending.HandMatched) > 0 {
			handCaptured = append(handCaptured, s.Pending.HandMatched[0])
			for _, fc := range s.Pending.HandMatched[1:] {
				removeCardByID(&s.Field, fc.ID)
				handCaptured = append(handCaptured, fc)
			}
			s.Pending.HandMatched = handCaptured
			// Recompute field matches after removing hand-matched cards
			matches = s.fieldMatches(flipped.Month)
		}

		switch len(matches) {
		case 0:
			s.Field = append(s.Field, flipped)
			events = append(events, s.finalizeTurn(playerIdx, nil)...)
			return events
		case 1:
			removeCardByID(&s.Field, matches[0].ID)
			events = append(events, s.finalizeTurn(playerIdx, []Card{flipped, matches[0]})...)
			return events
		case 2:
			s.Pending.ChoiceOptions = matches
			s.Phase = PhaseChooseFlipMatch
			return events
		default:
			captured := []Card{flipped}
			for _, m := range matches {
				removeCardByID(&s.Field, m.ID)
				captured = append(captured, m)
			}
			events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)
			events = append(events, s.finalizeTurn(playerIdx, captured)...)
			return events
		}
	}

	// Draw pile empty
	events = append(events, s.finalizeTurn(playerIdx, nil)...)
	return events
}

func (s *GameState) finalizeTurn(playerIdx int, flipCaptured []Card) []Event {
	var events []Event
	p := s.Players[playerIdx]

	var allCaptured []Card
	if s.Pending != nil && len(s.Pending.HandMatched) > 0 {
		for _, c := range s.Pending.HandMatched {
			removeCardByID(&s.Field, c.ID)
		}
		allCaptured = append(allCaptured, s.Pending.HandMatched...)
	}
	allCaptured = append(allCaptured, flipCaptured...)

	if len(allCaptured) > 0 {
		p.Captured = append(p.Captured, allCaptured...)
		events = append(events, Event{Type: "captured", PlayerIdx: playerIdx, Cards: allCaptured})
	}

	// Check Sweep (판쓸이): field is empty and game isn't on its very last draw
	if len(s.Field) == 0 && len(s.DrawPile) > 0 && len(allCaptured) > 0 {
		events = append(events, Event{Type: "sweep", PlayerIdx: playerIdx, Message: "Sweep! (판쓸이)"})
		events = append(events, s.stealPiFromOpponents(playerIdx, 1)...)
	}

	s.Pending = nil

	// Check win condition
	score := CalculateScore(p, s.Config)
	if score.BaseScore >= s.Config.WinScore && score.TotalScore > p.LastGoScore {
		// If player has no more turns left, auto-stop
		if (len(p.Hand) == 0 && p.BombPasses == 0) || len(s.DrawPile) == 0 {
			s.finishRound(playerIdx, "stop")
			events = append(events, Event{Type: "round_ended", PlayerIdx: playerIdx, Message: fmt.Sprintf("%s wins!", p.Name)})
			return events
		}
		s.Phase = PhaseGoOrStop
		return events
	}

	s.advanceTurn(&events)
	return events
}

func (s *GameState) handleCallGo(playerIdx int) ([]Event, error) {
	p := s.Players[playerIdx]
	score := CalculateScore(p, s.Config)
	if score.BaseScore < s.Config.WinScore || score.TotalScore <= p.LastGoScore {
		return nil, ErrCannotGo
	}
	p.GoCount++
	p.LastGoScore = CalculateScore(p, s.Config).TotalScore
	events := []Event{{Type: "go_called", PlayerIdx: playerIdx, Message: fmt.Sprintf("%s called %d Go!", p.Name, p.GoCount)}}
	s.advanceTurn(&events)
	return events, nil
}

func (s *GameState) handleCallStop(playerIdx int) ([]Event, error) {
	p := s.Players[playerIdx]
	score := CalculateScore(p, s.Config)
	if score.BaseScore < s.Config.WinScore || score.TotalScore <= p.LastGoScore {
		return nil, ErrCannotStop
	}
	s.finishRound(playerIdx, "stop")
	return []Event{{Type: "round_ended", PlayerIdx: playerIdx, Message: fmt.Sprintf("%s called STOP and wins!", p.Name)}}, nil
}

func (s *GameState) finishRound(winnerIdx int, reason string) {
	s.Phase = PhaseRoundOver
	settlements, penalties := CalculateSettlementDetailed(winnerIdx, s.Players, s.Config)
	s.Result = &RoundResult{
		WinnerIdx:   winnerIdx,
		Reason:      reason,
		Breakdown:   CalculateScore(s.Players[winnerIdx], s.Config),
		Settlements: settlements,
		Penalties:   penalties,
	}
}

func (s *GameState) advanceTurn(events *[]Event) {
	n := len(s.Players)
	for offset := 1; offset <= n; offset++ {
		next := (s.CurrentTurn + offset) % n
		if s.Players[next].Folded {
			continue
		}
		if len(s.Players[next].Hand) > 0 || s.Players[next].BombPasses > 0 {
			s.CurrentTurn = next
			s.Phase = PhasePlayHand
			return
		}
	}
	// Nobody has cards left -> Nagari (draw)
	s.Phase = PhaseRoundOver
	s.Result = &RoundResult{
		IsDraw:      true,
		WinnerIdx:   -1,
		Reason:      "nagari",
		Settlements: make([]int, n),
	}
	*events = append(*events, Event{Type: "round_ended", PlayerIdx: -1, Message: "Nagari (Draw)"})
}

func (s *GameState) stealPiFromOpponents(thiefIdx int, count int) []Event {
	var events []Event
	for i, opp := range s.Players {
		if i == thiefIdx || opp.Folded {
			continue
		}
		for c := 0; c < count; c++ {
			stolen, ok := popOnePi(&opp.Captured)
			if ok {
				s.Players[thiefIdx].Captured = append(s.Players[thiefIdx].Captured, stolen)
				events = append(events, Event{
					Type:      "pi_stolen",
					PlayerIdx: thiefIdx,
					Cards:     []Card{stolen},
					Message:   fmt.Sprintf("Stole Pi from %s", opp.Name),
				})
			}
		}
	}
	return events
}

func popOnePi(cards *[]Card) (Card, bool) {
	// Prefer 1-Pi first, then 2-Pi/3-Pi
	for idx, c := range *cards {
		if c.Type == TypePi && c.PiValue == 1 {
			*cards = append((*cards)[:idx], (*cards)[idx+1:]...)
			return c, true
		}
	}
	for idx, c := range *cards {
		if c.Type == TypePi {
			*cards = append((*cards)[:idx], (*cards)[idx+1:]...)
			return c, true
		}
	}
	return Card{}, false
}

func (s *GameState) fieldMatches(month int) []Card {
	var out []Card
	for _, c := range s.Field {
		if c.Month == month {
			out = append(out, c)
		}
	}
	return out
}

func (s *GameState) popDraw() Card {
	c := s.DrawPile[0]
	s.DrawPile = s.DrawPile[1:]
	return c
}

func removeCardByID(cards *[]Card, id string) (Card, bool) {
	for i, c := range *cards {
		if c.ID == id {
			*cards = append((*cards)[:i], (*cards)[i+1:]...)
			return c, true
		}
	}
	return Card{}, false
}
