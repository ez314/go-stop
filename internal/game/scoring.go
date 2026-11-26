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
	SakeCupAsPi   bool `json:"sakeCupAsPi"` // True when Sep Sake Cup (국진) is counted as 2-Pi

	BaseScore     int  `json:"baseScore"`
	GoCount       int  `json:"goCount"`
	GoBonus       int  `json:"goBonus"`
	Multiplier    int  `json:"multiplier"` // Combined Go (3+) and Shake/Bomb multiplier
	TotalScore    int  `json:"totalScore"`
}

// CalculateScore computes a player's current score from their captured cards and multipliers.
// Automatically optimizes the Month 9 Sake Cup (국진) as either 1 Animal or 2 Pi.
func CalculateScore(p *PlayerState, cfg RuleConfig) ScoreBreakdown {
	hasSakeCup := false
	for _, c := range p.Captured {
		if c.IsSakeCup {
			hasSakeCup = true
			break
		}
	}
	asAnimal := calculateScoreWithOption(p, false)
	if !hasSakeCup {
		return asAnimal
	}
	asPi := calculateScoreWithOption(p, true)
	// Prefer higher BaseScore; on tie, prefer counting as 2-Pi (helps avoid Pi-bak)
	if asPi.BaseScore >= asAnimal.BaseScore {
		return asPi
	}
	return asAnimal
}

func calculateScoreWithOption(p *PlayerState, sakeCupAsPi bool) ScoreBreakdown {
	var b ScoreBreakdown
	b.GoCount = p.GoCount
	b.SakeCupAsPi = sakeCupAsPi

	var birds, hongdan, cheongdan, chodan int
	for _, c := range p.Captured {
		if c.IsSakeCup && sakeCupAsPi {
			b.PiCount += 2
			continue
		}
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
// +1 point per Gwang ONLY (not Double-Pi or Sake Cup), awarded only when forced to fold.
func CountSellableCards(hand []Card) int {
	pts := 0
	for _, c := range hand {
		if c.Type == TypeGwang {
			pts++
		}
	}
	return pts
}

// CalculateSettlement computes net points won/lost per player when winnerIdx stops the round.
// Handles Pi-bak, Gwang-bak, Go-bak, and 4P Gwang-palgi payouts (paid by active losers).
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

	// Collect any forced-fold player's Gwang sale bonus (paid by active losers)
	foldedIdx := -1
	foldBonus := 0
	for i, p := range players {
		if p.Folded && p.FoldBonus > 0 {
			foldedIdx = i
			foldBonus = p.FoldBonus
			penalties[i] = append(penalties[i], "광팔기")
		}
	}

	// Calculate each active loser's payment to the winner (and Gwang-palgi payment to forced-fold player)
	for i, p := range players {
		if i == winnerIdx || p.Folded {
			continue
		}
		oppScore := CalculateScore(p, cfg)
		mult := 1

		// Pi-bak: winner scored with Pi and loser has <= PiBakThreshold Pi
		if winScore.PiScore > 0 && oppScore.PiCount <= cfg.PiBakThreshold {
			mult *= 2
			penalties[i] = append(penalties[i], "Pi-bak (×2)")
		}

		amt := winScore.TotalScore * mult
		payer := i
		if goBakIdx >= 0 {
			payer = goBakIdx
		}
		settlements[payer] -= amt
		settlements[winnerIdx] += amt

		// Active loser(s) pay the forced-fold player's Gwang sale bonus (1 per Gwang)
		if foldedIdx >= 0 && foldBonus > 0 {
			settlements[payer] -= foldBonus
			settlements[foldedIdx] += foldBonus
		}
	}

	return settlements, penalties
}
