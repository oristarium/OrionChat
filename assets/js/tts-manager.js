class TTSManager {
    constructor(debugManager) {
        this.messageQueue = new Map();
        this.isProcessing = false;
        this.debugManager = debugManager;
        this.currentlyPlaying = null;
        this.audioCache = new Map();
        this.lastProcessedMessageId = null;
    }

    async processTTSMessage(message, avatar) {
        // Validate required fields according to docs
        if (!message.data?.content?.sanitized) {
            this.debugManager.error('Invalid TTS message structure - missing required fields');
            return;
        }

        // Get message ID or generate one
        const messageId = message.message_id || Date.now().toString();

        // Check if this is a duplicate consecutive message
        if (messageId === this.lastProcessedMessageId) {
            this.debugManager.log('Blocking consecutive duplicate message:', messageId);
            return;
        }

        // Check if this message is already in queue
        if (this.messageQueue.has(messageId)) {
            this.debugManager.log('Message already in queue, skipping:', messageId);
            return;
        }

        // Skip empty content
        const sanitizedText = message.data.content.sanitized.trim();
        if (!sanitizedText) {
            this.debugManager.log('Empty TTS content, skipping');
            return;
        }

        try {
            // First add to queue with initial 'queued' status
            const queueItem = {
                avatar,
                message,
                status: 'queued'
            };
            
            this.messageQueue.set(messageId, queueItem);
            
            // Then start fetching audio and store the promise
            queueItem.audioPromise = this.fetchAudio(messageId, queueItem);

            // Update debug display
            this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);
            this.debugManager.log('Added message to queue:', messageId);

            // Process queue if not already processing
            if (!this.isProcessing) {
                this.processQueue();
            }
        } catch (error) {
            this.debugManager.error('Error processing TTS message:', error);
        }
    }

    async fetchAudio(messageId, queueItem) {
        try {
            // Update status to fetching before API call
            queueItem.status = 'fetching';
            this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);

            const sanitizedText = queueItem.message.data.content.sanitized.trim();
            const voiceConfig = {
                text: sanitizedText,
                voice_id: queueItem.message.voice_id || queueItem.avatar.getRandomVoice()?.voice_id,
                voice_provider: queueItem.message.voice_provider || queueItem.avatar.getRandomVoice()?.provider
            };

            if (!voiceConfig.voice_id || !voiceConfig.voice_provider) {
                throw new Error('No valid voice configuration available');
            }

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

            // Update status to fetched after successful fetch
            queueItem.status = 'fetched';
            this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);
            
            return `data:audio/mp3;base64,${data.audio}`;
        } catch (error) {
            this.debugManager.error('Error fetching audio:', error);
            throw error;
        }
    }

    async processQueue() {
        if (this.messageQueue.size === 0 || this.isProcessing) {
            return;
        }

        this.isProcessing = true;
        this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);
        this.debugManager.log('Started processing queue');

        while (this.messageQueue.size > 0) {
            const [messageId, queueItem] = this.messageQueue.entries().next().value;
            
            try {
                // Update last processed message ID
                this.lastProcessedMessageId = messageId;
                
                const { avatar, message, audioPromise } = queueItem;
                // Wait for audio to be ready (status should already be 'fetched' or 'fetching')
                const audioUrl = await audioPromise;
                
                // Update to playing status
                queueItem.status = 'playing';
                this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);

                // Update message content before playing
                avatar.updateMessageContent(message);

                // Play audio and wait for completion
                avatar.audioPlayer.src = audioUrl;
                
                await new Promise((resolve, reject) => {
                    avatar.audioPlayer.onended = resolve;
                    avatar.audioPlayer.onerror = reject;
                    avatar.audioPlayer.play().catch(reject);
                });

                // Add small delay between messages
                await new Promise(resolve => setTimeout(resolve, 300));

            } catch (error) {
                this.debugManager.error('Error playing audio:', error);
            } finally {
                // Remove from queue after playing
                this.messageQueue.delete(messageId);
                this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);
            }
        }

        this.isProcessing = false;
        this.debugManager.log('Queue processing completed');
    }

    clearQueue() {
        this.lastProcessedMessageId = null;
        // Keep currently playing message if any
        if (this.isProcessing && this.messageQueue.size > 0) {
            const [[firstMessageId, firstMessage]] = this.messageQueue.entries();
            this.messageQueue.clear();
            this.messageQueue.set(firstMessageId, firstMessage);
        } else {
            this.messageQueue.clear();
        }
        this.debugManager.updateQueueStatus(this.messageQueue, this.isProcessing);
        this.debugManager.log('Queue cleared');
    }
}

// Export the TTSManager
window.TTSManager = TTSManager; 