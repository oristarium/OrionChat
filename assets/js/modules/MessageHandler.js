export class MessageHandler {
    constructor() {
        this.languageSelect = document.getElementById('tts-language');
        this.providerSelect = document.getElementById('tts-provider');
    }

    async sendToDisplay(message) {
        const displayData = typeof message === 'string' 
            ? {
                type: "display",
                data: {
                    data: {
                        content: {
                            sanitized: message,
                            raw: message
                        }
                    }
                }
            }
            : message;

        await this.sendMessage(displayData);
    }

    async sendToTTS(message) {
        const ttsData = typeof message === 'string'
            ? {
                type: "tts",
                data: {
                    data: {
                        content: {
                            sanitized: message,
                            raw: message
                        }
                    }
                }
            }
            : message;

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
            type: 'clear_display',
            data: {}
        };

        await this.sendMessage(update);
    }

    async clearTTSQueue() {
        try {
            const update = {
                type: 'clear_tts',
                data: {}
            };

            await this.sendMessage(update);
        } catch (error) {
            console.error('Error clearing TTS queue:', error);
            throw error;
        }
    }
} 