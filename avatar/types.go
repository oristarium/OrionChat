package avatar

import "github.com/oristarium/orionchat/types"

// AvatarState represents the possible states of an avatar
type AvatarState string

const (
    StateIdle    AvatarState = "idle"
    StateTalking AvatarState = "talking"
)

// Avatar represents a complete avatar with all its states
type Avatar struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    States      map[AvatarState]string `json:"states"` // maps state to file path
    IsDefault   bool                   `json:"is_default"`
    IsActive    bool                   `json:"is_active"`
    CreatedAt   int64                  `json:"created_at"`
    TTSVoices   []types.TTSVoice      `json:"tts_voices"`
    SortOrder   int                    `json:"sort_order"`
}

// AvatarList represents a list of avatars with metadata
type AvatarList struct {
    Avatars     []Avatar `json:"avatars"`
    // HasDefault  bool     `json:"has_default"`
}

// AvatarImage represents a physical image file that can be used as an avatar state
type AvatarImage struct {
    Path      string `json:"path"`
    Type      string `json:"type"` // file type (png, gif, etc)
    CreatedAt int64  `json:"created_at"`
} 