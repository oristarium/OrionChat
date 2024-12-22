# Avatar Images

## Endpoints

1. **List Avatar Images**
```http
GET /api/avatar-images
Response: {
    "avatar-images": [
        "/avatars/idle.png",
        "/avatars/talking.gif",
        "/avatars/custom_idle.png",
        "/avatars/custom_talking.gif"
    ]
}
```

2. **Upload Avatar Image**
```http
POST /api/avatar-images/upload
Request: multipart/form-data
- avatar: file
- type: string (idle/talking)

Response: {
    "path": "/avatars/1734429234567890.png"
}
```

3. **Delete Avatar Image**
```http
DELETE /api/avatar-images/delete
Request: application/json
{
    "path": "/avatars/1734429234567890.png"
}

Response: 200 OK
Error Cases:
- 400: Path is required
- 400: Image is in use by avatar {id}
- 404: Image not found
```

## Usage Example

For updating an avatar's state (like setting a new idle image), follow these steps:

1. First, upload the new image:
```http
POST /api/avatar-images/upload
Content-Type: multipart/form-data
- avatar: [file]
- type: "idle"
```

2. After successful upload, you'll get a response with the path:
```json
{
    "path": "/avatars/1734429234567890.png"
}
```

3. Then update the avatar state:
```http
PUT /api/avatars/{id}/set
Content-Type: application/json
{
    "states": {
        "idle": "/avatars/1734429234567890.png"
    }
}
```

Note: The backend handles:
1. Saving the uploaded file to `/assets/avatars/`
2. Registering the image in the database
3. Updating the avatar's state in the database
4. Broadcasting the update to connected clients 