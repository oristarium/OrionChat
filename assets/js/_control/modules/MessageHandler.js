export class MessageHandler {
    constructor() {
        this.languageSelect = document.getElementById('tts-language');
        this.providerSelect = document.getElementById('tts-provider');
    }

    generateMessageId() {
        const id = 'manual_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        console.log('Generating manual message ID:', {
            id,
            location: new Error().stack // This will show us where the ID was generated
        });
        return id;
    }

    async sendToDisplay(message) {
        const displayData = typeof message === 'string' 
            ? {
                type: "display",
                data: {
                    message_id: this.generateMessageId(),
                    type: "chat",
                    platform: "manual",
                    timestamp: new Date().toISOString(),
                    content: {
                        formatted: message,
                        raw: message,
                        sanitized: message
                    }
                }
            }
            : {
                type: "display",
                data: {
                    message_id: message.message_id,
                    type: message.type,
                    platform: message.platform,
                    timestamp: message.timestamp,
                    ...message.data
                }
            };

        await this.sendMessage(displayData);
    }

    async sendToTTS(message) {
        const ttsData = typeof message === 'string'
            ? {
                type: "tts",
                data: {
                    message_id: this.generateMessageId(),
                    type: "chat",
                    platform: "manual",
                    timestamp: new Date().toISOString(),
                    content: {
                        formatted: message,
                        raw: message,
                        sanitized: message,
                        rawHtml: `<span class="text">${message}</span>`,
                        elements: [{
                            type: "text",
                            value: message,
                            position: [0, message.length]
                        }]
                    }
                }
            }
            : {
                type: "tts",
                data: {
                    message_id: message.message_id,
                    type: message.type,
                    platform: message.platform,
                    timestamp: message.timestamp,
                    ...message.data
                }
            };

        await this.sendMessage(ttsData);
    }

    async sendMessage(update) {
        try {
            console.log('Sending message:', update);

            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(update)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const responseData = await response.text();
            console.log('Server response:', responseData);

        } catch (error) {
            console.error('Error sending message:', error);
            throw error;
        }
    }

    async clearDisplay() {
        console.log('Clearing display...');
        const update = {
            type: 'clear_display'
        };

        await this.sendMessage(update);
    }

    async clearTTSQueue() {
        try {
            const update = {
                type: 'clear_tts'
            };

            await this.sendMessage(update);
        } catch (error) {
            console.error('Error clearing TTS queue:', error);
            throw error;
        }
    }
} 