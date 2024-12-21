// Database setup for saved messages only
const savedChatDbName = 'OrionSavedChatDB';
const dbVersion = 2;
let savedChatDb;

export class ChatManager {
    constructor() {
        this.messages = [];
        this.savedMessages = [];
        this.isDBReady = false;
        this.onMessagesChange = null;
        this.onSavedMessagesChange = null;
        this.showToast = null;
    }

    async init() {
        try {
            await this.initDB();
            this.isDBReady = true;
        } catch (error) {
            console.error('Failed to initialize ChatManager:', error);
        }
    }

    async initDB() {
        return new Promise((resolve, reject) => {
            console.log('Initializing Saved Chat IndexedDB...');
            const request = indexedDB.open(savedChatDbName, dbVersion);

            request.onerror = (event) => {
                console.error('Saved Chat IndexedDB error:', event.target.error);
                reject(event.target.error);
            };

            request.onsuccess = (event) => {
                console.log('Saved Chat IndexedDB initialized successfully');
                savedChatDb = event.target.result;
                resolve(savedChatDb);
            };

            request.onupgradeneeded = (event) => {
                console.log('Creating/upgrading Saved Chat IndexedDB structure...');
                const db = event.target.result;
                
                if (!db.objectStoreNames.contains('savedMessages')) {
                    const store = db.createObjectStore('savedMessages', { keyPath: 'message_id' });
                    store.createIndex('timestamp', 'timestamp', { unique: false });
                    console.log('Saved messages store created');
                }
            };
        });
    }

    addMessageUnique(message) {
        this.messages.push(message);
        this.onMessagesChange?.(this.messages);
    }

    async saveMessageToFavorites(message) {
        if (!this.isDBReady) return;

        try {
            const transaction = db.transaction(['savedMessages'], 'readwrite');
            const store = transaction.objectStore('savedMessages');
            
            await new Promise((resolve, reject) => {
                const request = store.put(message);
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });

            this.savedMessages.push(message);
            this.onSavedMessagesChange?.(this.savedMessages);
            this.showToast?.('Message saved');
        } catch (error) {
            console.error('Failed to save message:', error);
            this.showToast?.('Failed to save message', 'error');
        }
    }

    async removeSavedMessage(messageId) {
        if (!this.isDBReady) return;

        try {
            const transaction = db.transaction(['savedMessages'], 'readwrite');
            const store = transaction.objectStore('savedMessages');
            
            await new Promise((resolve, reject) => {
                const request = store.delete(messageId);
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });

            this.savedMessages = this.savedMessages.filter(m => m.message_id !== messageId);
            this.onSavedMessagesChange?.(this.savedMessages);
            this.showToast?.('Message removed');
        } catch (error) {
            console.error('Failed to remove message:', error);
            this.showToast?.('Failed to remove message', 'error');
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