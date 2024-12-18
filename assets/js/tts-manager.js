class TTSManager {
    constructor() {
        this.messageQueue = new Map();
        this.isProcessing = false;
    }

    async processTTSMessage(message, avatar) {
        // Validate required fields according to docs
        if (!message.data?.content?.sanitized) {
            console.error('Invalid TTS message structure - missing required fields');
            return;
        }

        // Check if this is a duplicate message
        const messageId = message.message_id;
        if (messageId && this.messageQueue.has(messageId)) {
            console.log('Duplicate TTS message, skipping:', messageId);
            return;
        }

        // Skip empty content
        const sanitizedText = message.data.content.sanitized.trim();
        if (!sanitizedText) {
            console.log('Empty TTS content, skipping');
            return;
        }

        try {
            // Create TTS request using provided voice info or avatar's random voice
            const voiceConfig = {
                text: sanitizedText,
                voice_id: message.voice_id || avatar.getRandomVoice()?.voice_id,
                voice_provider: message.voice_provider || avatar.getRandomVoice()?.provider
            };

            if (!voiceConfig.voice_id || !voiceConfig.voice_provider) {
                console.error('No valid voice configuration available');
                return;
            }

            // Update message display
            avatar.updateMessageContent(message);

            const response = await fetch('/tts-service', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(voiceConfig)
            });

            if (!response.ok) {
                throw new Error(`TTS request failed: ${response.statusText}`);
            }

            const data = await response.json();
            const audioUrl = `data:audio/mp3;base64,${data.audio}`;
            
            // Add to queue
            this.messageQueue.set(messageId, {
                audioUrl,
                avatar,
                message
            });

            // Update debug info
            if (window.ELEMENTS?.$debugQueueCount) {
                window.ELEMENTS.$debugQueueCount.text(this.messageQueue.size);
                window.ELEMENTS.$debugQueueStatus.text('Queued');
            }

            // Process queue if not already processing
            if (!this.isProcessing) {
                this.processQueue();
            }
        } catch (error) {
            console.error('Error processing TTS message:', error);
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