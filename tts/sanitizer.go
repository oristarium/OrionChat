package tts

import "strings"

// TextSanitizer provides methods for sanitizing text for TTS
type TextSanitizer struct {
	replacements map[string]string
	providerReplacements map[string]map[string]string
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
	}
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