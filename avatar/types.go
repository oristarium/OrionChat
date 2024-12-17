package avatar

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
    CreatedAt   int64                  `json:"created_at"`
}

// AvatarList represents a list of avatars with metadata
type AvatarList struct {
    Avatars     []Avatar `json:"avatars"`
    DefaultID   string   `json:"default_id"`
    CurrentID   string   `json:"current_id"`
} 