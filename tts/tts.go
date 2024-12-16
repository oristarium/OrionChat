package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultHost    = "https://translate.google.com"
	maxTextLength  = 200
	defaultTimeout = 10000 // ms
)

// TTSService handles text-to-speech conversion
type TTSService struct {
	host    string
	client  *http.Client
	timeout int
}

// NewTTSService creates a new TTS service instance
func NewTTSService() *TTSService {
	return &TTSService{
		host:    defaultHost,
		client:  &http.Client{},
		timeout: defaultTimeout,
	}
}

// GetAudioBase64 converts text to speech and returns base64 encoded audio
func (s *TTSService) GetAudioBase64(text, lang string, slow bool) (string, error) {
	log.Printf("TTS Request - Text: %q, Lang: %q, Slow: %v", text, lang, slow)
	if len(text) > maxTextLength {
		log.Printf("Text too long: %d characters (max: %d)", len(text), maxTextLength)
		return "", fmt.Errorf("text length (%d) should be less than %d characters", len(text), maxTextLength)
	}

	// Format exactly like the TypeScript version
	innerData := []interface{}{text, lang, slow, "null"}
	innerJSON, err := json.Marshal(innerData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inner data: %v", err)
	}

	data := [][3]interface{}{
		{
			"jQ1olc",
			string(innerJSON),
			nil,
		},
	}

	outerData := []interface{}{data}
	outerJSON, err := json.Marshal(outerData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal outer data: %v", err)
	}

	formData := url.Values{}
	formData.Set("f.req", string(outerJSON))

	// Create request
	req, err := http.NewRequest("POST", 
		s.host+"/_/TranslateWebserverUi/data/batchexecute",
			strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://translate.google.com")
	req.Header.Set("Referer", "https://translate.google.com/")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("x-client-data", "CIa2yQEIpbbJAQipncoBCMKTywEIkqHLAQiFoM0BCJyrzQEI2rHNAQjcuM0BCNy4zQEI/rnNAQiWus0BCMq6zQEI1rrNAQjpu80BCIPKzQEIhcrNAQ==")
	log.Printf("Sending request to: %s", req.URL.String())

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %s", resp.Status)
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %v", err)
	}
	log.Printf("Response body: %s", string(body))

	// Parse like the TypeScript version
	responseStr := string(body)
	if len(responseStr) < 6 {
		log.Printf("Invalid response format (length: %d)", len(responseStr))
		return "", fmt.Errorf("invalid response format")
	}

	// Remove prefix ")]}'"\n
	responseStr = responseStr[5:]

	var result interface{}
	if err := json.Unmarshal([]byte(responseStr), &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract like result = eval(res.data.slice(5))[0][2]
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) == 0 {
		return "", fmt.Errorf("invalid response format")
	}

	firstArray, ok := resultArray[0].([]interface{})
	if !ok || len(firstArray) < 3 {
		return "", fmt.Errorf("invalid response structure")
	}

	audioData := firstArray[2]
	if audioData == nil {
		return "", fmt.Errorf("no audio data found")
	}

	// Parse the audio data string
	audioStr, ok := audioData.(string)
	if !ok {
		return "", fmt.Errorf("invalid audio data format")
	}

	// Parse like result = eval(result)[0]
	var finalData []interface{}
	if err := json.Unmarshal([]byte(audioStr), &finalData); err != nil {
		return "", fmt.Errorf("failed to parse audio data: %v", err)
	}

	if len(finalData) == 0 {
		return "", fmt.Errorf("no audio data in final parse")
	}

	base64Audio, ok := finalData[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid base64 audio format")
	}

	log.Printf("Successfully generated audio (base64 length: %d)", len(base64Audio))
	return base64Audio, nil
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
		if currentChunk.Len()+len(word)+1 > maxTextLength {
			log.Printf("Current chunk would exceed max length with word: %q", word)
			// If current chunk is not empty, add it to chunks
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				log.Printf("Added chunk: %q", chunks[len(chunks)-1])
				currentChunk.Reset()
			}
			
			// If single word is longer than maxTextLength, split it
			if len(word) > maxTextLength {
				log.Printf("Word too long: %q (length: %d)", word, len(word))
				// Split by punctuation if provided
				if splitPunct != "" {
					parts := strings.FieldsFunc(word, func(r rune) bool {
						return strings.ContainsRune(splitPunct, r)
					})
					log.Printf("Split long word into %d parts", len(parts))
					for _, part := range parts {
						if len(part) > maxTextLength {
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