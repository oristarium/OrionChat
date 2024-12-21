export class ChatManager {
    constructor() {
        this.messages = [];
        this.savedMessages = [];
        this.onMessagesChange = null;
        this.onSavedMessagesChange = null;
        this.showToast = null;
    }

    addMessageUnique(message) {
        // Simply add the message and notify
        this.messages = [...this.messages, message];
        if (this.onMessagesChange) {
            this.onMessagesChange(this.messages);
        }
    }

    // API interaction methods
    async sendTTS(message) {
        const ttsData = {
            ...message,
            type: 'tts'
        };

        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(ttsData)
            });
            this.showToast?.('TTS sent');
        } catch (error) {
            console.error('Error sending TTS:', error);
            this.showToast?.('Failed to send TTS', 'error');
        }
    }

    async sendDisplay(message) {
        const displayData = {
            ...message,
            type: 'display'
        };

        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(displayData)
            });
            this.showToast?.('Message displayed');
        } catch (error) {
            console.error('Error displaying message:', error);
            this.showToast?.('Failed to display message', 'error');
        }
    }

    async sendCustomTTS(text) {
        if (!text) return false;

        const ttsData = {
            type: 'tts',
            data: {
                content: {
                    raw: text,
                    formatted: text,
                    sanitized: text
                }
            }
        };

        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(ttsData)
            });
            this.showToast?.('Custom TTS sent');
            return true;
        } catch (error) {
            console.error('Error sending custom TTS:', error);
            this.showToast?.('Failed to send custom TTS', 'error');
            return false;
        }
    }

    async sendCustomDisplay(text) {
        if (!text) return false;

        const displayData = {
            type: 'display',
            data: {
                content: {
                    raw: text,
                    formatted: text,
                    sanitized: text
                }
            }
        };

        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(displayData)
            });
            this.showToast?.('Custom message displayed');
            return true;
        } catch (error) {
            console.error('Error displaying custom message:', error);
            this.showToast?.('Failed to display custom message', 'error');
            return false;
        }
    }

    clearDisplay() {
        fetch('/update', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ type: 'clear_display' })
        }).catch(error => {
            console.error('Error clearing display:', error);
            this.showToast?.('Failed to clear display', 'error');
        });
    }

    clearTTS() {
        fetch('/update', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ type: 'clear_tts' })
        }).catch(error => {
            console.error('Error clearing TTS:', error);
            this.showToast?.('Failed to clear TTS', 'error');
        });
    }
} 