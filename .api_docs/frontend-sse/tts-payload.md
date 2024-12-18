# TTS Message Structure

This document outlines the expected data structure for messages sent to the TTS endpoint via Server-Sent Events (SSE).

## Message Structure

```javascript
{
    "type": "tts",                 // Required: Message type identifier
    "data": {                      // Required: Main data container
            "content": {           // Required: Must have sanitized field
                "sanitized": string,     // Required: Clean text for TTS
                "formatted": string,     // Optional: Formatted message text
                "raw": string,          // Optional: Raw message text
                "rawHtml": string,      // Optional: HTML formatted message
                "elements": [           // Optional: Message elements
                    {
                        "type": string,    // Element type (e.g., "text")
                        "value": string,   // Element content
                        "position": [      // Position in message [start, end]
                            number,
                            number
                        ]
                    }
                ]
            },
            "author": {            // Optional: Author information
                "avatar_url": string,    // Optional: URL to author's avatar
                "display_name": string,  // Optional: Author's display name
                "username": string,      // Optional: Author's username
                "id": string,           // Optional: Author's unique ID
                "badges": array,        // Optional: Author's badges
                "roles": {              // Optional: Author's roles
                    "broadcaster": boolean,
                    "moderator": boolean,
                    "subscriber": boolean,
                    "verified": boolean
                }
            },
            "metadata": {          // Optional: Additional message metadata
                "type": string,         // e.g., "chat"
                "monetary_data": null,  // Optional: Payment information
                "sticker": null        // Optional: Sticker information
            }
        },
        "message_id": string,      // Optional: Unique message identifier
        "platform": string,        // Optional: Source platform
        "timestamp": string,       // Optional: Message timestamp (ISO format)
        "type": string,           // Optional: Message type
        "voice_id": string,       // Optional: TTS voice identifier
        "voice_provider": string  // Optional: TTS provider name
    
}
```

## Required Fields

The only strictly required fields are:
- `type`: Must be "tts"
- `data.data.content.sanitized`: The clean text to be read by TTS

## TTS Behavior

1. **Content Handling**
   - Only the sanitized content field is used for TTS
   - Empty or whitespace-only content will be ignored
   - Other content fields (formatted, raw, rawHtml) are for display purposes

2. **Author Display**
   - Author name displays if either display_name or username exists
   - Badges are displayed if available
   - No author information shows a message-only layout
   - Avatar animation works regardless of author info

3. **Message Queue**
   - Messages are processed in order received
   - Duplicate messages (same message_id) are ignored
   - Messages without sanitized content are skipped

## Example Messages

### Minimal Valid Message
```javascript
{
    "type": "tts",
    "data": {
        "content": {
            "sanitized": "Hello world!"
        }
    }
}
```

### Full Message Example
```javascript
{
    "type": "tts",
    "data": {
            "author": {
                "avatar_url": "https://example.com/avatar.jpg",
                "badges": [],
                "display_name": "DisplayName",
                "username": "username",
                "id": "user123",
                "roles": {
                    "broadcaster": false,
                    "moderator": false,
                    "subscriber": false,
                    "verified": false
                }
            },
            "content": {
                "elements": [
                    {
                        "position": [0, 12],
                        "type": "text",
                        "value": "Hello world!"
                    }
                ],
                "formatted": "Hello world!",
                "raw": "Hello world!",
                "rawHtml": "<span class=\"text\">Hello world!</span>",
                "sanitized": "Hello world!"
            },
            "metadata": {
                "monetary_data": null,
                "sticker": null,
                "type": "chat"
            }
        },
        "message_id": "msg123",
        "platform": "youtube",
        "timestamp": "2024-03-18T12:00:00Z",
        "type": "chat",
        "voice_id": "en_male_narration",
        "voice_provider": "tiktok"
    
}
```

## Clear TTS Command

To clear the TTS queue, send:
```javascript
{
    "type": "clear_tts"
}
```

## Notes

- The TTS system only processes the sanitized content to ensure clean and appropriate text-to-speech output
- Author information is used only for display purposes and doesn't affect TTS functionality
- Messages are queued and processed sequentially to prevent overlap
- Voice selection can be specified using voice_id and voice_provider fields
- Elements array provides detailed text position information for highlighting