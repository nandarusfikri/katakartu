package game

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"
)

type Validator struct {
	dictionary     map[string]bool
	dictionaryPath string
	lastModified   time.Time
	mu             sync.RWMutex
}

func NewValidator(filename string) (*Validator, error) {
	v := &Validator{
		dictionary:     make(map[string]bool),
		dictionaryPath: "/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/wordlist.txt",
	}
	return v, nil
}

func (v *Validator) loadDictionary() error {
	file, err := os.Open(v.dictionaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	v.mu.Lock()
	v.dictionary = make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		words := strings.Split(line, ",")
		for _, word := range words {
			word = strings.TrimSpace(word)
			if word != "" {
				v.dictionary[strings.ToUpper(word)] = true
			}
		}
	}
	v.mu.Unlock()

	return nil
}

func (v *Validator) ReloadIfChanged() {
	info, err := os.Stat(v.dictionaryPath)
	if err != nil {
		return
	}

	if v.lastModified.IsZero() || !info.ModTime().Equal(v.lastModified) {
		v.lastModified = info.ModTime()
		v.loadDictionary()
	}
}

func (v *Validator) IsValid(word string) bool {
	v.ReloadIfChanged()

	word = strings.ToUpper(strings.TrimSpace(word))
	if word == "" {
		return false
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.dictionary[word]
}

func (v *Validator) Contains(word, sub string) bool {
	word = strings.ToUpper(word)
	sub = strings.ToUpper(sub)
	return strings.Contains(word, sub)
}

func (v *Validator) AddWord(word string) {
	word = strings.ToUpper(word)
	if word != "" {
		v.mu.Lock()
		defer v.mu.Unlock()
		v.dictionary[word] = true
	}
}

func (v *Validator) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.dictionary)
}
