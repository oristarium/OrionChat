package tts

import "fmt"

// Provider defines the interface for TTS providers
type Provider interface {
	GetAudioBase64(text string, voiceID string, options map[string]interface{}) (string, error)
	GetVoiceIDs() []string
	ValidateVoiceID(voiceID string) bool
}

// ProviderFactory is a map of provider names to their constructor functions
var ProviderFactory = map[string]func() Provider{
	"google": NewGoogleTranslateProvider,
}

// GetProvider returns a TTS provider by name
func GetProvider(name string) (Provider, error) {
	if constructor, exists := ProviderFactory[name]; exists {
		return constructor(), nil
	}
	return nil, fmt.Errorf("provider %s not found", name)
} 