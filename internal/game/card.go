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
	Letter string
}

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	allCards := generateCardPool()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allCards), func(i, j int) {
		allCards[i], allCards[j] = allCards[j], allCards[i]
	})

	return &Deck{Cards: allCards}
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

func generateCardPool() []Card {
	words := loadWordsFromFile()
	
	if len(words) == 0 {
		return generateFallbackCards()
	}
	
	syllables := extractSyllables(words)
	
	rand.Seed(time.Now().UnixNano())
	if len(syllables) < len(syllables) {
		rand.Shuffle(len(syllables), func(i, j int) {
			syllables[i], syllables[j] = syllables[j], syllables[i]
		})
	}
	
	count := 50
	if count > len(syllables) {
		count = len(syllables)
	}
	
	cards := make([]Card, count)
	for i := 0; i < count; i++ {
		cards[i] = Card{Letter: syllables[i]}
	}
	
	return cards
}

func loadWordsFromFile() []string {
	file, err := os.Open("/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/kata.txt")
	if err != nil {
		return []string{}
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && len(word) == 4 {
			words = append(words, strings.ToUpper(word))
		}
	}
	return words
}

func extractSyllables(words []string) []string {
	syllableSet := make(map[string]bool)
	
	for _, word := range words {
		if len(word) == 4 {
			prefix := word[0:2]
			suffix := word[2:4]
			syllableSet[prefix] = true
			syllableSet[suffix] = true
		}
	}
	
	var syllables []string
	for s := range syllableSet {
		syllables = append(syllables, s)
	}
	
	return syllables
}

func generateFallbackCards() []Card {
	pairs := []string{
		"MA", "KA", "SI", "AN", "NG", "EN", "IN", "UN", "AT", "AK",
		"AL", "AR", "AS", "BA", "BE", "BI", "BU", "DA", "DE", "DI",
		"DU", "GA", "GE", "GI", "HA", "HE", "HI", "JA", "JE", "JI",
		"KA", "KE", "KI", "LA", "LE", "LI", "LU", "NA", "NE", "NI",
		"NO", "NU", "PA", "PE", "PI", "PU", "RA", "RE", "RI", "RU",
		"SA", "SE", "SI", "SU", "TA", "TE", "TI", "TU", "YA", "YO",
	}

	cards := make([]Card, len(pairs))
	for i, p := range pairs {
		cards[i] = Card{Letter: p}
	}
	return cards
}

func GenerateMainCard() Card {
	pool := generateCardPool()
	if len(pool) == 0 {
		return Card{Letter: "MA"}
	}
	rand.Seed(time.Now().UnixNano())
	return pool[rand.Intn(len(pool))]
}

func GenerateHelperCards(count int) []Card {
	pool := generateCardPool()
	if count > len(pool) {
		count = len(pool)
	}
	rand.Seed(time.Now().UnixNano())
	
	cards := make([]Card, count)
	indices := rand.Perm(len(pool))[:count]
	for i, idx := range indices {
		cards[i] = pool[idx]
	}
	return cards
}