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
            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    message: message,
                    lang: this.languageSelect.value
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
        } catch (error) {
            console.error(`Error sending message to ${type}:`, error);
        }
    }
} 