class TTSManager {
    constructor() {
        this.messageQueue = new Map();
        this.isProcessing = false;
    }

    async processTTSMessage(message, avatar) {
        // Check if this is a duplicate message
        const messageId = message.message_id;
        if (this.messageQueue.has(messageId)) {
            console.log('Duplicate TTS message, skipping:', messageId);
            return;
        }

        // Get the voice for this message
        const voice = avatar.getRandomVoice();
        if (!voice || !voice.voice_id || !voice.provider) {
            console.error('Invalid voice configuration:', voice);
            console.error('Avatar:', avatar.id, 'TTS voices:', avatar.tts_voices);
            return;
        }

        // Log the voice selection
        console.log('Using voice:', {
            id: voice.voice_id,
            provider: voice.provider,
            avatar: avatar.id
        });

        // Update message display
        avatar.updateMessageContent(message);

        try {
            // Create TTS request
            const response = await fetch('/tts-service', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    text: message.data.content.sanitized,
                    voice_id: voice.voice_id,
                    voice_provider: voice.provider
                })
            });

            if (!response.ok) {
                throw new Error(`TTS request failed: ${response.statusText}`);
            }

            const data = await response.json();
            
            // Create audio URL from base64 data
            const audioUrl = `data:audio/mp3;base64,${data.audio}`;
            
            // Add to queue
            this.messageQueue.set(messageId, {
                audioUrl,
                avatar,
                message
            });

            // Process queue if not already processing
            if (!this.isProcessing) {
                this.processQueue();
            }
        } catch (error) {
            console.error('Error getting TTS audio:', error);
        }
    }

    async processQueue() {
        if (this.messageQueue.size === 0 || this.isProcessing) {
            return;
        }

        this.isProcessing = true;
        const [messageId, queueItem] = this.messageQueue.entries().next().value;
        const { audioUrl, avatar, message } = queueItem;

        try {
            // Set the audio source
            avatar.audioPlayer.src = audioUrl;
            
            // Play the audio
            await avatar.audioPlayer.play();

            // Wait for audio to finish
            await new Promise((resolve) => {
                avatar.audioPlayer.onended = resolve;
            });

        } catch (error) {
            console.error('Error processing TTS message:', error);
        } finally {
            // Remove from queue
            this.messageQueue.delete(messageId);
            
            // Update debug info if available
            if (window.ELEMENTS?.$debugQueueCount) {
                window.ELEMENTS.$debugQueueCount.text(this.messageQueue.size);
                window.ELEMENTS.$debugQueueStatus.text(this.isProcessing ? 'Processing' : 'Idle');
            }

            // Reset processing flag
            this.isProcessing = false;

            // Process next item if any
            if (this.messageQueue.size > 0) {
                this.processQueue();
            }
        }
    }

    clearQueue() {
        this.messageQueue.clear();
        if (window.ELEMENTS?.$debugQueueCount) {
            window.ELEMENTS.$debugQueueCount.text('0');
            window.ELEMENTS.$debugQueueStatus.text('Idle');
        }
    }
}

// Export the TTSManager
window.TTSManager = TTSManager; 