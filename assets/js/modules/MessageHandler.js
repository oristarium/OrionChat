export class MessageHandler {
    constructor() {
        this.languageSelect = document.getElementById('tts-language');
    }

    async sendToDisplay(message) {
        await this.sendMessage(message, 'display');
    }

    async sendToTTS(message) {
        await this.sendMessage(message, 'tts');
    }

    async sendMessage(message, type) {
        try {
            const update = {
                type: type,
                data: {},
                message: message,
                lang: this.languageSelect.value
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
            data: {},
            message: {
                type: 'chat',
                platform: 'system',
                timestamp: new Date().toISOString(),
                message_id: 'clear_' + Date.now(),
                room_id: '',
                data: {
                    author: {
                        id: 'system',
                        username: 'system',
                        display_name: 'System',
                        avatar_url: '',
                        roles: {
                            broadcaster: false,
                            moderator: false,
                            subscriber: false,
                            verified: false
                        },
                        badges: []
                    },
                    content: {
                        raw: '',
                        formatted: '',
                        sanitized: '',
                        rawHtml: '',
                        elements: []
                    },
                    metadata: {
                        type: 'chat'
                    }
                }
            },
            lang: this.languageSelect.value
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
} 