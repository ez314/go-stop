package game

// ChooseBotAction selects a smart, family-friendly move for an AI bot seat.
func ChooseBotAction(s *GameState, playerIdx int) Action {
	p := s.Players[playerIdx]

	switch s.Phase {
	case PhaseGwangPalgi:
		if playerIdx == len(s.Players)-1 {
			return Action{Type: ActionFold}
		}
		// Voluntary fold if hand has 0 Gwang and weak matching potential; otherwise stay in
		if CountSellableCards(p.Hand) == 0 && len(s.fieldMatches(p.Hand[0].Month)) == 0 {
			return Action{Type: ActionFold}
		}
		return Action{Type: ActionStayIn}

	case PhaseChooseHandMatch, PhaseChooseFlipMatch:
		if s.Pending == nil || len(s.Pending.ChoiceOptions) == 0 {
			return Action{Type: ActionChooseMatch}
		}
		best := s.Pending.ChoiceOptions[0]
		bestVal := cardPriority(best)
		for _, c := range s.Pending.ChoiceOptions[1:] {
			if v := cardPriority(c); v > bestVal {
				best = c
				bestVal = v
			}
		}
		return Action{Type: ActionChooseMatch, CardID: best.ID}

	case PhaseGoOrStop:
		// Call Go if we have plenty of cards left and no opponent is close to winning
		oppClose := false
		for i, opp := range s.Players {
			if i == playerIdx || opp.Folded {
				continue
			}
			if CalculateScore(opp, s.Config).BaseScore >= s.Config.WinScore-2 {
				oppClose = true
				break
			}
		}
		if !oppClose && len(p.Hand) >= 3 && p.GoCount < 2 {
			return Action{Type: ActionCallGo}
		}
		return Action{Type: ActionCallStop}

	case PhasePlayHand:
		if len(p.Hand) == 0 && p.BombPasses > 0 {
			return Action{Type: ActionBombPass}
		}

		// 1. Play bonus cards immediately
		for _, c := range p.Hand {
			if c.IsBonus {
				return Action{Type: ActionPlayCard, CardID: c.ID}
			}
		}

		// 2. Check Bomb / Shake
		if s.Config.EnableShakeBomb {
			shakeMonths, bombMonths := s.availableThreeOfAKind(playerIdx)
			if len(bombMonths) > 0 {
				for _, c := range p.Hand {
					if c.Month == bombMonths[0] {
						return Action{Type: ActionBomb, CardID: c.ID}
					}
				}
			}
			if len(shakeMonths) > 0 {
				for _, c := range p.Hand {
					if c.Month == shakeMonths[0] {
						return Action{Type: ActionShake, CardID: c.ID}
					}
				}
			}
		}

		// 3. Score each card in hand: prefer matching high-value cards on field, else discard lowest value
		bestCard := p.Hand[0]
		bestScore := -999
		for _, c := range p.Hand {
			matches := s.fieldMatches(c.Month)
			sc := 0
			if len(matches) == 3 {
				sc = 100 + cardPriority(c) // Clear a Ppeok!
			} else if len(matches) > 0 {
				maxField := 0
				for _, m := range matches {
					if v := cardPriority(m); v > maxField {
						maxField = v
					}
				}
				sc = 30 + cardPriority(c) + maxField
			} else {
				// No match: prefer discarding low-priority Pi
				sc = -cardPriority(c)
			}
			if sc > bestScore {
				bestScore = sc
				bestCard = c
			}
		}
		return Action{Type: ActionPlayCard, CardID: bestCard.ID}
	}

	return Action{}
}

func cardPriority(c Card) int {
	if c.IsBonus {
		return 25
	}
	switch c.Type {
	case TypeGwang:
		if c.IsRain {
			return 18
		}
		return 22
	case TypeYeolkkut:
		if c.IsSakeCup {
			return 19
		}
		if c.IsBird {
			return 17
		}
		return 11
	case TypeTti:
		if c.RibbonSet != RibbonNone {
			return 12
		}
		return 8
	default:
		if c.PiValue >= 2 {
			return 16
		}
		return 4
	}
}
