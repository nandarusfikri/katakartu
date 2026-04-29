package game

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"Game_KataBaku/internal/types"
)

type Game struct {
	RoomCode      string
	Status        string
	Players       map[string]*types.PlayerState
	MainCard      string
	Deck          *Deck
	Validator     *Validator
	LastPlay      time.Time
	LastPlayer    string
	PendingVote   *VoteSession
	Timer         time.Time
	TimerDuration int
	mu            sync.RWMutex
}

type VoteSession struct {
	InitiatorID   string
	InitiatorName string
	NewMainCard   string
	OldMainCard   string
	Approves      map[string]bool
	Rejects       map[string]bool
	TotalPlayers  int
	Deadline      time.Time
}

const DefaultTimerDuration = 5 * 60

func NewGame(roomCode string) *Game {
	return &Game{
		RoomCode:  roomCode,
		Status:    "waiting",
		Players:   make(map[string]*types.PlayerState),
		MainCard:  "",
		Deck:      NewDeck(),
		Validator: nil,
		LastPlay:  time.Time{},
	}
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
	g.MainCard = mainCard.Syllable

	for _, p := range g.Players {
		p.Cards = generatePlayerCards(10)
	}

	g.Deck = NewDeck()
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

func (g *Game) PlayCards(playerID string, prefixCards []string, suffixCards []string) *PlayResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, ok := g.Players[playerID]
	if !ok {
		return &PlayResult{Valid: false, Message: "player not found"}
	}

	allCards := append(prefixCards, suffixCards...)
	if len(allCards) == 0 {
		return &PlayResult{Valid: false, Message: "pilih minimal 1 kartu"}
	}

	if !hasCards(player.Cards, allCards) {
		return &PlayResult{Valid: false, Message: "kartu tidak dimiliki"}
	}

	word := g.buildWord(prefixCards, suffixCards)

	if !g.Validator.IsValid(word) {
		return &PlayResult{Valid: false, Message: "kata tidak valid: " + word}
	}

	player.Score += 10

	var newMainCard string
	if len(suffixCards) > 0 {
		newMainCard = suffixCards[len(suffixCards)-1]
	} else if len(prefixCards) > 0 {
		newMainCard = prefixCards[0]
	} else {
		newMainCard = allCards[len(allCards)-1]
	}

	player.Cards = removeCards(player.Cards, allCards)
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

func (g *Game) buildWord(prefixCards, suffixCards []string) string {
	prefix := strings.Join(prefixCards, "")
	suffix := strings.Join(suffixCards, "")
	return prefix + g.MainCard + suffix
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

func removeCards(hand []string, toRemove []string) []string {
	result := make([]string, len(hand))
	copy(result, hand)

	for _, c := range toRemove {
		for i, card := range result {
			if card == c {
				result = append(result[:i], result[i+1:]...)
				break
			}
		}
	}
	return result
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

	player.Cards = append(player.Cards, card.Syllable)
	return card.Syllable, nil
}

func (g *Game) GetState() *types.GameState {
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]types.PlayerState, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, *p)
	}

	return &types.GameState{
		RoomCode:    g.RoomCode,
		Status:      g.Status,
		Players:     players,
		MainCard:    g.MainCard,
		Leaderboard: g.getLeaderboardLocked(),
		Timer:       g.GetTimeLeft(),
		Timestamp:   g.LastPlay.Unix(),
	}
}

func (g *Game) getLeaderboardLocked() []types.PlayerState {
	players := make([]types.PlayerState, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, *p)
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})

	if len(players) > 5 {
		players = players[:5]
	}
	return players
}

func (g *Game) GetLeaderboard() []types.PlayerState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.getLeaderboardLocked()
}

func (g *Game) GetTimeLeft() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Timer.IsZero() {
		return 0
	}

	remaining := time.Since(g.Timer)
	elapsed := int(remaining.Seconds())
	left := g.TimerDuration - elapsed

	if left < 0 {
		return 0
	}
	return left
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

func generatePlayerCards(count int) []string {
	deck := GetDeckSyllables()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	if count > len(deck) {
		count = len(deck)
	}
	return deck[:count]
}

func (g *Game) CreateVote(initiatorID, initiatorName string) *VoteSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.PendingVote != nil {
		return nil
	}

	newCard := GenerateMainCard()
	g.PendingVote = &VoteSession{
		InitiatorID:   initiatorID,
		InitiatorName: initiatorName,
		NewMainCard:   newCard.Syllable,
		OldMainCard:   g.MainCard,
		Approves:      make(map[string]bool),
		Rejects:       make(map[string]bool),
		TotalPlayers:  len(g.Players),
		Deadline:      time.Now().Add(5 * time.Second),
	}
	return g.PendingVote
}

func (g *Game) ProcessVoteResponse(playerID string, approved bool) (bool, *VoteSession) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.PendingVote == nil {
		return false, nil
	}

	if g.PendingVote.Approves[playerID] || g.PendingVote.Rejects[playerID] {
		return false, nil
	}

	if approved {
		g.PendingVote.Approves[playerID] = true
	} else {
		g.PendingVote.Rejects[playerID] = true
	}

	return true, g.PendingVote
}

func (g *Game) ExecuteVoteIfExpired() *VoteSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.PendingVote == nil {
		return nil
	}

	if time.Now().Before(g.PendingVote.Deadline) {
		return nil
	}

	for id := range g.Players {
		if !g.PendingVote.Approves[id] && !g.PendingVote.Rejects[id] {
			g.PendingVote.Approves[id] = true
		}
	}

	vote := g.PendingVote
	g.PendingVote = nil

	return vote
}

func (g *Game) ExecuteVote(vote *VoteSession) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(vote.Approves) > len(vote.Rejects) {
		g.MainCard = vote.NewMainCard

		for _, p := range g.Players {
			card, err := g.Deck.Draw()
			if err == nil {
				p.Cards = append(p.Cards, card.Syllable)
			}
		}
		return true
	}
	return false
}

func (g *Game) StartTimer() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Timer = time.Now()
	if g.TimerDuration == 0 {
		g.TimerDuration = DefaultTimerDuration
	}
}

func (g *Game) IsTimerExpired() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Timer.IsZero() {
		return false
	}

	remaining := time.Since(g.Timer)
	elapsed := int(remaining.Seconds())
	return elapsed >= g.TimerDuration
}

func (g *Game) ResetTimer() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Timer = time.Time{}
	g.TimerDuration = 0
}

func (g *Game) GetWinner() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	winnerID := ""
	highestScore := -9999

	for id, p := range g.Players {
		if p.Score > highestScore {
			highestScore = p.Score
			winnerID = id
		}
	}

	return winnerID
}
