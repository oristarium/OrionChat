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
            await this.loadSavedMessages();
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
        // Simply add new messages to the end of the array
        this.messages.push(message);
        
        // Ensure Vue updates the view by creating a new array reference
        this.onMessagesChange?.([...this.messages]);
    }

    async clearAllMessages() {
        this.messages = [];
        this.onMessagesChange?.(this.messages);
        this.showToast?.('Chat messages cleared');
    }

    // Saved messages functionality
    async saveMessageToFavorites(message) {
        if (!savedChatDb) return;
        
        const transaction = savedChatDb.transaction(['savedMessages'], 'readwrite');
        const store = transaction.objectStore('savedMessages');
        
        try {
            await new Promise((resolve, reject) => {
                const request = store.add(message);
                request.onsuccess = () => {
                    console.log('Message saved to favorites:', message.message_id);
                    resolve();
                };
                request.onerror = () => reject(request.error);
            });
            
            this.showToast?.('Message saved to favorites');
            await this.loadSavedMessages();
        } catch (error) {
            if (error.name === 'ConstraintError') {
                this.showToast?.('Message already saved', 'info');
                return;
            }
            this.showToast?.('Error saving message', 'error');
            console.error('Error saving to favorites:', error);
        }
    }

    async loadSavedMessages() {
        if (!savedChatDb) return [];
        
        return new Promise((resolve, reject) => {
            const transaction = savedChatDb.transaction(['savedMessages'], 'readonly');
            const store = transaction.objectStore('savedMessages');
            
            const request = store.getAll();
            
            request.onsuccess = () => {
                this.savedMessages = request.result;
                this.onSavedMessagesChange?.(this.savedMessages);
                resolve(request.result);
            };
            
            request.onerror = () => {
                reject(request.error);
            };
        });
    }

    async removeSavedMessage(messageId) {
        if (!savedChatDb) return;
        
        const transaction = savedChatDb.transaction(['savedMessages'], 'readwrite');
        const store = transaction.objectStore('savedMessages');
        await new Promise((resolve, reject) => {
            const deleteRequest = store.delete(messageId);
            deleteRequest.onsuccess = () => resolve();
            deleteRequest.onerror = () => reject(deleteRequest.error);
        });
        
        this.savedMessages = this.savedMessages.filter(msg => msg.message_id !== messageId);
        this.onSavedMessagesChange?.(this.savedMessages);
        this.showToast?.('Saved message removed');
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