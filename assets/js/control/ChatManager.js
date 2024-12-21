// Database setup
const chatDbName = 'OrionChatDB';
const savedChatDbName = 'OrionSavedChatDB';
const dbVersion = 2;
let chatDb;
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
        return Promise.all([
            // Initialize main chat database
            new Promise((resolve, reject) => {
                console.log('Initializing Chat IndexedDB...');
                const request = indexedDB.open(chatDbName, dbVersion);

                request.onerror = (event) => {
                    console.error('Chat IndexedDB error:', event.target.error);
                    reject(event.target.error);
                };

                request.onsuccess = (event) => {
                    console.log('Chat IndexedDB initialized successfully');
                    chatDb = event.target.result;
                    resolve(chatDb);
                };

                request.onupgradeneeded = (event) => {
                    console.log('Creating/upgrading Chat IndexedDB structure...');
                    const db = event.target.result;
                    
                    if (!db.objectStoreNames.contains('messages')) {
                        const store = db.createObjectStore('messages', { keyPath: 'message_id' });
                        store.createIndex('connectionId', 'connectionId', { unique: false });
                        store.createIndex('timestamp', 'timestamp', { unique: false });
                        console.log('Message store created');
                    }
                };
            }),
            // Initialize saved messages database
            new Promise((resolve, reject) => {
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
            })
        ]);
    }

    addMessageUnique(message) {
        const exists = this.messages.some(m => m.message_id === message.message_id);
        if (!exists) {
            this.messages.push(message);
            if (this.messages.length > 200) {
                this.messages = this.messages.slice(-200);
            }
            this.onMessagesChange?.(this.messages);
        }
    }

    async saveMessage(message, connectionId) {
        if (!chatDb) return;
        
        const transaction = chatDb.transaction(['messages'], 'readwrite');
        const store = transaction.objectStore('messages');
        
        const messageToStore = {
            ...message,
            connectionId
        };
        
        try {
            await store.add(messageToStore);
            console.log('Message saved to IndexedDB');
        } catch (error) {
            if (error.name === 'ConstraintError') {
                console.log('Message already exists in IndexedDB');
                return;
            }
            console.error('Error saving message:', error);
        }
    }

    async loadMessages(connectionId) {
        if (!chatDb) return [];
        
        return new Promise((resolve, reject) => {
            console.log('Loading messages for connection:', connectionId);
            const transaction = chatDb.transaction(['messages'], 'readonly');
            const store = transaction.objectStore('messages');
            const index = store.index('connectionId');
            
            const request = index.getAll(connectionId);
            
            request.onsuccess = () => {
                console.log(`Loaded ${request.result.length} messages from IndexedDB`);
                resolve(request.result);
            };
            
            request.onerror = () => {
                console.error('Error loading messages:', request.error);
                reject(request.error);
            };
        });
    }

    async clearMessages(connectionId) {
        if (!chatDb) return;
        
        const transaction = chatDb.transaction(['messages'], 'readwrite');
        const store = transaction.objectStore('messages');
        const index = store.index('connectionId');
        
        const request = index.getAllKeys(connectionId);
        
        request.onsuccess = () => {
            const keys = request.result;
            keys.forEach(key => {
                store.delete(key);
            });
        };
    }

    async clearAllMessages() {
        this.messages = [];
        this.onMessagesChange?.(this.messages);
        this.showToast?.('Chat messages cleared');
    }

    async saveMessageToFavorites(message) {
        if (!savedChatDb) return;
        
        const transaction = savedChatDb.transaction(['savedMessages'], 'readwrite');
        const store = transaction.objectStore('savedMessages');
        
        const cleanMessage = {
            message_id: message.message_id,
            platform: message.platform,
            timestamp: message.timestamp,
            data: {
                author: {
                    id: message.data.author.id,
                    username: message.data.author.username,
                    display_name: message.data.author.display_name,
                    avatar_url: message.data.author.avatar_url,
                    roles: { ...message.data.author.roles },
                    badges: message.data.author.badges.map(badge => ({
                        type: badge.type,
                        label: badge.label,
                        image_url: badge.image_url
                    }))
                },
                content: {
                    raw: message.data.content.raw,
                    formatted: message.data.content.formatted,
                    sanitized: message.data.content.sanitized,
                    rawHtml: message.data.content.rawHtml,
                    elements: message.data.content.elements?.map(el => ({
                        type: el.type,
                        value: el.value,
                        position: [...el.position],
                        metadata: el.metadata ? { ...el.metadata } : undefined
                    }))
                },
                metadata: message.data.metadata ? {
                    type: message.data.metadata.type,
                    monetary_data: message.data.metadata.monetary_data ? { ...message.data.metadata.monetary_data } : undefined,
                    sticker: message.data.metadata.sticker ? { ...message.data.metadata.sticker } : undefined
                } : undefined
            }
        };
        
        try {
            await store.add(cleanMessage);
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
        await store.delete(messageId);
        
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
            console.error('Error sending display:', error);
            this.showToast?.('Failed to display message', 'error');
        }
    }

    async sendCustomTTS(message) {
        if (!message.trim()) {
            this.showToast?.('Please enter a message', 'error');
            return;
        }

        const ttsData = {
            type: 'tts',
            data: {
                content: {
                    sanitized: message.trim()
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

    async sendCustomDisplay(message) {
        if (!message.trim()) {
            this.showToast?.('Please enter a message', 'error');
            return;
        }

        const displayData = {
            type: 'display',
            data: {
                content: {
                    sanitized: message.trim()
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
            console.error('Error sending custom display:', error);
            this.showToast?.('Failed to display custom message', 'error');
            return false;
        }
    }

    async clearDisplay() {
        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ type: 'clear_display' })
            });
            this.showToast?.('Display cleared');
        } catch (error) {
            console.error('Error clearing display:', error);
            this.showToast?.('Failed to clear display', 'error');
        }
    }

    async clearTTS() {
        try {
            await fetch('/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ type: 'clear_tts' })
            });
            this.showToast?.('TTS cleared');
        } catch (error) {
            console.error('Error clearing TTS:', error);
            this.showToast?.('Failed to clear TTS', 'error');
        }
    }
} 