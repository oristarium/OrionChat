class TTSManager {
    constructor() {
        this.messageQueue = [];
        this.isProcessing = false;
        this.audioCache = new Map();
        this.lastPlayedMessageId = null;
    }

    async processTTSMessage(chatMessage, avatar) {
        // Check for duplicate consecutive message
        if (chatMessage.message_id === this.lastPlayedMessageId) {
            console.log('Ignoring duplicate consecutive message:', chatMessage.message_id);
            return;
        }

        // Check if this message is already in queue
        const isDuplicate = this.messageQueue.some(item => 
            item.chatMessage.message_id === chatMessage.message_id
        );
        if (isDuplicate) {
            console.log('Ignoring duplicate message already in queue:', chatMessage.message_id);
            return;
        }

        // Get a random voice from the selected avatar
        const voice = avatar.getRandomVoice();
        if (!voice) {
            console.error('No voice found for avatar:', avatar.id);
            return;
        }

        // Add to queue
        this.messageQueue.push({
            chatMessage,
            avatar,
            voice
        });
        
        this.updateDebugBox();

        if (!this.isProcessing) {
            this.processQueue();
        }
    }

    async processQueue() {
        if (this.isProcessing || this.messageQueue.length === 0) {
            return;
        }

        this.isProcessing = true;

        while (this.messageQueue.length > 0) {
            const { chatMessage, avatar, voice } = this.messageQueue[0];
            console.log(`Processing message for avatar ${avatar.id} with voice ${voice.voice_id}`);

            const content = chatMessage.data.content;
            const textForTTS = typeof content === 'string' 
                ? content 
                : (content.sanitized || content.raw || content.formatted || '');

            if (!textForTTS.trim()) {
                this.messageQueue.shift();
                this.updateDebugBox();
                continue;
            }

            const authorName = chatMessage.data.author.display_name 
                || chatMessage.data.author.name 
                || 'Anonymous';

            // Update content
            avatar.element.$messageBubble.text(textForTTS);
            avatar.element.$username.text(authorName);

            try {
                let audio;
                if (this.audioCache.has(chatMessage.message_id)) {
                    console.log('Using cached audio for message:', chatMessage.message_id);
                    audio = this.audioCache.get(chatMessage.message_id);
                    this.audioCache.delete(chatMessage.message_id);
                } else {
                    console.log('Fetching TTS audio from API...');
                    audio = await this.fetchTTSAudio(textForTTS, voice.voice_id, voice.provider);
                }

                this.lastPlayedMessageId = chatMessage.message_id;
                
                // Wait for audio to finish playing
                await new Promise((resolve, reject) => {
                    avatar.audioPlayer.src = `data:audio/mp3;base64,${audio}`;
                    
                    const onEnded = () => {
                        console.log(`Audio finished for avatar ${avatar.id}`);
                        avatar.updateUIState(false);
                        avatar.audioPlayer.onended = null;
                        resolve();
                    };

                    const onError = (error) => {
                        console.error(`Audio error for avatar ${avatar.id}:`, error);
                        avatar.updateUIState(false);
                        avatar.audioPlayer.onended = null;
                        reject(error);
                    };

                    avatar.audioPlayer.onended = onEnded;
                    avatar.audioPlayer.onerror = onError;
                    
                    avatar.audioPlayer.play().catch(onError);
                });

                // Add small delay between messages
                await new Promise(resolve => setTimeout(resolve, 300));

            } catch (error) {
                console.error('Error processing queued TTS:', error);
                avatar.updateUIState(false);
            }

            this.messageQueue.shift();
            this.updateDebugBox();
        }

        this.isProcessing = false;
    }

    async fetchTTSAudio(text, voiceId, voiceProvider) {
        try {
            const response = await fetch('/tts-service', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    text, 
                    voice_id: voiceId,
                    voice_provider: voiceProvider
                })
            });

            if (!response.ok) {
                throw new Error(`TTS API error: ${response.status}`);
            }

            const { audio } = await response.json();
            return audio;
        } catch (error) {
            console.error('TTS API error:', error);
            throw error;
        }
    }

    clearQueue() {
        // Keep the currently playing message if any
        if (this.isProcessing && this.messageQueue.length > 0) {
            const currentMessage = this.messageQueue[0];
            this.messageQueue = [currentMessage];
        } else {
            this.messageQueue = [];
        }
        this.audioCache.clear();
        this.updateDebugBox();
    }

    updateDebugBox() {
        if (!window.debugMode) return;
        
        const $debugQueueCount = $('#queue-count');
        const $debugQueueStatus = $('#queue-status');
        
        $debugQueueCount.text(this.messageQueue.length);
        
        const queueHTML = this.messageQueue.map((item, index) => {
            const text = typeof item.chatMessage.data.content === 'string' 
                ? item.chatMessage.data.content 
                : (item.chatMessage.data.content.sanitized || 
                   item.chatMessage.data.content.raw || 
                   item.chatMessage.data.content.formatted || '');
            
            let status = 'queued';
            if (index === 0 && this.isProcessing) {
                status = this.audioCache.has(item.chatMessage.message_id) ? 'done' : 'fetching';
            }
            
            return `
                <div class="debug-box__row">
                    <div class="debug-box__message">${text.substring(0, 30)}${text.length > 30 ? '...' : ''}</div>
                    <div class="debug-box__status debug-box__status--${status}">${status}</div>
                </div>
            `;
        }).join('');

        $debugQueueStatus.html(queueHTML);
    }
}

// Export the TTSManager
window.TTSManager = TTSManager; 