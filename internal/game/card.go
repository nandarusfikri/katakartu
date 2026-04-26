package game

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type Card struct {
	Syllable string
}

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	syllables := loadDeckFromFile()
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

func loadDeckFromFile() []string {
	file, err := os.Open("/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/deck.txt")
	if err != nil {
		return generateFallbackSyllables()
	}
	defer file.Close()

	var allSyllables []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
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

func GenerateMainCard() Card {
	pool := loadDeckFromFile()
	if len(pool) == 0 {
		return Card{Syllable: "MA"}
	}
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(pool))
	return Card{Syllable: pool[idx]}
}

func GetDeckSyllables() []string {
	return loadDeckFromFile()
}
