# Display Message Structure

This document outlines the expected data structure for messages sent to the display endpoint via Server-Sent Events (SSE).

## Message Structure

```javascript
{
    "type": "display",              // Required: Message type identifier
    "data": {                       // Required: Main data container
        "content": {            // Required: Must have at least one of these content fields
            "formatted": string,  // Optional: Formatted message text
            "raw": string,        // Optional: Raw message text
            "sanitized": string   // Optional: Sanitized message text
        },
        "author": {             // Optional: Author information
            "avatar_url": string,     // Optional: URL to author's avatar image
            "badges": [               // Optional: Array of badge objects
                {
                    "image_url": string,  // Badge image URL
                    "label": string       // Badge tooltip/label
                }
            ],
            "display_name": string,   // Optional: Author's display name
            "username": string,       // Optional: Author's username
            "id": string,            // Optional: Author's unique ID
            "roles": {               // Optional: Author's roles
                "broadcaster": boolean,
                "moderator": boolean,
                "subscriber": boolean,
                "verified": boolean
            }
        },
        "metadata": {           // Optional: Additional message metadata
            "type": string,          // e.g., "chat", "super_chat"
            "monetary_data": {       // Optional: For super chats/donations
                "color": string      // Color code for the message background
            }
        }
    },
    "message_id": string,      // Optional: Unique message identifier
    "platform": string,        // Optional: Source platform (e.g., "youtube", "twitch")
    "timestamp": string,       // Optional: Message timestamp
    "type": string            // Optional: Message type (e.g., "chat")
}
```

## Required Fields

The only strictly required fields are:

- `type`: Must be "display"
- `data.data.content`: Must contain at least one of:
  - `formatted`
  - `raw`
  - `sanitized`

## Display Behavior

1. **Content Display**

   - Will use the first available content field in order: formatted → raw → sanitized
   - Empty or whitespace-only content will not be displayed

2. **Author Information**

   - If no author information is provided, displays message-only layout
   - Author section only shows if either display_name or username exists
   - Avatar only shows if avatar_url is provided
   - Badges only show if provided and valid

3. **Styling**
   - Platform-specific styling applied if platform field is present
   - Role-based styling applied if roles are provided
   - Super chat styling applied if metadata.type is "super_chat"

## Example Messages

### Minimal Valid Message

```javascript
{
    "type": "display",
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
    "type": "display",
    "data": {
            "author": {
                "avatar_url": "https://example.com/avatar.jpg",
                "badges": [
                    {
                        "image_url": "https://example.com/badge.png",
                        "label": "Moderator"
                    }
                ],
                "display_name": "DisplayName",
                "username": "username",
                "id": "user123",
                "roles": {
                    "moderator": true
                }
            },
            "content": {
                "formatted": "Hello world!",
                "raw": "Hello world!",
                "sanitized": "Hello world!"
            },
            "metadata": {
                "type": "chat"
            }
        },
        "message_id": "msg123",
        "platform": "youtube",
        "timestamp": "2024-03-18T12:00:00Z",
        "type": "chat"

}
```

## Clear Display Command

To clear the current display, send:

```javascript
{
    "type": "clear_display"
}
```
