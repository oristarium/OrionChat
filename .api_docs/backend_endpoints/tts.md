# Text-to-Speech (TTS)

## Endpoint Summary

**Endpoint:** `/tts-service`
**Method:** `POST`
**Content-Type:** `application/json`

## Request Body Parameters
```json
{
    "text": "Text to convert to speech",
    "voice_id": "Voice ID for the selected provider",
    "voice_provider": "google|tiktok"
}
```

## Voice Providers and IDs

### 1. Google Translate TTS
Voice IDs are language codes:
- `en` - English
- `es` - Spanish
- `fr` - French
- `de` - German
- `it` - Italian
- `ja` - Japanese
- `ko` - Korean
- `zh-CN` - Chinese (Simplified)
And many more (50+ languages supported)

### 2. TikTok TTS
TikTok provides various voice options with different personalities and styles:

English Voices:
- `en_uk_001` - Narrator (Chris)
- `en_us_002` - Jessie
- `en_us_006` - Joey
- `en_us_007` - Professor
- `en_us_009` - Scientist
- `en_us_010` - Confidence
- `en_female_samc` - Empathetic
- `en_male_cody` - Serious
- `en_male_narration` - Story Teller
- `en_male_funny` - Wacky

Character Voices:
- `en_us_ghostface` - Ghostface (Scream)
- `en_us_chewbacca` - Chewbacca
- `en_us_c3po` - C-3PO
- `en_us_stitch` - Stitch
- `en_us_rocket` - Rocket

Other Languages:
- French: `fr_001`, `fr_002`
- German: `de_001`, `de_002`
- Japanese: `jp_001`, `jp_003`, `jp_005`, `jp_006`
- Korean: `kr_002`, `kr_003`, `kr_004`
- Spanish: `es_002`, `es_mx_002`

For a complete list of TikTok voices with descriptions, see the `tiktok_voice_ids.csv` file.

## Example Usage

```bash
curl -X POST http://localhost:7777/tts-service \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, how are you?",
    "voice_id": "en_male_narration",
    "voice_provider": "tiktok"
  }'
```

## Response
```json
{
    "audio": "base64_encoded_audio_data"
}
```

## Important Notes
1. Maximum text length is 200 characters
2. Long text is automatically split into chunks
3. Text is sanitized before processing
4. The response contains base64-encoded audio data
5. Voice IDs are provider-specific
6. Invalid voice IDs will return a 400 error
7. Each avatar can have multiple TTS voices configured

## Error Responses
- 400: Invalid request body
- 400: Invalid voice ID
- 400: Empty text
- 400: Text too long
- 404: Avatar not found
- 500: Provider errors 