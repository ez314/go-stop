package game

import "math/rand"

// CardType represents the four Hwatu scoring categories.
type CardType string

const (
	TypeGwang    CardType = "gwang"
	TypeYeolkkut CardType = "yeolkkut"
	TypeTti      CardType = "tti"
	TypePi       CardType = "pi"
)

// RibbonSet identifies special 3-card ribbon scoring combinations.
type RibbonSet string

const (
	RibbonNone     RibbonSet = ""
	RibbonHongdan  RibbonSet = "hongdan"  // Jan, Feb, Mar red poetry ribbons
	RibbonCheongdan RibbonSet = "cheongdan" // Jun, Sep, Oct blue ribbons
	RibbonChodan   RibbonSet = "chodan"   // Apr, May, Jul plain red ribbons
)

// Card represents a single Hwatu card.
type Card struct {
	ID        string    `json:"id"`
	Month     int       `json:"month"`     // 1-12 for standard cards, 0 for bonus cards
	Type      CardType  `json:"type"`
	PiValue   int       `json:"piValue"`   // 1 for normal pi, 2 for double pi, 3 for bonus 3-pi
	Name      string    `json:"name"`      // Korean/English display name
	IsBird    bool      `json:"isBird"`    // Feb, Apr, Aug birds (Godori)
	IsRain    bool      `json:"isRain"`    // Dec Gwang (Bi-gwang)
	IsSakeCup bool      `json:"isSakeCup"` // Sep Sake Cup (국진): counts as 1 Animal OR 2 Pi
	IsBonus   bool      `json:"isBonus"`   // Joker/Service card
	RibbonSet RibbonSet `json:"ribbonSet"` // Special ribbon set if applicable
}

// NewDeck returns a fresh, unshuffled Hwatu deck (48 standard cards, plus 2 bonus cards if enabled).
func NewDeck(includeBonus bool) []Card {
	deck := []Card{
		// Month 1: Pine (송학)
		{ID: "01-gwang", Month: 1, Type: TypeGwang, Name: "Pine Crane"},
		{ID: "01-tti", Month: 1, Type: TypeTti, RibbonSet: RibbonHongdan, Name: "Pine Poetry"},
		{ID: "01-pi1", Month: 1, Type: TypePi, PiValue: 1, Name: "Pine Pi"},
		{ID: "01-pi2", Month: 1, Type: TypePi, PiValue: 1, Name: "Pine Pi"},

		// Month 2: Plum Blossom (매조)
		{ID: "02-yeol", Month: 2, Type: TypeYeolkkut, IsBird: true, Name: "Plum Bird"},
		{ID: "02-tti", Month: 2, Type: TypeTti, RibbonSet: RibbonHongdan, Name: "Plum Poetry"},
		{ID: "02-pi1", Month: 2, Type: TypePi, PiValue: 1, Name: "Plum Pi"},
		{ID: "02-pi2", Month: 2, Type: TypePi, PiValue: 1, Name: "Plum Pi"},

		// Month 3: Cherry Blossom (벚꽃)
		{ID: "03-gwang", Month: 3, Type: TypeGwang, Name: "Cherry Curtain"},
		{ID: "03-tti", Month: 3, Type: TypeTti, RibbonSet: RibbonHongdan, Name: "Cherry Poetry"},
		{ID: "03-pi1", Month: 3, Type: TypePi, PiValue: 1, Name: "Cherry Pi"},
		{ID: "03-pi2", Month: 3, Type: TypePi, PiValue: 1, Name: "Cherry Pi"},

		// Month 4: Wisteria (흑싸리)
		{ID: "04-yeol", Month: 4, Type: TypeYeolkkut, IsBird: true, Name: "Wisteria Cuckoo"},
		{ID: "04-tti", Month: 4, Type: TypeTti, RibbonSet: RibbonChodan, Name: "Wisteria Ribbon"},
		{ID: "04-pi1", Month: 4, Type: TypePi, PiValue: 1, Name: "Wisteria Pi"},
		{ID: "04-pi2", Month: 4, Type: TypePi, PiValue: 1, Name: "Wisteria Pi"},

		// Month 5: Iris (난초)
		{ID: "05-yeol", Month: 5, Type: TypeYeolkkut, Name: "Iris Bridge"},
		{ID: "05-tti", Month: 5, Type: TypeTti, RibbonSet: RibbonChodan, Name: "Iris Ribbon"},
		{ID: "05-pi1", Month: 5, Type: TypePi, PiValue: 1, Name: "Iris Pi"},
		{ID: "05-pi2", Month: 5, Type: TypePi, PiValue: 1, Name: "Iris Pi"},

		// Month 6: Peony (모란)
		{ID: "06-yeol", Month: 6, Type: TypeYeolkkut, Name: "Peony Butterfly"},
		{ID: "06-tti", Month: 6, Type: TypeTti, RibbonSet: RibbonCheongdan, Name: "Peony Blue"},
		{ID: "06-pi1", Month: 6, Type: TypePi, PiValue: 1, Name: "Peony Pi"},
		{ID: "06-pi2", Month: 6, Type: TypePi, PiValue: 1, Name: "Peony Pi"},

		// Month 7: Bush Clover (홍싸리)
		{ID: "07-yeol", Month: 7, Type: TypeYeolkkut, Name: "Clover Boar"},
		{ID: "07-tti", Month: 7, Type: TypeTti, RibbonSet: RibbonChodan, Name: "Clover Ribbon"},
		{ID: "07-pi1", Month: 7, Type: TypePi, PiValue: 1, Name: "Clover Pi"},
		{ID: "07-pi2", Month: 7, Type: TypePi, PiValue: 1, Name: "Clover Pi"},

		// Month 8: Pampas Grass / Moon (공산)
		{ID: "08-gwang", Month: 8, Type: TypeGwang, Name: "Full Moon"},
		{ID: "08-yeol", Month: 8, Type: TypeYeolkkut, IsBird: true, Name: "Moon Geese"},
		{ID: "08-pi1", Month: 8, Type: TypePi, PiValue: 1, Name: "Moon Pi"},
		{ID: "08-pi2", Month: 8, Type: TypePi, PiValue: 1, Name: "Moon Pi"},

		// Month 9: Chrysanthemum (국화)
		{ID: "09-yeol", Month: 9, Type: TypeYeolkkut, IsSakeCup: true, Name: "Sake Cup"},
		{ID: "09-tti", Month: 9, Type: TypeTti, RibbonSet: RibbonCheongdan, Name: "Mum Blue"},
		{ID: "09-pi1", Month: 9, Type: TypePi, PiValue: 1, Name: "Mum Pi"},
		{ID: "09-pi2", Month: 9, Type: TypePi, PiValue: 1, Name: "Mum Pi"},

		// Month 10: Maple (단풍)
		{ID: "10-yeol", Month: 10, Type: TypeYeolkkut, Name: "Maple Deer"},
		{ID: "10-tti", Month: 10, Type: TypeTti, RibbonSet: RibbonCheongdan, Name: "Maple Blue"},
		{ID: "10-pi1", Month: 10, Type: TypePi, PiValue: 1, Name: "Maple Pi"},
		{ID: "10-pi2", Month: 10, Type: TypePi, PiValue: 1, Name: "Maple Pi"},

		// Month 11: Paulownia (오동)
		{ID: "11-gwang", Month: 11, Type: TypeGwang, Name: "Phoenix"},
		{ID: "11-pi2x", Month: 11, Type: TypePi, PiValue: 2, Name: "Paulownia 2-Pi"},
		{ID: "11-pi1", Month: 11, Type: TypePi, PiValue: 1, Name: "Paulownia Pi"},
		{ID: "11-pi2", Month: 11, Type: TypePi, PiValue: 1, Name: "Paulownia Pi"},

		// Month 12: Willow / Rain (비)
		{ID: "12-gwang", Month: 12, Type: TypeGwang, IsRain: true, Name: "Rain Man"},
		{ID: "12-yeol", Month: 12, Type: TypeYeolkkut, Name: "Rain Swallow"},
		{ID: "12-tti", Month: 12, Type: TypeTti, Name: "Rain Ribbon"},
		{ID: "12-pi2x", Month: 12, Type: TypePi, PiValue: 2, Name: "Rain 2-Pi"},
	}

	if includeBonus {
		deck = append(deck,
			Card{ID: "00-bonus2", Month: 0, Type: TypePi, PiValue: 2, IsBonus: true, Name: "Bonus 2-Pi"},
			Card{ID: "00-bonus3", Month: 0, Type: TypePi, PiValue: 3, IsBonus: true, Name: "Bonus 3-Pi"},
		)
	}
	return deck
}

// Shuffle randomly reorders a slice of cards in place.
func Shuffle(cards []Card, rng *rand.Rand) {
	rng.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
}
