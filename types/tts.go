package types

// TTSVoice represents a TTS voice configuration
type TTSVoice struct {
    VoiceID  string `json:"voice_id"`
    Provider string `json:"provider"` // "google" or "tiktok"
} 