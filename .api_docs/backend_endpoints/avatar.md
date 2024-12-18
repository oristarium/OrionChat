# Avatar Management

## Endpoints

1. **List All Avatars**
```http
GET /api/avatars
Response: {
    "avatars": [
        {
            "id": "avatar_1734429111883435200",
            "name": "Default",
            "description": "Default avatar",
            "states": {
                "idle": "/avatars/idle.png",
                "talking": "/avatars/talking.gif"
            },
            "is_default": true,
            "is_active": true,
            "created_at": 1710691200,
            "tts_voices": [
                {
                    "voice_id": "en_female_emotional",
                    "provider": "tiktok"
                },
                {
                    "voice_id": "en_au_001",
                    "provider": "tiktok"
                }
            ]
        }
    ],
    "current_id": "avatar_1734429111883435200"
}
```

2. **Get Single Avatar**
```http
GET /api/avatars/{id}/get
Response: {
    "id": "avatar_1734429111883435200",
    "name": "Custom Avatar",
    "description": "My custom avatar",
    "states": {
        "idle": "/avatars/custom_idle.png",
        "talking": "/avatars/custom_talking.gif"
    },
    "is_default": false,
    "is_active": false,
    "created_at": 1710691200,
    "tts_voices": [
        {
            "voice_id": "en_female_emotional",
            "provider": "tiktok"
        }
    ]
}
```

3. **Update Avatar States**
```http
PUT /api/avatars/{id}/set
Request: {
    "states": {
        "idle": "/avatars/new_idle.png",
        "talking": "/avatars/new_talking.gif"
    }
}
Response: 200 OK
```

4. **Delete Avatar**
```http
DELETE /api/avatars/{id}/delete
Response: 200 OK
Error Cases:
- 400: Cannot delete default avatar
- 404: Avatar not found
```

5. **List Active Avatars**
```http
GET /api/avatars/active
Response: {
    "avatars": [
        {
            "id": "avatar_1734429111883435200",
            "name": "Active Avatar",
            "description": "Currently active avatar",
            "states": {
                "idle": "/avatars/active_idle.png",
                "talking": "/avatars/active_talking.gif"
            },
            "is_default": false,
            "is_active": true,
            "created_at": 1710691200,
            "tts_voices": [
                {
                    "voice_id": "en_female_emotional",
                    "provider": "tiktok"
                }
            ]
        }
    ]
}
```

6. **Create New Avatar**
```http
POST /api/avatars/create
Response: {
    "id": "avatar_1234567890",
    "name": "New Avatar",
    "description": "New avatar",
    "states": {
        "idle": "/avatars/idle.png",
        "talking": "/avatars/talking.gif"
    },
    "is_default": false,
    "is_active": false,
    "created_at": 1710691200,
    "tts_voices": null
}
``` 