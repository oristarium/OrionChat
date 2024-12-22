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
                    store.createIndex('status', 'status', { unique: false });
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
     * Saves a chatter to favorites in IndexedDB with a status
     * @param {ChatAuthor} chatter - The chatter to save
     * @param {('pinned'|'hidden')} status - The status to save the chatter with
     */
    async saveChatter(chatter, status) {
        if (!this.isDBReady) {
            console.log('DB not ready, cannot save chatter');
            return;
        }

        try {
            console.log(`Attempting to save chatter ${chatter.display_name} with status: ${status}`);
            const transaction = this.db.transaction(['savedChatters'], 'readwrite');
            const store = transaction.objectStore('savedChatters');
            
            const chatterWithStatus = {
                ...chatter,
                status: status
            };
            
            await new Promise((resolve, reject) => {
                const request = store.put(chatterWithStatus);
                request.onsuccess = () => {
                    console.log(`Successfully saved chatter to IndexedDB: ${chatter.display_name}`);
                    resolve();
                };
                request.onerror = () => reject(request.error);
            });

            // Create a new array to trigger reactivity
            const newSavedChatters = [...this.savedChatters.filter(c => c.id !== chatter.id), chatterWithStatus];
            console.log(`Previous saved chatters count: ${this.savedChatters.length}`);
            this.savedChatters = newSavedChatters;
            console.log(`New saved chatters count: ${this.savedChatters.length}`);
            
            // Notify of changes
            if (this.onSavedChattersChange) {
                console.log('Notifying of saved chatters change');
                this.onSavedChattersChange(this.savedChatters);
            }

            this.showToast?.(`Chatter ${status === 'pinned' ? 'pinned' : 'hidden'}`);
            console.log(`Successfully processed chatter ${chatter.display_name} with status ${status}`);
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
        if (!this.isDBReady) {
            console.log('DB not ready, cannot remove chatter');
            return;
        }

        try {
            console.log(`Attempting to remove chatter with ID: ${chatterId}`);
            const transaction = this.db.transaction(['savedChatters'], 'readwrite');
            const store = transaction.objectStore('savedChatters');
            
            await new Promise((resolve, reject) => {
                const request = store.delete(chatterId);
                request.onsuccess = () => {
                    console.log('Successfully removed chatter from IndexedDB');
                    resolve();
                };
                request.onerror = () => reject(request.error);
            });

            console.log(`Previous saved chatters count: ${this.savedChatters.length}`);
            this.savedChatters = this.savedChatters.filter(c => c.id !== chatterId);
            console.log(`New saved chatters count: ${this.savedChatters.length}`);

            if (this.onSavedChattersChange) {
                console.log('Notifying of saved chatters change');
                this.onSavedChattersChange(this.savedChatters);
            }

            this.showToast?.('Chatter removed');
            console.log('Successfully completed chatter removal');
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
        if (beforeCount !== this.uniqueChatters.size) {
            const chatters = Array.from(this.uniqueChatters.values());
            console.log('Total unique chatters after removal:', chatters.length);
            this.onChattersChange?.(chatters);
        }
    }

    /**
     * Adds or updates a chatter
     * @param {ChatAuthor} author - The chat author to add/update
     */
    addChatter(author) {
        if (!this.uniqueChatters.has(author.id)) {
            console.log('New unique chatter:', author.display_name, 'from liveId:', author.liveId);
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

            // Create a new array to trigger reactivity
            const chatters = Array.from(this.uniqueChatters.values());
            console.log('Total unique chatters:', chatters.length);
            this.onChattersChange?.(chatters);
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

    /**
     * Gets all chatters with a specific status
     * @param {('pinned'|'hidden')} status - The status to filter by
     * @returns {ChatAuthor[]}
     */
    getChattersWithStatus(status) {
        return this.savedChatters.filter(chatter => chatter.status === status);
    }

    /**
     * Gets pinned chatters
     * @returns {ChatAuthor[]}
     */
    getPinnedChatters() {
        return this.getChattersWithStatus('pinned');
    }

    /**
     * Gets hidden chatters
     * @returns {ChatAuthor[]}
     */
    getHiddenChatters() {
        return this.getChattersWithStatus('hidden');
    }
} 