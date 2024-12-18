# Server-Sent Events (SSE) Broadcasting

The application uses Server-Sent Events (SSE) to push real-time updates to connected clients.

## Endpoints

1. **Connect to SSE Stream**
```http
GET /sse
Response: text/event-stream
```

2. **Send Update**
```http
POST /update
Request: {
    "type": "string",
    "data": object
}
Response: 200 OK
```

## Update Types

Updates are sent as JSON objects with the following structure:
```json
{
    "type": "update_type",
    "data": {
        // Update-specific data
    }
}
```

Common update types include:
- `avatar_state_change`: When an avatar's state changes
- `avatar_created`: When a new avatar is created
- `avatar_deleted`: When an avatar is deleted
- `voice_updated`: When an avatar's voice settings are updated

## Example Usage

1. Connect to SSE stream (Client-side JavaScript):
```javascript
const eventSource = new EventSource('/sse');

eventSource.onmessage = (event) => {
    const update = JSON.parse(event.data);
    console.log('Received update:', update);
};

eventSource.onerror = (error) => {
    console.error('SSE error:', error);
    eventSource.close();
};
```

2. Send an update:
```bash
curl -X POST http://localhost:7777/update \
  -H "Content-Type: application/json" \
  -d '{
    "type": "avatar_state_change",
    "data": {
        "avatar_id": "avatar_123",
        "state": "talking"
    }
}'
```

## Important Notes

1. The SSE connection:
   - Uses standard HTTP/HTTPS
   - Automatically reconnects on disconnection
   - Maintains a persistent connection
   - Is unidirectional (server to client only)

2. Headers set by the server:
   ```http
   Content-Type: text/event-stream
   Cache-Control: no-cache
   Connection: keep-alive
   Access-Control-Allow-Origin: *
   ```

3. Message format:
   ```
   data: {"type":"update_type","data":{...}}\n\n
   ```

4. Error handling:
   - Client disconnections are automatically detected
   - Resources are cleaned up when clients disconnect
   - Failed broadcasts are logged but don't affect other clients

## PowerShell Example

```powershell
$body = @{
    type = "avatar_state_change"
    data = @{
        avatar_id = "avatar_123"
        state = "talking"
    }
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri "http://localhost:7777/update" `
    -Body $body -ContentType "application/json"
``` 