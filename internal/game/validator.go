package game

import (
	"bufio"
	"os"
	"strings"
)

type Validator struct {
	dictionary map[string]bool
}

func NewValidator(filename string) (*Validator, error) {
	v := &Validator{
		dictionary: make(map[string]bool),
	}

	file, err := os.Open("/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/kata.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && len(word) == 4 {
			v.dictionary[strings.ToUpper(word)] = true
		}
	}

	return v, nil
}

func (v *Validator) IsValid(word string) bool {
	word = strings.ToUpper(strings.TrimSpace(word))
	if len(word) != 4 {
		return false
	}
	return v.dictionary[word]
}

func (v *Validator) Contains(word, sub string) bool {
	word = strings.ToUpper(word)
	sub = strings.ToUpper(sub)
	return strings.Contains(word, sub)
}

func (v *Validator) AddWord(word string) {
	word = strings.ToUpper(word)
	if len(word) == 4 {
		v.dictionary[word] = true
	}
}

func (v *Validator) Count() int {
	return len(v.dictionary)
}