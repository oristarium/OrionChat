# Key-Value Storage API

The key-value storage API provides a simple way to store and retrieve both JSON and plain text data using a key-based system.

## Base URL
All endpoints are relative to: `http://localhost:7777`

## Endpoints

### Store Value
```http
PUT /api/kv/{key}
```

Stores a value associated with the specified key.

#### Headers
- `Content-Type`: (Required)
  - `application/json` for JSON data
  - `text/plain` for plain text data

#### Path Parameters
- `key` (Required): The unique identifier for storing the value

#### Request Examples

**Storing JSON:**
```bash
curl -X PUT \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "age": 30}' \
  http://localhost:7777/api/kv/user1
```

**Storing Plain Text:**
```bash
curl -X PUT \
  -H "Content-Type: text/plain" \
  -d "Hello World" \
  http://localhost:7777/api/kv/greeting
```

#### Responses

| Status Code | Description |
|------------|-------------|
| 200 | Value successfully stored |
| 400 | Bad request (invalid JSON or missing key) |
| 500 | Internal server error |

### Retrieve Value
```http
GET /api/kv/{key}
```

Retrieves the value associated with the specified key.

#### Path Parameters
- `key` (Required): The unique identifier of the stored value

#### Response Headers
The response will automatically set the appropriate Content-Type:
- `application/json` for JSON data
- `text/plain` for plain text data

#### Request Example
```bash
curl http://localhost:7777/api/kv/user1
```

#### Responses

| Status Code | Description | Example Response |
|------------|-------------|------------------|
| 200 | Success | `{"name": "John", "age": 30}` or `Hello World` |
| 404 | Key not found | `Key not found` |
| 500 | Internal server error | Error message |

## Storage Details
- Values are stored in a BBolt database
- Uses a dedicated 'general' bucket for key-value pairs
- JSON validation is performed when Content-Type is application/json
- JSON formatting is preserved between storage and retrieval
- Plain text is stored and retrieved as-is

## Error Handling
The API returns appropriate HTTP status codes and error messages:

| Status Code | Scenario |
|------------|----------|
| 400 | - Missing key<br>- Invalid JSON format when Content-Type is application/json |
| 404 | Requested key doesn't exist |
| 405 | Method other than GET or PUT used |
| 500 | Database or internal server errors |

## Notes
- Keys are case-sensitive
- The API automatically detects JSON content on retrieval
- Values are stored in their raw format
- No size limits are explicitly set (governed by BBolt database limits)
