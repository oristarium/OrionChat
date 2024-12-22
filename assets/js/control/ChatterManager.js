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
        /** @type {function(string, string=): void} */
        this.showToast = null;
        /** @type {boolean} */
        this.shouldShowChatterToasts = false;
    }

    /**
     * Initializes the ChatterManager
     */
    init() {
        // Enable chatter toasts after 3 seconds
        setTimeout(() => {
            this.shouldShowChatterToasts = true;
        }, 3000);
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