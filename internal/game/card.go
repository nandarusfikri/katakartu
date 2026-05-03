package game

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Card struct {
	Syllable string
}

type Deck struct {
	Cards []Card
}

func NewDeck(level string) *Deck {
	syllables := loadDeckFromFile(level)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(syllables), func(i, j int) {
		syllables[i], syllables[j] = syllables[j], syllables[i]
	})

	cards := make([]Card, len(syllables))
	for i, s := range syllables {
		cards[i] = Card{Syllable: s}
	}

	return &Deck{Cards: cards}
}

func (d *Deck) Draw() (Card, error) {
	if len(d.Cards) == 0 {
		return Card{}, fmt.Errorf("deck empty")
	}
	card := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return card, nil
}

func (d *Deck) Remaining() int {
	return len(d.Cards)
}

func loadDeckFromFile(level string) []string {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}

	// Determine deck file based on level
	deckFile := "deck.txt"
	if level == "medium" {
		deckFile = "deck_medium.txt"
	}
	deckPath := filepath.Join(baseDir, "data", deckFile)
	file, err := os.Open(deckPath)
	if err != nil {
		// Fallback to deck.txt if level-specific file not found
		if level == "medium" {
			deckPath = filepath.Join(baseDir, "data", "deck.txt")
			file, err = os.Open(deckPath)
			if err != nil {
				return generateFallbackSyllables()
			}
		} else {
			return generateFallbackSyllables()
		}
	}
	defer file.Close()

	var allSyllables []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		for _, p := range parts {
			syllable := strings.TrimSpace(p)
			if syllable != "" {
				allSyllables = append(allSyllables, strings.ToUpper(syllable))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return generateFallbackSyllables()
	}

	if len(allSyllables) == 0 {
		return generateFallbackSyllables()
	}

	return allSyllables
}

func generateFallbackSyllables() []string {
	return []string{
		"BA", "MA", "KA", "LA", "RA", "SA", "TA", "PA", "NA", "DA",
		"AN", "IN", "UN", "EN", "ON", "NG",
		"ANG", "ING", "UNG", "ENG", "ONG",
		"BER", "TER", "PER", "KAN", "KIN",
	}
}

func GenerateMainCard(level string) Card {
	pool := loadDeckFromFile(level)
	if len(pool) == 0 {
		return Card{Syllable: "MA"}
	}
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(pool))
	return Card{Syllable: pool[idx]}
}

func GetDeckSyllables(level string) []string {
	return loadDeckFromFile(level)
}
