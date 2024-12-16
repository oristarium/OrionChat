export class MessageHandler {
    constructor() {
        this.languageSelect = document.getElementById('tts-language');
        this.providerSelect = document.getElementById('tts-provider');
    }

    async sendToDisplay(message) {
        await this.sendMessage(message, 'display');
    }

    async sendToTTS(message) {
        await this.sendMessage(message, 'tts');
    }

    async sendMessage(message, type) {
        try {
            const voiceId = window.voiceManager.getCurrentVoiceId();
            const voiceProvider = window.voiceManager.providerSelect.value || 'google';

            const update = {
                type: type,
                data: {
                    message: message,
                    voice_id: voiceId,
                    voice_provider: voiceProvider
                }
            };
            
            console.log(`Sending ${type} message:`, update);

            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(update)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
        } catch (error) {
            console.error(`Error sending message to ${type}:`, error);
            throw error;
        }
    }

    async clearDisplay() {
        console.log('Clearing display...');
        const update = {
            type: 'clear_display',
            data: {}
        };

        console.log('Sending clear display command:', update);

        const response = await fetch('/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(update)
        });

        const responseText = await response.text();
        console.log('Server response:', response.status, responseText);

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}, body: ${responseText}`);
        }
    }

    async clearTTSQueue() {
        try {
            const update = {
                type: 'clear_tts',
                data: {}
            };

            console.log('Sending clear TTS queue command:', update);

            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(update)
            });

            const responseText = await response.text();
            console.log('Server response:', response.status, responseText);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}, body: ${responseText}`);
            }
        } catch (error) {
            console.error('Error clearing TTS queue:', error);
            throw error;
        }
    }
} 