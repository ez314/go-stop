package game

import (
	"math/rand"
	"testing"
)

func TestNewDeck(t *testing.T) {
	std := NewDeck(false)
	if len(std) != 48 {
		t.Fatalf("expected 48 standard cards, got %d", len(std))
	}
	bonus := NewDeck(true)
	if len(bonus) != 50 {
		t.Fatalf("expected 50 cards with bonus, got %d", len(bonus))
	}
}

func TestScoringAndSettlement(t *testing.T) {
	cfg := DefaultRules(2)
	p0 := &PlayerState{
		Name: "Alice",
		Captured: []Card{
			{ID: "01-gwang", Month: 1, Type: TypeGwang},
			{ID: "03-gwang", Month: 3, Type: TypeGwang},
			{ID: "08-gwang", Month: 8, Type: TypeGwang},
			{ID: "02-yeol", Month: 2, Type: TypeYeolkkut, IsBird: true},
			{ID: "04-yeol", Month: 4, Type: TypeYeolkkut, IsBird: true},
			{ID: "08-yeol", Month: 8, Type: TypeYeolkkut, IsBird: true},
			{ID: "01-pi1", Month: 1, Type: TypePi, PiValue: 1},
			{ID: "01-pi2", Month: 1, Type: TypePi, PiValue: 1},
			{ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1},
			{ID: "02-pi2", Month: 2, Type: TypePi, PiValue: 1},
			{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1},
			{ID: "03-pi2", Month: 3, Type: TypePi, PiValue: 1},
			{ID: "11-pi2x", Month: 11, Type: TypePi, PiValue: 2},
			{ID: "12-pi2x", Month: 12, Type: TypePi, PiValue: 2},
		},
		GoCount: 1,
	}
	b := CalculateScore(p0, cfg)
	// 3 Gwang (3) + Godori (5) + 10 Pi (1) = 9 base + 1 Go = 10
	if b.BaseScore != 9 || b.TotalScore != 10 {
		t.Fatalf("unexpected score breakdown: %+v", b)
	}

	p1 := &PlayerState{
		Name: "Bob",
		Captured: []Card{
			{ID: "04-pi1", Month: 4, Type: TypePi, PiValue: 1},
		},
	}
	// Bob has 0 Gwang (Gwang-bak x2) and 1 Pi <= 7 (Pi-bak x2) => 10 * 4 = 40
	settlements := CalculateSettlement(0, []*PlayerState{p0, p1}, cfg)
	if settlements[0] != 40 || settlements[1] != -40 {
		t.Fatalf("expected [40, -40], got %v", settlements)
	}
}

func TestJjokAndSweep(t *testing.T) {
	cfg := DefaultRules(2)
	s := &GameState{
		Config: cfg,
		Players: []*PlayerState{
			{Name: "P1", Hand: []Card{{ID: "01-gwang", Month: 1, Type: TypeGwang}, {ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1}}},
			{Name: "P2", Hand: []Card{{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1}}, Captured: []Card{{ID: "04-pi1", Month: 4, Type: TypePi, PiValue: 1}, {ID: "04-pi2", Month: 4, Type: TypePi, PiValue: 1}}},
		},
		Field:       []Card{},
		DrawPile:    []Card{{ID: "01-tti", Month: 1, Type: TypeTti}, {ID: "05-pi1", Month: 5, Type: TypePi, PiValue: 1}},
		CurrentTurn: 0,
		Phase:       PhasePlayHand,
	}

	events, err := ApplyAction(s, 0, Action{Type: ActionPlayCard, CardID: "01-gwang"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasJjok := false
	hasSweep := false
	for _, ev := range events {
		if ev.Type == "jjok" {
			hasJjok = true
		}
		if ev.Type == "sweep" {
			hasSweep = true
		}
	}
	if !hasJjok || !hasSweep {
		t.Fatalf("expected jjok and sweep events, got %+v", events)
	}
	// P1 should have captured 2 month-1 cards + stolen 1 Pi from Jjok + stolen 1 Pi from Sweep = 4 cards
	if len(s.Players[0].Captured) != 4 {
		t.Fatalf("expected 4 captured cards for P1, got %d", len(s.Players[0].Captured))
	}
}

func TestPpeok(t *testing.T) {
	cfg := DefaultRules(2)
	s := &GameState{
		Config: cfg,
		Players: []*PlayerState{
			{Name: "P1", Hand: []Card{{ID: "01-gwang", Month: 1, Type: TypeGwang}, {ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1}}},
			{Name: "P2", Hand: []Card{{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1}}},
		},
		Field:       []Card{{ID: "01-tti", Month: 1, Type: TypeTti}},
		DrawPile:    []Card{{ID: "01-pi1", Month: 1, Type: TypePi, PiValue: 1}, {ID: "05-pi1", Month: 5, Type: TypePi, PiValue: 1}},
		CurrentTurn: 0,
		Phase:       PhasePlayHand,
	}

	events, err := ApplyAction(s, 0, Action{Type: ActionPlayCard, CardID: "01-gwang"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasPpeok := false
	for _, ev := range events {
		if ev.Type == "ppeok" {
			hasPpeok = true
		}
	}
	if !hasPpeok {
		t.Fatalf("expected ppeok event, got %+v", events)
	}
	if len(s.Field) != 3 {
		t.Fatalf("expected 3 cards left on field after ppeok, got %d", len(s.Field))
	}
	if len(s.Players[0].Captured) != 0 {
		t.Fatalf("expected 0 captured cards after ppeok, got %d", len(s.Players[0].Captured))
	}
}

func TestFullRandomGameCompletes(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cfg := DefaultRules(2)
	s, _, err := NewGame([]string{"Alice", "Bob"}, cfg, rng)
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	steps := 0
	for s.Phase != PhaseRoundOver && steps < 200 {
		steps++
		pIdx := s.CurrentTurn
		p := s.Players[pIdx]
		var act Action
		switch s.Phase {
		case PhasePlayHand:
			if len(p.Hand) > 0 {
				act = Action{Type: ActionPlayCard, CardID: p.Hand[0].ID}
			} else if p.BombPasses > 0 {
				act = Action{Type: ActionBombPass}
			}
		case PhaseChooseHandMatch, PhaseChooseFlipMatch:
			act = Action{Type: ActionChooseMatch, CardID: s.Pending.ChoiceOptions[0].ID}
		case PhaseGoOrStop:
			act = Action{Type: ActionCallStop}
		}
		if _, err := ApplyAction(s, pIdx, act); err != nil {
			t.Fatalf("step %d (%s) error: %v", steps, s.Phase, err)
		}
	}

	if s.Phase != PhaseRoundOver {
		t.Fatalf("game did not finish within 200 steps, phase=%s", s.Phase)
	}
}

func TestFourPlayerGwangPalgi(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	cfg := DefaultRules(4)
	s, _, err := NewGame([]string{"A", "B", "C", "D"}, cfg, rng)
	if err != nil {
		t.Fatalf("NewGame 4P error: %v", err)
	}
	if s.Phase != PhaseGwangPalgi {
		t.Fatalf("expected PhaseGwangPalgi, got %s", s.Phase)
	}
	// Player 0 stays in, Player 1 folds
	if _, err := ApplyAction(s, 0, Action{Type: ActionStayIn}); err != nil {
		t.Fatalf("stay_in error: %v", err)
	}
	if s.CurrentTurn != 1 || s.Phase != PhaseGwangPalgi {
		t.Fatalf("expected turn 1 in PhaseGwangPalgi, got turn=%d phase=%s", s.CurrentTurn, s.Phase)
	}
	if _, err := ApplyAction(s, 1, Action{Type: ActionFold}); err != nil {
		t.Fatalf("fold error: %v", err)
	}
	if !s.Players[1].Folded || s.Phase != PhasePlayHand || len(s.DrawPile) != 21 {
		t.Fatalf("expected P1 folded, PhasePlayHand, and 21 draw cards; got folded=%v phase=%s draw=%d", s.Players[1].Folded, s.Phase, len(s.DrawPile))
	}

	// Play rest of the 3 active players' round to completion
	for steps := 0; steps < 250 && s.Phase != PhaseRoundOver; steps++ {
		pIdx := s.CurrentTurn
		if pIdx == 1 {
			t.Fatalf("folded player 1 should never receive a turn")
		}
		p := s.Players[pIdx]
		var act Action
		switch s.Phase {
		case PhasePlayHand:
			if len(p.Hand) > 0 {
				act = Action{Type: ActionPlayCard, CardID: p.Hand[0].ID}
			} else {
				act = Action{Type: ActionBombPass}
			}
		case PhaseChooseHandMatch, PhaseChooseFlipMatch:
			act = Action{Type: ActionChooseMatch, CardID: s.Pending.ChoiceOptions[0].ID}
		case PhaseGoOrStop:
			act = Action{Type: ActionCallStop}
		}
		if _, err := ApplyAction(s, pIdx, act); err != nil {
			t.Fatalf("4P step %d error: %v", steps, err)
		}
	}
	if s.Phase != PhaseRoundOver {
		t.Fatalf("4P game did not finish, phase=%s", s.Phase)
	}
}
