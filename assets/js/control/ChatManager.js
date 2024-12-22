import { ChatterManager } from './ChatterManager.js';

/**
 * @typedef {Object} ChatBadge
 * @property {('subscriber'|'moderator'|'verified'|'custom')} type - The type of badge
 * @property {string} label - Display label for the badge
 * @property {string} image_url - URL to the badge image
 */

/**
 * @typedef {Object} ChatAuthor
 * @property {string} id - Format: {platform}-{provider_author_id}
 * @property {string} platform - Same as root platform field
 * @property {string} liveId - Same as root liveId field
 * @property {string} username - Author's username
 * @property {string} display_name - Author's display name
 * @property {string} avatar_url - URL to author's avatar
 * @property {Object} roles - Author's roles
 * @property {boolean} roles.broadcaster - Whether the author is the broadcaster
 * @property {boolean} roles.moderator - Whether the author is a moderator
 * @property {boolean} roles.subscriber - Whether the author is a subscriber
 * @property {boolean} roles.verified - Whether the author is verified
 * @property {ChatBadge[]} badges - Array of author's badges
 */

/**
 * @typedef {Object} ChatElement
 * @property {('text'|'emote')} type - Type of content element
 * @property {string} value - The content value
 * @property {[number, number]} position - Start and end position in the message
 * @property {Object} [metadata] - Additional metadata for emotes
 * @property {string} [metadata.url] - URL to the emote image
 * @property {string} [metadata.alt] - Alt text for the emote
 * @property {boolean} [metadata.is_custom] - Whether this is a custom emote
 */

/**
 * @typedef {Object} ChatContent
 * @property {string} raw - Original message with emote codes
 * @property {string} formatted - Message with emote codes
 * @property {string} sanitized - Plain text only
 * @property {string} rawHtml - Pre-rendered HTML with emotes
 * @property {ChatElement[]} elements - Message broken into parts
 */

/**
 * @typedef {Object} MonetaryData
 * @property {string} amount - The donation amount
 * @property {string} formatted - Formatted amount string
 * @property {string} color - Color associated with the donation
 */

/**
 * @typedef {Object} StickerData
 * @property {string} url - URL to the sticker image
 * @property {string} alt - Alt text for the sticker
 */

/**
 * @typedef {Object} ChatMetadata
 * @property {('chat'|'super_chat')} type - Type of chat message
 * @property {MonetaryData} [monetary_data] - Data for monetary contributions
 * @property {StickerData} [sticker] - Data for sticker super chats
 */

/**
 * @typedef {Object} ChatMessage
 * @property {string} type - Type of message (always 'chat')
 * @property {('youtube'|'twitch'|'tiktok')} platform - Source platform
 * @property {string} liveId - Standardized liveId of the stream
 * @property {string} timestamp - Message timestamp
 * @property {string} message_id - Format: {liveId}-{provider_message_id} to ensure uniqueness
 * @property {Object} data - Message data
 * @property {ChatAuthor} data.author - Message author information
 * @property {ChatContent} data.content - Message content
 * @property {ChatMetadata} [data.metadata] - Additional message metadata
 */

// Database setup for saved messages
const savedChatDbName = 'OrionSavedChatDB';
const dbVersion = 2;
let savedChatDb;

/**
 * Manages chat messages, including saving, loading, and interacting with messages
 */
export class ChatManager {
    constructor() {
        /** @type {ChatMessage[]} */
        this.messages = [];
        /** @type {ChatMessage[]} */
        this.savedMessages = [];
        /** @type {boolean} */
        this.isDBReady = false;
        /** @type {function(ChatMessage[]): void} */
        this.onMessagesChange = null;
        /** @type {function(ChatMessage[]): void} */
        this.onSavedMessagesChange = null;
        /** @type {function(string, string=): void} */
        this.showToast = null;
        /** @type {boolean} */
        this.isTTSAll = false;

        // Initialize ChatterManager
        this.chatterManager = new ChatterManager();
    }

    /**
     * Initializes the ChatManager and its database
     */
    async init() {
        try {
            await this.initDB();
            this.isDBReady = true;
            await this.loadSavedMessages();
            
            // Initialize ChatterManager
            this.chatterManager.init();
            
            // Pass the showToast function to ChatterManager
            this.chatterManager.showToast = this.showToast;
        } catch (error) {
            console.error('Failed to initialize ChatManager:', error);
        }
    }

    /**
     * Initializes the IndexedDB database for saved messages
     * @returns {Promise<IDBDatabase>}
     */
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

    /**
     * Sets the callback for chatter changes
     * @param {function(ChatAuthor[]): void} callback
     */
    set onChattersChange(callback) {
        this.chatterManager.onChattersChange = callback;
    }

    /**
     * Sets the callback for saved chatters changes
     * @param {function(ChatAuthor[]): void} callback
     */
    set onSavedChattersChange(callback) {
        this.chatterManager.onSavedChattersChange = callback;
    }

    /**
     * Adds a new chat message to the messages array and updates unique chatters
     * @param {ChatMessage} message - The chat message to add
     */
    addMessageUnique(message) {
        // Update messages
        this.messages = [...this.messages, message];
        if (this.onMessagesChange) {
            this.onMessagesChange(this.messages);
        }

        // Update unique chatters through ChatterManager
        this.chatterManager.addChatter(message.data.author);
    }

    /**
     * Removes a live ID and prunes associated chatters
     * @param {string} liveId - The live ID to remove
     */
    removeLiveId(liveId) {
        this.chatterManager.removeLiveId(liveId);
    }

    /**
     * Gets all unique chatters
     * @returns {ChatAuthor[]}
     */
    getChatters() {
        return this.chatterManager.getChatters();
    }

    /**
     * Filters chatters based on criteria
     * @param {Object} filters
     * @param {string} [filters.search] - Search term for username/display name
     * @param {string} [filters.platform] - Platform filter
     * @param {Object} [filters.roles] - Role filters
     * @returns {ChatAuthor[]}
     */
    filterChatters(filters) {
        return this.chatterManager.filterChatters(filters);
    }

    /**
     * Loads saved messages from IndexedDB
     * @returns {Promise<void>}
     */
    async loadSavedMessages() {
        if (!this.isDBReady) return;

        try {
            const transaction = savedChatDb.transaction(['savedMessages'], 'readonly');
            const store = transaction.objectStore('savedMessages');
            const request = store.getAll();

            return new Promise((resolve, reject) => {
                request.onsuccess = () => {
                    this.savedMessages = request.result;
                    this.onSavedMessagesChange?.(this.savedMessages);
                    resolve();
                };
                request.onerror = () => reject(request.error);
            });
        } catch (error) {
            console.error('Failed to load saved messages:', error);
        }
    }

    /**
     * Saves a chat message to favorites in IndexedDB
     * @param {ChatMessage} message - The chat message to save
     */
    async saveMessageToFavorites(message) {
        if (!this.isDBReady) return;

        try {
            // Clean the message object for storage
            const cleanMessage = {
                message_id: message.message_id,
                platform: message.platform,
                liveId: message.liveId,
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
                        elements: message.data.content.elements.map(element => ({
                            type: element.type,
                            value: element.value,
                            position: [...element.position],
                            metadata: element.metadata ? {
                                url: element.metadata.url,
                                alt: element.metadata.alt,
                                is_custom: element.metadata.is_custom
                            } : undefined
                        }))
                    },
                    metadata: message.data.metadata ? {
                        type: message.data.metadata.type,
                        monetary_data: message.data.metadata.monetary_data ? {
                            amount: message.data.metadata.monetary_data.amount,
                            formatted: message.data.metadata.monetary_data.formatted,
                            color: message.data.metadata.monetary_data.color
                        } : undefined,
                        sticker: message.data.metadata.sticker ? {
                            url: message.data.metadata.sticker.url,
                            alt: message.data.metadata.sticker.alt
                        } : undefined
                    } : undefined
                }
            };

            const transaction = savedChatDb.transaction(['savedMessages'], 'readwrite');
            const store = transaction.objectStore('savedMessages');
            
            await new Promise((resolve, reject) => {
                const request = store.put(cleanMessage);
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });

            // Create a new array to trigger reactivity
            this.savedMessages = [...this.savedMessages, cleanMessage];
            this.onSavedMessagesChange?.(this.savedMessages);
            this.showToast?.('Message saved');
        } catch (error) {
            console.error('Failed to save message:', error);
            this.showToast?.('Failed to save message', 'error');
        }
    }

    /**
     * Removes a saved message from IndexedDB
     * @param {string} messageId - ID of the message to remove
     */
    async removeSavedMessage(messageId) {
        if (!this.isDBReady) return;

        try {
            const transaction = savedChatDb.transaction(['savedMessages'], 'readwrite');
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

    /**
     * Sends a chat message to TTS
     * @param {ChatMessage} message - The message to send to TTS
     */
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
            if (!this.isTTSAll) {
                this.showToast?.('TTS sent');
            }
        } catch (error) {
            console.error('Error sending TTS:', error);
            this.showToast?.('Failed to send TTS', 'error');
        }
    }

    /**
     * Sends a chat message to display
     * @param {ChatMessage} message - The message to display
     */
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

    /**
     * Sends custom text to TTS
     * @param {string} text - The text to send to TTS
     * @returns {Promise<boolean>} Whether the operation was successful
     */
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

    /**
     * Sends custom text to display
     * @param {string} text - The text to display
     * @returns {Promise<boolean>} Whether the operation was successful
     */
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

    /**
     * Clears the current display
     */
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

    /**
     * Clears the current TTS queue
     */
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

    /**
     * Saves a chatter to favorites
     * @param {ChatAuthor} chatter - The chatter to save
     */
    async saveChatter(chatter) {
        return this.chatterManager.saveChatter(chatter);
    }

    /**
     * Removes a saved chatter
     * @param {string} chatterId - ID of the chatter to remove
     */
    async removeSavedChatter(chatterId) {
        return this.chatterManager.removeSavedChatter(chatterId);
    }
} 