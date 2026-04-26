package game

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"Game_KataBaku/internal/types"
)

type Game struct {
	RoomCode    string
	Status      string
	Players     map[string]*types.PlayerState
	MainCard    string
	Deck        *Deck
	HelperCards []Card
	Validator   *Validator
	LastPlay    time.Time
	LastPlayer  string
	mu          sync.RWMutex
}

func NewGame(roomCode string) *Game {
	return &Game{
		RoomCode:    roomCode,
		Status:      "waiting",
		Players:     make(map[string]*types.PlayerState),
		MainCard:    "",
		Deck:        NewDeck(),
		HelperCards: []Card{},
		Validator:   nil,
		LastPlay:    time.Time{},
	}
}

func (g *Game) Init(dictionaryFile string) error {
	v, err := NewValidator(dictionaryFile)
	if err != nil {
		return err
	}
	if err := v.loadDictionary(); err != nil {
		return err
	}
	g.Validator = v

	mainCard := GenerateMainCard()
	g.MainCard = mainCard.Letter

	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	for _, id := range playerIDs {
		if p, ok := g.Players[id]; ok {
			p.Cards = generatePlayerCards(10)
		}
	}

	return nil
}

func generatePlayerCards(count int) []string {
	pool := generateCardPool()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if count > len(pool) {
		count = len(pool)
	}
	cards := make([]string, count)
	for i := 0; i < count; i++ {
		cards[i] = pool[i].Letter
	}
	return cards
}

func (g *Game) AddPlayer(id, username string, isHost bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Players[id] = &types.PlayerState{
		ID:       id,
		Username: username,
		IsHost:   isHost,
		Cards:    []string{},
		Score:    0,
	}
}

func (g *Game) RemovePlayer(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.Players, id)
}

func (g *Game) Start() error {
	if len(g.Players) < 1 {
		return fmt.Errorf("need at least 1 player")
	}

	if g.Validator == nil {
		v, _ := NewValidator("")
		if err := v.loadDictionary(); err == nil {
			g.Validator = v
		}
	}

	mainCard := GenerateMainCard()
	g.MainCard = mainCard.Letter

	for _, p := range g.Players {
		p.Cards = generatePlayerCards(10)
	}

	Deck := NewDeck()
	g.Deck = Deck
	g.HelperCards = GenerateHelperCards(3)
	g.Status = "playing"

	return nil
}

type PlayResult struct {
	Valid       bool
	NewMainCard string
	Word        string
	Message     string
	PlayerID    string
	Timestamp   time.Time
}

func (g *Game) PlayCards(playerID string, cards []string, position string) *PlayResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, ok := g.Players[playerID]
	if !ok {
		return &PlayResult{Valid: false, Message: "player not found"}
	}

	// HARUS PERSIS 1 KARTU DARI TANGAN
	if len(cards) != 1 {
		return &PlayResult{Valid: false, Message: "hanya boleh menggunakan 1 kartu"}
	}

	if !hasCards(player.Cards, cards) {
		return &PlayResult{Valid: false, Message: "kartu tidak dimiliki"}
	}

	playedCard := cards[0]
	// posisi prefix = kartu di depan main card = KARTU + MAIN
	// posisi suffix = kartu di belakang main card = MAIN + KARTU
	word := g.MainCard + playedCard
	if position == "prefix" {
		word = playedCard + g.MainCard
	}

	if !g.Validator.IsValid(word) {
		return &PlayResult{Valid: false, Message: "kata tidak valid: " + word}
	}

	if len(word) != 4 {
		return &PlayResult{Valid: false, Message: "kata harus 4 huruf"}
	}

	// Main card baru = kartu yang dimainkan
	newMainCard := playedCard
	player.Cards = removeFirstCard(player.Cards, playedCard)
	g.MainCard = newMainCard
	g.LastPlay = time.Now()
	g.LastPlayer = playerID

	return &PlayResult{
		Valid:       true,
		NewMainCard: newMainCard,
		Word:        word,
		Message:     "valid",
		PlayerID:    playerID,
		Timestamp:   g.LastPlay,
	}
}

func hasCards(hand []string, cards []string) bool {
	count := make(map[string]int)
	for _, c := range hand {
		count[c]++
	}
	for _, c := range cards {
		count[c]--
		if count[c] < 0 {
			return false
		}
	}
	return true
}

func buildWord(mainCard string, cards []string, position string) string {
	if position == "prefix" {
		return strings.Join(cards, "") + mainCard
	}
	return mainCard + strings.Join(cards, "")
}

func containsWord(mainCard string, cards []string) bool {
	for _, c := range cards {
		if c == mainCard {
			return true
		}
	}
	return false
}

func removeCardsFromHand(hand []string, toRemove []string) []string {
	for _, c := range toRemove {
		for i, card := range hand {
			if card == c {
				hand = append(hand[:i], hand[i+1:]...)
				break
			}
		}
	}
	return hand
}

func removeFirstCard(hand []string, cardToRemove string) []string {
	for i, card := range hand {
		if card == cardToRemove {
			return append(hand[:i], hand[i+1:]...)
		}
	}
	return hand
}

func (g *Game) DrawCard(playerID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, ok := g.Players[playerID]
	if !ok {
		return "", fmt.Errorf("player not found")
	}

	card, err := g.Deck.Draw()
	if err != nil {
		return "", err
	}

	player.Cards = append(player.Cards, card.Letter)
	return card.Letter, nil
}

func (g *Game) ChangeMainCard() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	newCard := GenerateMainCard()
	oldCard := g.MainCard
	g.MainCard = newCard.Letter
	return oldCard, nil
}

func (g *Game) GetState() *types.GameState {
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]types.PlayerState, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, *p)
	}

	return &types.GameState{
		RoomCode:  g.RoomCode,
		Status:    g.Status,
		Players:   players,
		MainCard:  g.MainCard,
		Timestamp: g.LastPlay.Unix(),
	}
}

func (g *Game) CheckWinner() (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for id, p := range g.Players {
		if len(p.Cards) == 0 {
			return id, true
		}
	}

	if g.Deck != nil && g.Deck.Remaining() == 0 && g.timeSinceLastMove(g.LastPlay) > 60 {
		return findMinCards(g.Players), true
	}

	return "", false
}

func (g *Game) timeSinceLastMove(t time.Time) time.Duration {
	if t.IsZero() {
		return 0
	}
	return time.Since(t)
}

func findMinCards(players map[string]*types.PlayerState) string {
	minCards := 999
	minID := ""
	for id, p := range players {
		if len(p.Cards) < minCards {
			minCards = len(p.Cards)
			minID = id
		}
	}
	return minID
}

func (g *Game) RemainingCards() int {
	if g.Deck == nil {
		return 0
	}
	return g.Deck.Remaining()
}

func (g *Game) IsPlaying() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Status == "playing"
}
