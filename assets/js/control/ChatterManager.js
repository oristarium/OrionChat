/**
 * Manages chat authors (chatters) and their states
 */
export class ChatterManager {
    constructor() {
        /** @type {Map<string, ChatAuthor>} */
        this.uniqueChatters = new Map();
        /** @type {Set<string>} */
        this.activeLiveIds = new Set();
        /** @type {function(ChatAuthor[]): void} */
        this.onChattersChange = null;
        /** @type {function(ChatAuthor[]): void} */
        this.onSavedChattersChange = null;
        /** @type {function(string, string=): void} */
        this.showToast = null;
        /** @type {boolean} */
        this.shouldShowChatterToasts = false;
        /** @type {boolean} */
        this.isDBReady = false;
        /** @type {ChatAuthor[]} */
        this.savedChatters = [];
    }

    /**
     * Initializes the ChatterManager
     */
    async init() {
        // Initialize IndexedDB
        try {
            await this.initDB();
            this.isDBReady = true;
            await this.loadSavedChatters();
        } catch (error) {
            console.error('Failed to initialize ChatterManager DB:', error);
        }

        // Enable chatter toasts after 3 seconds
        setTimeout(() => {
            this.shouldShowChatterToasts = true;
        }, 3000);
    }

    /**
     * Initializes the IndexedDB database for saved chatters
     * @returns {Promise<IDBDatabase>}
     */
    async initDB() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open('OrionSavedChattersDB', 1);

            request.onerror = (event) => {
                console.error('Saved Chatters IndexedDB error:', event.target.error);
                reject(event.target.error);
            };

            request.onsuccess = (event) => {
                console.log('Saved Chatters IndexedDB initialized successfully');
                this.db = event.target.result;
                resolve(this.db);
            };

            request.onupgradeneeded = (event) => {
                console.log('Creating/upgrading Saved Chatters IndexedDB structure...');
                const db = event.target.result;
                
                if (!db.objectStoreNames.contains('savedChatters')) {
                    const store = db.createObjectStore('savedChatters', { keyPath: 'id' });
                    store.createIndex('platform', 'platform', { unique: false });
                    store.createIndex('username', 'username', { unique: false });
                    console.log('Saved chatters store created');
                }
            };
        });
    }

    /**
     * Loads saved chatters from IndexedDB
     * @returns {Promise<void>}
     */
    async loadSavedChatters() {
        if (!this.isDBReady) return;

        try {
            const transaction = this.db.transaction(['savedChatters'], 'readonly');
            const store = transaction.objectStore('savedChatters');
            const request = store.getAll();

            return new Promise((resolve, reject) => {
                request.onsuccess = () => {
                    this.savedChatters = request.result;
                    this.onSavedChattersChange?.(this.savedChatters);
                    resolve();
                };
                request.onerror = () => reject(request.error);
            });
        } catch (error) {
            console.error('Failed to load saved chatters:', error);
        }
    }

    /**
     * Saves a chatter to favorites in IndexedDB
     * @param {ChatAuthor} chatter - The chatter to save
     */
    async saveChatter(chatter) {
        if (!this.isDBReady) return;

        try {
            const transaction = this.db.transaction(['savedChatters'], 'readwrite');
            const store = transaction.objectStore('savedChatters');
            
            await new Promise((resolve, reject) => {
                const request = store.put(chatter);
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });

            // Create a new array to trigger reactivity
            this.savedChatters = [...this.savedChatters, chatter];
            this.onSavedChattersChange?.(this.savedChatters);
            this.showToast?.('Chatter saved');
        } catch (error) {
            console.error('Failed to save chatter:', error);
            this.showToast?.('Failed to save chatter', 'error');
        }
    }

    /**
     * Removes a saved chatter from IndexedDB
     * @param {string} chatterId - ID of the chatter to remove
     */
    async removeSavedChatter(chatterId) {
        if (!this.isDBReady) return;

        try {
            const transaction = this.db.transaction(['savedChatters'], 'readwrite');
            const store = transaction.objectStore('savedChatters');
            
            await new Promise((resolve, reject) => {
                const request = store.delete(chatterId);
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });

            this.savedChatters = this.savedChatters.filter(c => c.id !== chatterId);
            this.onSavedChattersChange?.(this.savedChatters);
            this.showToast?.('Chatter removed');
        } catch (error) {
            console.error('Failed to remove chatter:', error);
            this.showToast?.('Failed to remove chatter', 'error');
        }
    }

    /**
     * Adds a new live ID to track
     * @param {string} liveId - The live ID to add
     */
    addLiveId(liveId) {
        if (!this.activeLiveIds.has(liveId)) {
            this.activeLiveIds.add(liveId);
        }
    }

    /**
     * Removes a live ID and prunes associated chatters
     * @param {string} liveId - The live ID to remove
     */
    removeLiveId(liveId) {
        if (!this.activeLiveIds.has(liveId)) return;

        this.activeLiveIds.delete(liveId);

        // Get current chatters count for logging
        const beforeCount = this.uniqueChatters.size;

        // Filter out chatters from the disconnected live ID
        for (const [chatterId, chatter] of this.uniqueChatters.entries()) {
            if (chatter.liveId === liveId) {
                this.uniqueChatters.delete(chatterId);
            }
        }

        // Notify of chatters change if any were removed
        if (beforeCount !== this.uniqueChatters.size && this.onChattersChange) {
            const chatters = Array.from(this.uniqueChatters.values());
            this.onChattersChange(chatters);
        }
    }

    /**
     * Adds or updates a chatter
     * @param {ChatAuthor} author - The chat author to add/update
     */
    addChatter(author) {
        if (!this.uniqueChatters.has(author.id)) {
            this.uniqueChatters.set(author.id, author);

            // Only show toast if enabled and after initialization delay
            if (this.shouldShowChatterToasts) {
                const platformIcon = {
                    'youtube': '🔴',
                    'twitch': '💜',
                    'tiktok': '🎵'
                }[author.platform] || '👤';

                let roleIcon = '';
                if (author.roles.broadcaster) roleIcon = '👑';
                else if (author.roles.moderator) roleIcon = '🛡️';
                else if (author.roles.subscriber) roleIcon = '⭐';
                else if (author.roles.verified) roleIcon = '✓';

                this.showToast?.(
                    `${platformIcon} New chatter: ${roleIcon}${author.display_name}`,
                    'info'
                );
            }

            if (this.onChattersChange) {
                const chatters = Array.from(this.uniqueChatters.values());
                this.onChattersChange(chatters);
            }
        }
    }

    /**
     * Gets all unique chatters
     * @returns {ChatAuthor[]}
     */
    getChatters() {
        const chatters = Array.from(this.uniqueChatters.values());
        return chatters;
    }

    /**
     * Filters chatters based on criteria
     * @param {Object} filters
     * @param {string} [filters.search] - Search term for username/display name
     * @param {string} [filters.platform] - Platform filter
     * @param {Object} [filters.roles] - Role filters
     * @returns {ChatAuthor[]}
     */
    filterChatters({ search = '', platform = '', roles = {} } = {}) {
        const allChatters = this.getChatters();
        const filtered = allChatters.filter(chatter => {
            // Platform filter
            if (platform && chatter.platform !== platform) return false;

            // Role filters
            if (roles.broadcaster && !chatter.roles.broadcaster) return false;
            if (roles.moderator && !chatter.roles.moderator) return false;
            if (roles.subscriber && !chatter.roles.subscriber) return false;
            if (roles.verified && !chatter.roles.verified) return false;

            // Search filter
            if (search) {
                const searchLower = search.toLowerCase();
                return (
                    chatter.username.toLowerCase().includes(searchLower) ||
                    chatter.display_name.toLowerCase().includes(searchLower)
                );
            }

            return true;
        });
        return filtered;
    }
} 