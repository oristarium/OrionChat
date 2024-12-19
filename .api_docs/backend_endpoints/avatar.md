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
            ],
            "sort_order": 0
        }
    ]
}
```
Note: Avatars are returned sorted by their `sort_order` field in ascending order. If multiple avatars have the same `sort_order`, they are sorted by `created_at`.

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
    ],
    "sort_order": 1
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
            ],
            "sort_order": 2
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
    "tts_voices": null,
    "sort_order": 3
}
```
Note: New avatars are automatically assigned a `sort_order` value of (highest existing sort_order + 1)

7. **Update Avatar Sort Order**
```http
PUT /api/avatars/{id}/sort
Request: {
    "sort_order": 5
}
Response: 200 OK
Error Cases:
- 400: Invalid request body
- 404: Avatar not found
```
Note: When updating sort order, the change is immediately broadcasted to all connected clients via the SSE connection.

## SSE Broadcasts

The server sends Server-Sent Events (SSE) to notify clients of avatar updates. Connect to the `/sse` endpoint to receive these events.

1. **Avatar Update Broadcast**
```http
event: avatar_update
data: {
    "type": "avatar_update",
    "data": {
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
                ],
                "sort_order": 5
            }
            // ... other active avatars
        ]
    }
}
```
Note: The `avatar_update` event is sent whenever an avatar is created, updated, deleted, or when sort orders change. The data contains only the active avatars in their current sorted order.
``` 