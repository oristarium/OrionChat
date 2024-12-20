package types

import (
	"mime/multipart"
)

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
    TTSVoices   []TTSVoice            `json:"tts_voices"`
    SortOrder   int                    `json:"sort_order"`
}

// AvatarList represents a list of avatars with metadata
type AvatarList struct {
    Avatars     []Avatar `json:"avatars"`
}

// AvatarImage represents a physical image file that can be used as an avatar state
type AvatarImage struct {
    Path      string `json:"path"`
    Type      string `json:"type"` // file type (png, gif, etc)
    CreatedAt int64  `json:"created_at"`
}

// Config holds server configuration
type Config struct {
	Port      string
	AssetsDir string
	Routes    map[string]string
}

// ChatAuthor represents the author of a chat message
type ChatAuthor struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Roles       struct {
		Broadcaster bool `json:"broadcaster"`
		Moderator   bool `json:"moderator"`
		Subscriber  bool `json:"subscriber"`
		Verified    bool `json:"verified"`
	} `json:"roles"`
	Badges []struct {
		Type     string `json:"type"`
		Label    string `json:"label"`
		ImageURL string `json:"image_url"`
	} `json:"badges"`
}

// ChatContent represents the content of a chat message
type ChatContent struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
	Sanitized string `json:"sanitized"`
	RawHTML   string `json:"rawHtml"`
	Elements  []struct {
		Type     string `json:"type"`
		Value    string `json:"value"`
		Position []int  `json:"position"`
		Metadata *struct {
			URL      string `json:"url"`
			Alt      string `json:"alt"`
			IsCustom bool   `json:"is_custom"`
		} `json:"metadata,omitempty"`
	} `json:"elements"`
}

// ChatMetadata represents metadata for a chat message
type ChatMetadata struct {
	Type         string `json:"type"`
	MonetaryData *struct {
		Amount    string `json:"amount"`
		Formatted string `json:"formatted"`
		Color     string `json:"color"`
	} `json:"monetary_data,omitempty"`
	Sticker *struct {
		URL string `json:"url"`
		Alt string `json:"alt"`
	} `json:"sticker,omitempty"`
}

// ChatMessageData represents the inner data of a chat message
type ChatMessageData struct {
	Author   ChatAuthor   `json:"author"`
	Content  ChatContent  `json:"content"`
	Metadata ChatMetadata `json:"metadata"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Type      string         `json:"type"`
	Platform  string         `json:"platform"`
	Timestamp string         `json:"timestamp"`
	MessageID string         `json:"message_id"`
	RoomID    string         `json:"room_id"`
	Data      ChatMessageData `json:"data"`
}

type SSEClient chan string

type File = multipart.File

// FileStorage interface defines methods for file operations
type FileStorage interface {
	Upload(file *File, filename string, directory string) (string, error)
	Get(key string, bucket string) (string, error)
	Save(key string, value string, bucket string) error
} 

type TTSVoice struct {
    VoiceID  string `json:"voice_id"`
    Provider string `json:"provider"` // "google" or "tiktok"
} 