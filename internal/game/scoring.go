package game

// ScoreBreakdown details a player's current Go-Stop score components.
type ScoreBreakdown struct {
	GwangCount    int  `json:"gwangCount"`
	HasRainGwang  bool `json:"hasRainGwang"`
	GwangScore    int  `json:"gwangScore"`

	YeolkkutCount int  `json:"yeolkkutCount"`
	YeolkkutScore int  `json:"yeolkkutScore"`
	Godori        bool `json:"godori"`

	TtiCount      int  `json:"ttiCount"`
	TtiScore      int  `json:"ttiScore"`
	Hongdan       bool `json:"hongdan"`
	Cheongdan     bool `json:"cheongdan"`
	Chodan        bool `json:"chodan"`

	PiCount       int  `json:"piCount"`
	PiScore       int  `json:"piScore"`

	BaseScore     int  `json:"baseScore"`
	GoCount       int  `json:"goCount"`
	GoBonus       int  `json:"goBonus"`
	Multiplier    int  `json:"multiplier"` // Combined Go (3+) and Shake/Bomb multiplier
	TotalScore    int  `json:"totalScore"`
}

// CalculateScore computes a player's current score from their captured cards and multipliers.
func CalculateScore(p *PlayerState, cfg RuleConfig) ScoreBreakdown {
	var b ScoreBreakdown
	b.GoCount = p.GoCount

	var birds, hongdan, cheongdan, chodan int
	for _, c := range p.Captured {
		switch c.Type {
		case TypeGwang:
			b.GwangCount++
			if c.IsRain {
				b.HasRainGwang = true
			}
		case TypeYeolkkut:
			b.YeolkkutCount++
			if c.IsBird {
				birds++
			}
		case TypeTti:
			b.TtiCount++
			switch c.RibbonSet {
			case RibbonHongdan:
				hongdan++
			case RibbonCheongdan:
				cheongdan++
			case RibbonChodan:
				chodan++
			}
		case TypePi:
			val := c.PiValue
			if val <= 0 {
				val = 1
			}
			b.PiCount += val
		}
	}

	// Gwang scoring
	switch {
	case b.GwangCount >= 5:
		b.GwangScore = 15
	case b.GwangCount == 4:
		b.GwangScore = 4
	case b.GwangCount == 3:
		if b.HasRainGwang {
			b.GwangScore = 2
		} else {
			b.GwangScore = 3
		}
	}

	// Yeolkkut scoring (5+ = 1 pt + 1 per extra; Godori = +5)
	if b.YeolkkutCount >= 5 {
		b.YeolkkutScore += b.YeolkkutCount - 4
	}
	if birds == 3 {
		b.Godori = true
		b.YeolkkutScore += 5
	}

	// Tti scoring (5+ = 1 pt + 1 per extra; each 3-ribbon set = +3)
	if b.TtiCount >= 5 {
		b.TtiScore += b.TtiCount - 4
	}
	if hongdan == 3 {
		b.Hongdan = true
		b.TtiScore += 3
	}
	if cheongdan == 3 {
		b.Cheongdan = true
		b.TtiScore += 3
	}
	if chodan == 3 {
		b.Chodan = true
		b.TtiScore += 3
	}

	// Pi scoring (10+ = 1 pt + 1 per extra)
	if b.PiCount >= 10 {
		b.PiScore = b.PiCount - 9
	}

	b.BaseScore = b.GwangScore + b.YeolkkutScore + b.TtiScore + b.PiScore

	// Go bonus & multiplier
	b.Multiplier = 1
	if b.GoCount == 1 {
		b.GoBonus = 1
	} else if b.GoCount == 2 {
		b.GoBonus = 2
	} else if b.GoCount >= 3 {
		b.GoBonus = b.GoCount
		for i := 2; i < b.GoCount; i++ {
			b.Multiplier *= 2
		}
	}

	// Shake/Bomb multiplier (each shake/bomb doubles)
	for i := 0; i < p.ShakeCount; i++ {
		b.Multiplier *= 2
	}

	b.TotalScore = (b.BaseScore + b.GoBonus) * b.Multiplier
	return b
}

// CountSellableCards returns the Gwang-palgi (광팔기) payout value of a hand in 4P mode:
// +1 point per Gwang or Double-Pi card sold when folding early.
func CountSellableCards(hand []Card) int {
	pts := 0
	for _, c := range hand {
		if c.Type == TypeGwang || (c.Type == TypePi && c.PiValue >= 2) {
			pts++
		}
	}
	return pts
}

// CalculateSettlement computes net points won/lost per player when winnerIdx stops the round.
// Handles Pi-bak, Gwang-bak, Go-bak, and 4P Gwang-palgi payouts for folded players.
func CalculateSettlement(winnerIdx int, players []*PlayerState, cfg RuleConfig) []int {
	s, _ := CalculateSettlementDetailed(winnerIdx, players, cfg)
	return s
}

// CalculateSettlementDetailed returns both net settlements and human-readable penalty/bonus tags per player.
func CalculateSettlementDetailed(winnerIdx int, players []*PlayerState, cfg RuleConfig) ([]int, [][]string) {
	n := len(players)
	settlements := make([]int, n)
	penalties := make([][]string, n)
	winner := players[winnerIdx]
	winScore := CalculateScore(winner, cfg)

	// Check if another active player had called Go (Go-bak: that player pays for everyone)
	goBakIdx := -1
	for i, p := range players {
		if i != winnerIdx && !p.Folded && p.GoCount > 0 {
			goBakIdx = i
			penalties[i] = append(penalties[i], "Go-bak (pays all)")
			break
		}
	}

	// Calculate each active loser's payment to the winner, and pay folded players' Gwang-palgi bonus
	for i, p := range players {
		if i == winnerIdx {
			continue
		}
		if p.Folded {
			if p.FoldBonus > 0 {
				settlements[i] += p.FoldBonus
				settlements[winnerIdx] -= p.FoldBonus
				penalties[i] = append(penalties[i], "광팔기")
			}
			continue
		}
		oppScore := CalculateScore(p, cfg)
		mult := 1

		// Pi-bak: winner scored with Pi and loser has <= PiBakThreshold Pi
		if winScore.PiScore > 0 && oppScore.PiCount <= cfg.PiBakThreshold {
			mult *= 2
			penalties[i] = append(penalties[i], "Pi-bak (×2)")
		}
		// Gwang-bak: winner scored with Gwang and loser has 0 Gwang
		if winScore.GwangScore > 0 && oppScore.GwangCount == 0 {
			mult *= 2
			penalties[i] = append(penalties[i], "Gwang-bak (×2)")
		}

		amt := winScore.TotalScore * mult
		if goBakIdx >= 0 {
			settlements[goBakIdx] -= amt
		} else {
			settlements[i] -= amt
		}
		settlements[winnerIdx] += amt
	}

	return settlements, penalties
}
