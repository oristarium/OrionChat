# Avatar Voices

## Endpoints

1. **Get Avatar TTS Voices**
```http
GET /api/avatars/{id}/voices
Response: {
    "voices": [
        {
            "voice_id": "en_male_narration",
            "provider": "tiktok"
        },
        {
            "voice_id": "en",
            "provider": "google"
        }
    ]
}
Error Cases:
- 404: Avatar not found
```

2. **Update Avatar TTS Voices**
```http
PUT /api/avatars/{id}/voices
Request: {
    "voices": [
        {
            "voice_id": "en_male_narration",
            "provider": "tiktok"
        },
        {
            "voice_id": "en",
            "provider": "google"
        }
    ]
}
Response: 200 OK
Error Cases:
- 400: Invalid request body
- 404: Avatar not found
- 500: Failed to save avatar
```

## Example Usage

1. Get voices for an avatar:
```bash
curl -X GET http://localhost:7777/api/avatars/avatar_123/voices
```

2. Set voices for an avatar:
```bash
curl -X PUT http://localhost:7777/api/avatars/avatar_123/voices \
  -H "Content-Type: application/json" \
  -d '{
    "voices": [
        {
            "voice_id": "en_male_narration",
            "provider": "tiktok"
        },
        {
            "voice_id": "en",
            "provider": "google"
        }
    ]
}'
```

### PowerShell Example
```powershell
# Get voices
Invoke-RestMethod -Method Get -Uri "http://localhost:7777/api/avatars/avatar_123/voices"

# Set voices
$body = @{
    voices = @(
        @{
            voice_id = "en_male_narration"
            provider = "tiktok"
        },
        @{
            voice_id = "en"
            provider = "google"
        }
    )
} | ConvertTo-Json

Invoke-RestMethod -Method Put -Uri "http://localhost:7777/api/avatars/avatar_123/voices" `
    -Body $body -ContentType "application/json"
``` 