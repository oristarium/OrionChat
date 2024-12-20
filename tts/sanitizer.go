package tts

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
)

// TextSanitizer provides methods for sanitizing text for TTS
type TextSanitizer struct {
	replacements         map[string]string
	providerReplacements map[string]map[string]string
	slangDictionaries    map[string]map[string]string // Maps language code to slang dictionary
	blockedWords        map[string]map[string]bool   // Maps language code to set of blocked words
	loadedLanguages     map[string]bool              // Tracks which languages have been attempted to load
	loadedBlockLists    map[string]bool              // Tracks which block lists have been attempted to load
}

// NewTextSanitizer creates a new sanitizer with default replacements
func NewTextSanitizer() *TextSanitizer {
	return &TextSanitizer{
		replacements: map[string]string{
			"+":  "plus",
			"&":  "and",
			"ä":  "ae",
			"ö":  "oe",
			"ü":  "ue",
			"ß":  "ss",
			"\n": " ", // Convert newlines to spaces
			"\r": " ", // Convert carriage returns to spaces
			"\t": " ", // Convert tabs to spaces
			"$":  "dollar",
			"€":  "euro",
			"£":  "pound",
			"¥":  "yen",
			"@":  "at",
			"#":  "hash",
			"%":  "percent",
			"=":  "equals",
			"*":  "asterisk",
			"~":  "tilde",
			"^":  "caret",
			"<":  "less than",
			">":  "greater than",
			"|":  "pipe",
			"\\": "backslash",
			"\"": "", // Remove quotes
			"'":  "", // Remove single quotes
		},
		providerReplacements: map[string]map[string]string{
			ProviderGoogle: {
				// Google-specific replacements
			},
			ProviderTikTok: {
				// TikTok-specific replacements
			},
		},
		slangDictionaries: make(map[string]map[string]string),
		blockedWords:     make(map[string]map[string]bool),
		loadedLanguages:  make(map[string]bool),
		loadedBlockLists: make(map[string]bool),
	}
}

// loadSlangDictionary loads a slang dictionary from a CSV file
func (s *TextSanitizer) loadSlangDictionary(langCode string) error {
	filepath := filepath.Join("assets", "data", "slang_dict", langCode+".csv")
	
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Mark as attempted to load even if file doesn't exist
			s.loadedLanguages[langCode] = true
			return nil // Return nil as this is an expected case
		}
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	dict := make(map[string]string)

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) >= 2 {
			dict[record[0]] = record[1]
		}
	}

	s.slangDictionaries[langCode] = dict
	s.loadedLanguages[langCode] = true
	return nil
}

// loadBlockedWords loads blocked words from a CSV file
func (s *TextSanitizer) loadBlockedWords(langCode string) error {
	filepath := filepath.Join("assets", "data", "blocked", langCode+".csv")
	
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Mark as attempted to load even if file doesn't exist
			s.loadedBlockLists[langCode] = true
			return nil // Return nil as this is an expected case
		}
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	blockedSet := make(map[string]bool)

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) >= 1 {
			blockedSet[strings.ToLower(record[0])] = true
		}
	}

	s.blockedWords[langCode] = blockedSet
	s.loadedBlockLists[langCode] = true
	return nil
}

// getLanguageFromProvider extracts language code from provider string
func (s *TextSanitizer) getLanguageFromProvider(provider string) string {
	// Extract first two characters as language code
	if len(provider) >= 2 {
		return strings.ToLower(provider[:2])
	}
	return ""
}

// numberWords maps numbers to their word representations
var numberWords = map[string]string{
	"0": "zero",
	"1": "one",
	"2": "two",
	// ... etc
}

// emojiDescriptions maps common emojis to their descriptions
var emojiDescriptions = map[string]string{
	"😊": "smiling face",
	"👍": "thumbs up",
	"❤️": "heart",
	// ... etc
}

// ContainsBlockedWords checks if the text contains any blocked words for the given language
// Returns true and the blocked word if found, false and empty string if not found
func (s *TextSanitizer) ContainsBlockedWords(text, provider string) (bool, string) {
	langCode := s.getLanguageFromProvider(provider)
	if langCode == "" {
		return false, ""
	}

	// Try to load blocked words if not loaded before
	if !s.loadedBlockLists[langCode] {
		if err := s.loadBlockedWords(langCode); err != nil {
			// Log error but continue with processing
			println("Error loading blocked words for", langCode+":", err.Error())
			return false, ""
		}
	}

	// Check if we have blocked words for this language
	blockedSet, ok := s.blockedWords[langCode]
	if !ok {
		return false, ""
	}

	// Convert text to lowercase for case-insensitive comparison
	text = strings.ToLower(text)

	// First, check each word after splitting by whitespace and removing punctuation
	words := strings.Fields(text)
	for _, word := range words {
		// Remove common punctuation from word boundaries
		word = strings.Trim(word, "!@#$%^&*()_+-=[]{}|;:'\",.<>?/~`")
		if blockedSet[word] {
			return true, word
		}
	}

	// Then check for blocked words that might be part of a larger word or have punctuation
	// Create a version of text with punctuation removed
	cleanText := strings.Map(func(r rune) rune {
		if strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:'\",.<>?/~`", r) {
			return ' '
		}
		return r
	}, text)
	
	// Join words without spaces to catch concatenated blocked words
	noSpaceText := strings.Join(strings.Fields(cleanText), "")
	
	for blockedWord := range blockedSet {
		if strings.Contains(cleanText, blockedWord) || strings.Contains(noSpaceText, blockedWord) {
			return true, blockedWord
		}
	}

	return false, ""
}

// Sanitize cleans the text for TTS processing
func (s *TextSanitizer) Sanitize(text string, provider string) string {
	// Apply common replacements first
	var oldNew []string
	for old, new := range s.replacements {
		oldNew = append(oldNew, old, new)
	}

	// Apply provider-specific replacements
	if providerReps, ok := s.providerReplacements[provider]; ok {
		for old, new := range providerReps {
			oldNew = append(oldNew, old, new)
		}
	}

	replacer := strings.NewReplacer(oldNew...)

	// Apply replacements and clean up spaces
	sanitized := replacer.Replace(text)
	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.Join(strings.Fields(sanitized), " ") // Normalize spaces

	// Try to load and apply language-specific slang dictionary if not loaded before
	langCode := s.getLanguageFromProvider(provider)
	if langCode != "" && !s.loadedLanguages[langCode] {
		if err := s.loadSlangDictionary(langCode); err != nil {
			// Log error but continue with processing
			println("Error loading slang dictionary for", langCode+":", err.Error())
		}
	}

	// Apply slang dictionary replacements if available
	if dict, ok := s.slangDictionaries[langCode]; ok {
		words := strings.Fields(sanitized)
		for i, word := range words {
			if replacement, exists := dict[strings.ToLower(word)]; exists {
				words[i] = replacement
			}
		}
		sanitized = strings.Join(words, " ")
	}

	// Handle numbers
	words := strings.Fields(sanitized)
	for i, word := range words {
		if replacement, ok := numberWords[word]; ok {
			words[i] = replacement
		}
	}
	sanitized = strings.Join(words, " ")

	// Replace emojis with descriptions
	for emoji, desc := range emojiDescriptions {
		sanitized = strings.ReplaceAll(sanitized, emoji, desc)
	}

	// Handle URLs
	words = strings.Fields(sanitized)
	for i, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			words[i] = "link"
		}
	}
	sanitized = strings.Join(words, " ")

	return sanitized
}

// WithReplacements adds or updates specific replacements
func (s *TextSanitizer) WithReplacements(replacements map[string]string) *TextSanitizer {
	for k, v := range replacements {
		s.replacements[k] = v
	}
	return s
} 