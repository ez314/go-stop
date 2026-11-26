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
	// Bob has 1 Pi <= 7 (Pi-bak x2; no Gwang-bak) => 10 * 2 = 20
	settlements := CalculateSettlement(0, []*PlayerState{p0, p1}, cfg)
	if settlements[0] != 20 || settlements[1] != -20 {
		t.Fatalf("expected [20, -20], got %v", settlements)
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
	s, _, err := NewGame([]string{"Dealer", "B", "C", "D"}, cfg, rng)
	if err != nil {
		t.Fatalf("NewGame 4P error: %v", err)
	}
	if s.Phase != PhaseGwangPalgi || s.CurrentTurn != 1 {
		t.Fatalf("expected PhaseGwangPalgi starting at seat 1 (Dealer seat 0 stays in), got phase=%s turn=%d", s.Phase, s.CurrentTurn)
	}
	// Seat 1 & Seat 2 stay in -> Seat 3 is forced to fold and can sell Gwang!
	if _, err := ApplyAction(s, 1, Action{Type: ActionStayIn}); err != nil {
		t.Fatalf("seat 1 stay_in error: %v", err)
	}
	if _, err := ApplyAction(s, 2, Action{Type: ActionStayIn}); err != nil {
		t.Fatalf("seat 2 stay_in error: %v", err)
	}
	if s.CurrentTurn != 3 || s.Phase != PhaseGwangPalgi {
		t.Fatalf("expected turn 3 forced fold, got turn=%d phase=%s", s.CurrentTurn, s.Phase)
	}
	expectedGwang := CountSellableCards(s.Players[3].Hand)
	if _, err := ApplyAction(s, 3, Action{Type: ActionFold}); err != nil {
		t.Fatalf("seat 3 forced fold error: %v", err)
	}
	if !s.Players[3].Folded || s.Players[3].FoldBonus != expectedGwang || s.Phase != PhasePlayHand || s.CurrentTurn != 0 || len(s.DrawPile) != 21 {
		t.Fatalf("unexpected state after forced fold: folded=%v bonus=%d (want %d) phase=%s turn=%d draw=%d",
			s.Players[3].Folded, s.Players[3].FoldBonus, expectedGwang, s.Phase, s.CurrentTurn, len(s.DrawPile))
	}

	// Play rest of the 3 active players' round to completion
	for steps := 0; steps < 250 && s.Phase != PhaseRoundOver; steps++ {
		pIdx := s.CurrentTurn
		if pIdx == 3 {
			t.Fatalf("folded player 3 should never receive a turn")
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

func TestSakeCupAlwaysAnimal(t *testing.T) {
	cfg := DefaultRules(2)
	// Even with 8 Pi, Sake Cup always counts as Animal (never 2-Pi)
	p1 := &PlayerState{
		Name: "Alice",
		Captured: []Card{
			{ID: "09-yeol", Month: 9, Type: TypeYeolkkut, IsSakeCup: true},
			{ID: "01-pi1", Month: 1, Type: TypePi, PiValue: 1},
			{ID: "01-pi2", Month: 1, Type: TypePi, PiValue: 1},
			{ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1},
			{ID: "02-pi2", Month: 2, Type: TypePi, PiValue: 1},
			{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1},
			{ID: "03-pi2", Month: 3, Type: TypePi, PiValue: 1},
			{ID: "11-pi2x", Month: 11, Type: TypePi, PiValue: 2},
		},
	}
	b1 := CalculateScore(p1, cfg)
	if b1.YeolkkutCount != 1 || b1.PiCount != 8 || b1.PiScore != 0 {
		t.Fatalf("expected Sake Cup counted strictly as Animal (Pi=8, Animal=1), got %+v", b1)
	}

	// With 4 other Animals + Sake Cup => 5 Animals = 1 pt
	p2 := &PlayerState{
		Name: "Bob",
		Captured: []Card{
			{ID: "09-yeol", Month: 9, Type: TypeYeolkkut, IsSakeCup: true},
			{ID: "02-yeol", Month: 2, Type: TypeYeolkkut},
			{ID: "04-yeol", Month: 4, Type: TypeYeolkkut},
			{ID: "05-yeol", Month: 5, Type: TypeYeolkkut},
			{ID: "06-yeol", Month: 6, Type: TypeYeolkkut},
		},
	}
	b2 := CalculateScore(p2, cfg)
	if b2.YeolkkutCount != 5 || b2.YeolkkutScore != 1 {
		t.Fatalf("expected Sake Cup counted as 5th Animal (1 pt), got %+v", b2)
	}
}

func TestBotPlaysFullGame(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	cfg := DefaultRules(3)
	s, _, err := NewGame([]string{"Bot1", "Bot2", "Bot3"}, cfg, rng)
	if err != nil {
		t.Fatal(err)
	}
	for steps := 0; steps < 250 && s.Phase != PhaseRoundOver; steps++ {
		act := ChooseBotAction(s, s.CurrentTurn)
		if _, err := ApplyAction(s, s.CurrentTurn, act); err != nil {
			t.Fatalf("bot step %d (%s) error: %v", steps, s.Phase, err)
		}
	}
	if s.Phase != PhaseRoundOver {
		t.Fatalf("bot game did not finish, phase=%s", s.Phase)
	}
}

func TestFiveGwangAndThreeSsaAndSsakSeuRi(t *testing.T) {
	cfg := DefaultRules(2)
	// 1. 5 Gwang = 25 points
	p5g := &PlayerState{
		Name: "Alice",
		Captured: []Card{
			{ID: "01-gwang", Month: 1, Type: TypeGwang},
			{ID: "03-gwang", Month: 3, Type: TypeGwang},
			{ID: "08-gwang", Month: 8, Type: TypeGwang},
			{ID: "11-gwang", Month: 11, Type: TypeGwang},
			{ID: "12-gwang", Month: 12, Type: TypeGwang, IsRain: true},
		},
	}
	if b := CalculateScore(p5g, cfg); b.GwangScore != 25 || b.BaseScore != 25 {
		t.Fatalf("expected 5 Gwang = 25 pts, got %+v", b)
	}

	// 2. 3x Ssa = auto-win 3 points
	s := &GameState{
		Config: cfg,
		Players: []*PlayerState{
			{Name: "P1", PpeokCount: 2, Hand: []Card{{ID: "01-gwang", Month: 1, Type: TypeGwang}, {ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1}}},
			{Name: "P2", Hand: []Card{{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1}}},
		},
		Field:       []Card{{ID: "01-tti", Month: 1, Type: TypeTti}},
		DrawPile:    []Card{{ID: "01-pi1", Month: 1, Type: TypePi, PiValue: 1}, {ID: "05-pi1", Month: 5, Type: TypePi, PiValue: 1}},
		CurrentTurn: 0,
		Phase:       PhasePlayHand,
	}
	if _, err := ApplyAction(s, 0, Action{Type: ActionPlayCard, CardID: "01-gwang"}); err != nil {
		t.Fatalf("3x ssa error: %v", err)
	}
	if s.Phase != PhaseRoundOver || s.Result == nil || s.Result.Reason != "three_ssa" || s.Result.Settlements[0] != 3 || s.Result.Settlements[1] != -3 {
		t.Fatalf("expected 3x ssa auto-win 3 pts, got phase=%s res=%+v", s.Phase, s.Result)
	}

	// 3. Ssak-seu-ri: 2 cards on field, play 1 match + flip 2nd match -> empty field -> steal 1 Pi!
	s2 := &GameState{
		Config: cfg,
		Players: []*PlayerState{
			{Name: "P1", Hand: []Card{{ID: "01-gwang", Month: 1, Type: TypeGwang}, {ID: "05-pi1", Month: 5, Type: TypePi, PiValue: 1}}},
			{Name: "P2", Hand: []Card{{ID: "06-pi1", Month: 6, Type: TypePi, PiValue: 1}}, Captured: []Card{{ID: "07-pi1", Month: 7, Type: TypePi, PiValue: 1}}},
		},
		Field:       []Card{{ID: "01-tti", Month: 1, Type: TypeTti}, {ID: "02-tti", Month: 2, Type: TypeTti}},
		DrawPile:    []Card{{ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1}, {ID: "08-pi1", Month: 8, Type: TypePi, PiValue: 1}},
		CurrentTurn: 0,
		Phase:       PhasePlayHand,
	}
	evs, err := ApplyAction(s2, 0, Action{Type: ActionPlayCard, CardID: "01-gwang"})
	if err != nil {
		t.Fatalf("ssak-seu-ri error: %v", err)
	}
	hasSweep := false
	for _, ev := range evs {
		if ev.Type == "sweep" {
			hasSweep = true
		}
	}
	if !hasSweep || len(s2.Field) != 0 || len(s2.Players[1].Captured) != 0 {
		t.Fatalf("expected ssak-seu-ri stealing P2's Pi, sweep=%v field=%d p2cap=%d", hasSweep, len(s2.Field), len(s2.Players[1].Captured))
	}
}
