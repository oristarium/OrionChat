package tts

import (
	"bytes"
	"fmt"
	"log"
	"strings"
)

// TTSService handles text-to-speech conversion
type TTSService struct {
	provider Provider
}

// NewTTSService creates a new TTS service instance
func NewTTSService() *TTSService {
	provider, err := GetProvider("google") // Default to Google provider
	if err != nil {
		log.Printf("Error creating default provider: %v", err)
		return nil
	}
	
	return &TTSService{
		provider: provider,
	}
}

// GetAudioBase64 converts text to speech and returns base64 encoded audio
func (s *TTSService) GetAudioBase64(text, voiceID string, slow bool) (string, error) {
	options := map[string]interface{}{
		"slow": slow,
	}
	
	if !s.provider.ValidateVoiceID(voiceID) {
		return "", fmt.Errorf("invalid voice ID: %s", voiceID)
	}
	
	return s.provider.GetAudioBase64(text, voiceID, options)
}

// SplitLongText splits text into chunks that are less than maxTextLength
func (s *TTSService) SplitLongText(text string, splitPunct string) ([]string, error) {
	log.Printf("Splitting text (length: %d): %q", len(text), text)
	var chunks []string
	words := strings.Fields(text)
	log.Printf("Split into %d words", len(words))
	
	currentChunk := new(bytes.Buffer)
	
	for _, word := range words {
		// Check if adding this word would exceed maxTextLength
		if currentChunk.Len()+len(word)+1 > MaxTextLength {
			log.Printf("Current chunk would exceed max length with word: %q", word)
			// If current chunk is not empty, add it to chunks
			if currentChunk.Len() > 0 {
				
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				log.Printf("Added chunk: %q", chunks[len(chunks)-1])
				currentChunk.Reset()
			}
			
			// If single word is longer than maxTextLength, split it
			if len(word) > MaxTextLength {
				log.Printf("Word too long: %q (length: %d)", word, len(word))
				// Split by punctuation if provided
				if splitPunct != "" {
					parts := strings.FieldsFunc(word, func(r rune) bool {
						return strings.ContainsRune(splitPunct, r)
					})
					log.Printf("Split long word into %d parts", len(parts))
					for _, part := range parts {
						if len(part) > MaxTextLength {
							log.Printf("Part still too long: %q (length: %d)", part, len(part))
							return nil, fmt.Errorf("word too long to split: %s", part)
						}
						chunks = append(chunks, part)
						log.Printf("Added part as chunk: %q", part)
					}
				} else {
					return nil, fmt.Errorf("word too long: %s", word)
				}
			} else {
				currentChunk.WriteString(word)
			}
		} else {
			// Add space if chunk is not empty
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(word)
		}
	}

	// Add final chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
		log.Printf("Added final chunk: %q", chunks[len(chunks)-1])
	}

	log.Printf("Split text into %d chunks: %v", len(chunks), chunks)
	return chunks, nil
} 