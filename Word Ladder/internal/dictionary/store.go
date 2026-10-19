package dictionary

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

type Store struct {
	words               map[string]bool  // Full word dictionary for move validation
	commonWordsByLength map[int][]string // Most common words used for start/target words
	rng                 *rand.Rand
	rngMu               sync.Mutex
}

var (
	instance *Store
	once     sync.Once
)

func GetInstance(filePath string) (*Store, error) {
	var loadErr error
	once.Do(func() {
		// read full dictionary for validation
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			loadErr = fmt.Errorf("failed to read validation file '%s': %w", filePath, err)
			return
		}

		rawMap := make(map[string]int)
		if err := json.Unmarshal(fileBytes, &rawMap); err != nil {
			loadErr = fmt.Errorf("failed to parse validation JSON '%s': %w", filePath, err)
			return
		}
		words := make(map[string]bool, len(rawMap))
		for word := range rawMap {
			words[word] = true
		}

		// read common words for start/target selection
		commonFileName := "top_10000_english_words_usa.json"
		commonBytes, err := os.ReadFile(commonFileName)
		if err != nil {
			loadErr = fmt.Errorf("CRITICAL ERROR: Could not find '%s' in current working directory. Ensure you run the command from the project root folder: %w", commonFileName, err)
			return
		}

		var commonList []string
		if err := json.Unmarshal(commonBytes, &commonList); err != nil {
			var commonMap map[string]interface{}
			if mapErr := json.Unmarshal(commonBytes, &commonMap); mapErr != nil {
				loadErr = fmt.Errorf("failed to parse '%s' as array or object map: %w", commonFileName, err)
				return
			}
			for k := range commonMap {
				commonList = append(commonList, k)
			}
		}

		if len(commonList) == 0 {
			loadErr = fmt.Errorf("file '%s' loaded 0 words - check file content", commonFileName)
			return
		}

		// Cap pool to the top 2,500 words to guarantee recognizable vocabulary
		maxPoolSize := 2500
		if len(commonList) < maxPoolSize {
			maxPoolSize = len(commonList)
		}

		commonByLen := make(map[int][]string)
		maxPerLength := 200

		for i := 0; i < maxPoolSize; i++ {
			w := strings.ToUpper(strings.TrimSpace(commonList[i]))
			if _, exists := rawMap[strings.ToLower(w)]; exists && isAlpha(w) {
				l := len(w)
				if len(commonByLen[l]) < maxPerLength {
					commonByLen[l] = append(commonByLen[l], w)
				}
			}
		}

		for l, list := range commonByLen {
			log.Printf("[Dictionary] Loaded %d common words for length %d", len(list), l)
		}

		instance = &Store{
			words:               words,
			commonWordsByLength: commonByLen,
			rng:                 rand.New(rand.NewSource(time.Now().UnixNano())),
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}
	return instance, nil
}

// IsValid checks if a user input exists in the full validation dictionary
func (s *Store) IsValid(word string) bool {
	if s == nil || s.words == nil {
		return false
	}
	return s.words[strings.ToLower(word)]
}

// GetRandomPair picks two distinct words from the top 2,500 common words list
func (s *Store) GetRandomPair(length int) (string, string, bool) {
	list, exists := s.commonWordsByLength[length]
	if !exists || len(list) < 2 {
		return "", "", false
	}

	s.rngMu.Lock()
	defer s.rngMu.Unlock()

	idx1 := s.rng.Intn(len(list))
	idx2 := s.rng.Intn(len(list))
	for idx1 == idx2 {
		idx2 = s.rng.Intn(len(list))
	}

	return list[idx1], list[idx2], true
}

func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
