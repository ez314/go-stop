package game

// RuleConfig defines configurable Go-Stop / Matgo rules so variants can be
// toggled without changing engine code.
type RuleConfig struct {
	PlayerCount       int  `json:"playerCount"`       // 2 (Matgo), 3 (Go-Stop), or 4 (4P with Gwang-palgi fold)
	IncludeBonusCards bool `json:"includeBonusCards"` // Add 2-Pi and 3-Pi bonus cards
	WinScore          int  `json:"winScore"`          // Minimum score to call Stop (7 for 2P, 3 for 3P/4P)
	PiBakThreshold    int  `json:"piBakThreshold"`    // Opponent Pi <= threshold triggers Pi-bak (7 for 2P, 5 for 3P/4P)
	EnableShakeBomb   bool `json:"enableShakeBomb"`   // Allow Shake and Bomb with 3-of-a-kind
	EnableChongtong   bool `json:"enableChongtong"`   // Instant win on 4-of-a-kind in initial hand
}

// DefaultRules returns standard rules for 2-player Matgo, 3-player Go-Stop, or 4-player Go-Stop.
func DefaultRules(playerCount int) RuleConfig {
	if playerCount == 2 {
		return RuleConfig{
			PlayerCount:       2,
			IncludeBonusCards: false,
			WinScore:          7,
			PiBakThreshold:    7,
			EnableShakeBomb:   true,
			EnableChongtong:   true,
		}
	}
	if playerCount == 4 {
		return RuleConfig{
			PlayerCount:       4,
			IncludeBonusCards: false,
			WinScore:          3,
			PiBakThreshold:    5,
			EnableShakeBomb:   true,
			EnableChongtong:   true,
		}
	}
	return RuleConfig{
		PlayerCount:       3,
		IncludeBonusCards: false,
		WinScore:          3,
		PiBakThreshold:    5,
		EnableShakeBomb:   true,
		EnableChongtong:   true,
	}
}

// HandSize returns the number of cards dealt to each player.
func (c RuleConfig) HandSize() int {
	if c.PlayerCount == 2 {
		return 10
	}
	return 7
}

// FieldSize returns the number of cards initially dealt face-up to the field.
func (c RuleConfig) FieldSize() int {
	if c.PlayerCount == 2 {
		return 8
	}
	return 6
}
