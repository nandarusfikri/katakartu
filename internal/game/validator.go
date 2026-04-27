package game

import (
	"bufio"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Validator struct {
	dictionary         map[string]bool
	dictionaryPath     string
	dictionaryPathList string
	lastModified       time.Time
	mu                 sync.RWMutex
}

func NewValidator(filename string) (*Validator, error) {
	v := &Validator{
		dictionary:         make(map[string]bool),
		dictionaryPath:     "/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/wordlist.txt",
		dictionaryPathList: "/Users/nandarusfikri/Documents/NandaRusfikri/Labs/Game KataBaku/data/list_1.0.0.txt",
	}
	return v, nil
}

func (v *Validator) loadDictionary() error {
	v.mu.Lock()
	v.dictionary = make(map[string]bool)

	if err := v.loadFromFile(v.dictionaryPath); err != nil {
		log.Printf("Warning: could not load %s: %v", v.dictionaryPath, err)
	}

	if err := v.loadFromFileList(v.dictionaryPathList); err != nil {
		log.Printf("Warning: could not load %s: %v", v.dictionaryPathList, err)
	}

	v.mu.Unlock()
	log.Printf("Dictionary loaded with %d words", len(v.dictionary))
	return nil
}

func (v *Validator) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

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
	return nil
}

func (v *Validator) loadFromFileList(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		word := strings.ToUpper(line)
		if word != "" {
			v.dictionary[word] = true
		}
	}
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
