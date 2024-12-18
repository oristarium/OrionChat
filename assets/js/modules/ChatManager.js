import { MessageHandler } from './MessageHandler.js';

export class ChatManager {
    constructor() {
        this.$container = $('#chat-container');
        this.$scrollButton = $('#scroll-button');
        this.isUserScrolling = false;
        this.scrollTimeout = null;
        this.messageHandler = new MessageHandler();
        this.ttsAllChat = false;
        
        // Initialize saved messages
        this.savedMessages = JSON.parse(localStorage.getItem('savedMessages') || '[]');
        this.initializeSavedMessages();
        
        this.setupScrollHandlers();
        this.setupMessageHandler();
        this.setupTTSToggle();
        this.setupCustomEventHandlers();
    }

    initializeSavedMessages() {
        this.renderSavedMessages();
    }

    renderSavedMessages() {
        const container = document.getElementById('saved-messages-container');
        if (!container) return;

        try {
            if (!this.savedMessages || this.savedMessages.length === 0) {
                container.innerHTML = `
                    <div class="no-saved-messages">
                        No saved messages yet
                    </div>
                `;
                return;
            }

            container.innerHTML = this.savedMessages
                .map((encodedMessage, index) => this.createSavedMessageElement(encodedMessage, index))
                .join('');
        } catch (error) {
            console.error('Error rendering saved messages:', error);
            container.innerHTML = `
                <div class="error-message">
                    Error loading saved messages
                </div>
            `;
        }
    }

    createSavedMessageElement(encodedMessage, index) {
        try {
            const message = JSON.parse(atob(encodedMessage));
            const messageDiv = document.createElement('div');
            messageDiv.className = 'chat-message';
            
            const authorName = this.getAuthorName(message);
            const authorClasses = this.getAuthorClasses(message);
            const { displayContent, hasValidContent } = this.processMessageContent(message);
            
            const ttsButton = hasValidContent
                ? `<button class="tts-button" onclick="window.chatManager.handleTTS('${encodedMessage}')"><span>🔉</span></button>`
                : '';
            
            const displayButton = `<button class="display-button" onclick="window.chatManager.handleDisplay('${encodedMessage}')"><span>📌</span></button>`;
            
            const deleteButton = `<button class="delete-button" onclick="window.chatManager.deleteMessage(${index})"><span>🗑️</span></button>`;

            messageDiv.innerHTML = `
                <div class="message-main">
                    <span class="${authorClasses.join(' ')}">${this.cleanString(authorName)}:</span>
                    <span class="message-content">${displayContent}</span>
                </div>
                <div class="message-actions">
                    ${ttsButton}
                    ${displayButton}
                    ${deleteButton}
                </div>
            `;
            
            return messageDiv.outerHTML;
        } catch (error) {
            console.error('Error creating saved message element:', error);
            return `<div class="chat-message error">Error loading message</div>`;
        }
    }

    saveMessage(message) {
        try {
            if (!message || !message.data) {
                console.error('Invalid message structure:', message);
                $.toast('Error: Invalid message structure');
                return;
            }

            this.savedMessages.push(message);
            localStorage.setItem('savedMessages', JSON.stringify(this.savedMessages));
            this.renderSavedMessages();
            $.toast('Message saved');
        } catch (error) {
            console.error('Error saving message:', error);
            $.toast('Error saving message');
        }
    }

    deleteMessage(index) {
        try {
            if (index >= 0 && index < this.savedMessages.length) {
                this.savedMessages.splice(index, 1);
                localStorage.setItem('savedMessages', JSON.stringify(this.savedMessages));
                this.renderSavedMessages();
                $.toast('Message deleted');
            }
        } catch (error) {
            console.error('Error deleting message:', error);
            $.toast('Error deleting message');
        }
    }

    setupScrollHandlers() {
        this.$container
            .on('scroll', () => this.handleScroll())
            .on('wheel', () => this.handleWheel())
            .on('mouseleave', () => this.handleMouseLeave());
        
        this.$scrollButton.on('click', () => this.scrollToBottom());
    }

    setupMessageHandler() {
        window.addEventListener('chatMessage', (event) => {
            this.handleChatMessage(event.detail);
        });
    }

    setupTTSToggle() {
        $('#tts-all-chat').on('change', (e) => {
            this.ttsAllChat = e.target.checked;
            console.log('TTS All Chat:', this.ttsAllChat ? 'enabled' : 'disabled');
            $.toast(this.ttsAllChat ? 'TTS All Chat enabled' : 'TTS All Chat disabled');
        });
    }

    setupCustomEventHandlers() {
        $(window).on('sendToTTS', (event) => {
            this.messageHandler.sendToTTS(event.originalEvent.detail);
        });

        $(window).on('sendToDisplay', (event) => {
            this.messageHandler.sendToDisplay(event.originalEvent.detail);
        });
    }

    handleScroll() {
        if (!this.isNearBottom()) {
            this.isUserScrolling = true;
            this.$scrollButton.removeClass('hidden');
        } else {
            this.isUserScrolling = false;
            this.$scrollButton.addClass('hidden');
        }
    }

    handleWheel() {
        clearTimeout(this.scrollTimeout);
        this.scrollTimeout = setTimeout(() => {
            if (!this.isNearBottom()) {
                this.isUserScrolling = true;
                this.$scrollButton.removeClass('hidden');
            }
        }, 150);
    }

    handleMouseLeave() {
        if (this.isNearBottom()) {
            this.isUserScrolling = false;
            this.$scrollButton.addClass('hidden');
        }
    }

    isNearBottom() {
        const threshold = 100;
        const $container = this.$container;
        return Math.abs(
            $container[0].scrollHeight - 
            $container.scrollTop() - 
            $container.height()
        ) < threshold;
    }

    scrollToBottom() {
        this.$container.scrollTop(this.$container[0].scrollHeight);
        this.$scrollButton.addClass('hidden');
        this.isUserScrolling = false;
    }

    appendMessage(message) {
        const $messageEl = $(this.createMessageElement(message));
        this.$container.append($messageEl);
        
        if (!this.isUserScrolling) {
            this.scrollToBottom();
        } else {
            this.$scrollButton.removeClass('hidden');
        }
    }

    createMessageElement(message) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'chat-message';
        
        const authorName = this.getAuthorName(message);
        const authorClasses = this.getAuthorClasses(message);
        const { displayContent, hasValidContent } = this.processMessageContent(message);
        
        messageDiv.innerHTML = this.generateMessageHTML(message, authorName, authorClasses, displayContent, hasValidContent);
        
        return messageDiv;
    }

    getAuthorName(message) {
        if (!message || !message.data || !message.data.author) {
            return 'Anonymous';
        }
        return message.data.author.display_name || message.data.author.name || 'Anonymous';
    }

    getAuthorClasses(message) {
        const classes = ['author-name'];
        
        if (message && message.data && message.data.author && message.data.author.roles) {
            const roles = message.data.author.roles;
            if (roles.broadcaster) classes.push('broadcaster');
            if (roles.moderator) classes.push('moderator');
            if (roles.subscriber) classes.push('subscriber');
        }
        
        return classes;
    }

    processMessageContent(message) {
        let displayContent = '';
        let hasValidContent = false;
        
        if (!message || !message.data) {
            return { displayContent, hasValidContent };
        }
        
        if (typeof message.data.content === 'string') {
            displayContent = message.data.content;
            hasValidContent = displayContent.trim().length > 0;
        } else if (message.data.content) {
            // Use rawHtml directly if it exists, otherwise create safe HTML
            displayContent = message.data.content.rawHtml || 
                            this.cleanString(message.data.content.formatted || 
                            message.data.content.raw || '');
            hasValidContent = message.data.content.sanitized && 
                            message.data.content.sanitized.trim().length > 0;
        }
        
        return { displayContent, hasValidContent };
    }

    generateMessageHTML(message, authorName, authorClasses, displayContent, hasValidContent) {
        // Base64 encode the message data
        const encodedMessage = btoa(JSON.stringify(message));

        const ttsButton = hasValidContent
            ? `<button class="tts-button" onclick="window.chatManager.handleTTS('${encodedMessage}')"><span>🔉</span></button>`
            : '';
        
        const displayButton = `<button class="display-button" onclick="window.chatManager.handleDisplay('${encodedMessage}')"><span>📌</span></button>`;
        
        const saveButton = `<button class="save-button" onclick="window.chatManager.saveMessage('${encodedMessage}')"><span>💾</span></button>`;

        return `
            <div class="message-main">
                <span class="${authorClasses.join(' ')}">${this.cleanString(authorName)}:</span>
                <span class="message-content">${displayContent}</span>
            </div>
            <div class="message-actions">
                ${ttsButton}
                ${displayButton}
                ${saveButton}
            </div>
        `;
    }

    canMessageBeTTSed(message) {
        const content = message.data.content;
        return content && content.sanitized && content.sanitized.trim() !== '';
    }

    async handleChatMessage(message) {
        try {
            if (this.ttsAllChat && this.canMessageBeTTSed(message)) {
                console.log('TTS All Chat enabled, sending to TTS:', message);
                await this.messageHandler.sendToTTS(message);
                $.toast('Message sent to TTS');
            }

            this.appendMessage(message);
        } catch (error) {
            console.error('Error handling chat message:', error);
            $.toast('Error handling chat message');
        }
    }

    // Helper method to clean strings
    cleanString(str) {
        if (!str) return '';
        return str
            .replace(/&/g, '&amp;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/=/g, '&#61;');
    }

    // Add these new handler methods
    handleTTS(encodedMessage) {
        try {
            const message = JSON.parse(atob(encodedMessage));
            window.dispatchEvent(new CustomEvent('sendToTTS', { detail: message }));
        } catch (error) {
            console.error('Error handling TTS:', error);
            $.toast('Error sending message to TTS');
        }
    }

    handleDisplay(encodedMessage) {
        try {
            const message = JSON.parse(atob(encodedMessage));
            window.dispatchEvent(new CustomEvent('sendToDisplay', { detail: message }));
        } catch (error) {
            console.error('Error handling display:', error);
            $.toast('Error sending message to display');
        }
    }

    saveMessage(encodedMessage) {
        try {
            const message = JSON.parse(atob(encodedMessage));
            if (!message || !message.data) {
                console.error('Invalid message structure:', message);
                $.toast('Error: Invalid message structure');
                return;
            }

            this.savedMessages.push(encodedMessage); // Store the encoded version
            localStorage.setItem('savedMessages', JSON.stringify(this.savedMessages));
            this.renderSavedMessages();
            $.toast('Message saved');
        } catch (error) {
            console.error('Error saving message:', error);
            $.toast('Error saving message');
        }
    }

    // Add these utility methods to the ChatManager class
    encodeMessage(message) {
        try {
            // Convert the message to a JSON string
            const jsonString = JSON.stringify(message);
            // Convert to base64 while handling Unicode characters
            return btoa(encodeURIComponent(jsonString));
        } catch (error) {
            console.error('Error encoding message:', error);
            return null;
        }
    }

    decodeMessage(encodedMessage) {
        try {
            // Decode base64 and handle Unicode characters
            const jsonString = decodeURIComponent(atob(encodedMessage));
            return JSON.parse(jsonString);
        } catch (error) {
            console.error('Error decoding message:', error);
            return null;
        }
    }

    // Update the generateMessageHTML method
    generateMessageHTML(message, authorName, authorClasses, displayContent, hasValidContent) {
        // Base64 encode the message data with Unicode support
        const encodedMessage = this.encodeMessage(message);
        if (!encodedMessage) {
            console.error('Failed to encode message');
            return '';
        }

        const ttsButton = hasValidContent
            ? `<button class="tts-button" onclick="window.chatManager.handleTTS('${encodedMessage}')"><span>🔉</span></button>`
            : '';
        
        const displayButton = `<button class="display-button" onclick="window.chatManager.handleDisplay('${encodedMessage}')"><span>📌</span></button>`;
        
        const saveButton = `<button class="save-button" onclick="window.chatManager.saveMessage('${encodedMessage}')"><span>💾</span></button>`;

        return `
            <div class="message-main">
                <span class="${authorClasses.join(' ')}">${this.cleanString(authorName)}:</span>
                <span class="message-content">${displayContent}</span>
            </div>
            <div class="message-actions">
                ${ttsButton}
                ${displayButton}
                ${saveButton}
            </div>
        `;
    }

    // Update the handler methods to use the new decode function
    handleTTS(encodedMessage) {
        try {
            const message = this.decodeMessage(encodedMessage);
            if (!message) {
                $.toast('Error decoding message');
                return;
            }
            window.dispatchEvent(new CustomEvent('sendToTTS', { detail: message }));
        } catch (error) {
            console.error('Error handling TTS:', error);
            $.toast('Error sending message to TTS');
        }
    }

    handleDisplay(encodedMessage) {
        try {
            const message = this.decodeMessage(encodedMessage);
            if (!message) {
                $.toast('Error decoding message');
                return;
            }
            window.dispatchEvent(new CustomEvent('sendToDisplay', { detail: message }));
        } catch (error) {
            console.error('Error handling display:', error);
            $.toast('Error sending message to display');
        }
    }

    saveMessage(encodedMessage) {
        try {
            const message = this.decodeMessage(encodedMessage);
            if (!message || !message.data) {
                console.error('Invalid message structure:', message);
                $.toast('Error: Invalid message structure');
                return;
            }

            this.savedMessages.push(encodedMessage); // Store the encoded version
            localStorage.setItem('savedMessages', JSON.stringify(this.savedMessages));
            this.renderSavedMessages();
            $.toast('Message saved');
        } catch (error) {
            console.error('Error saving message:', error);
            $.toast('Error saving message');
        }
    }

    // Update the createSavedMessageElement method
    createSavedMessageElement(encodedMessage, index) {
        try {
            const message = this.decodeMessage(encodedMessage);
            if (!message) {
                return `<div class="chat-message error">Error decoding message</div>`;
            }

            const messageDiv = document.createElement('div');
            messageDiv.className = 'chat-message';
            
            const authorName = this.getAuthorName(message);
            const authorClasses = this.getAuthorClasses(message);
            const { displayContent, hasValidContent } = this.processMessageContent(message);
            
            const ttsButton = hasValidContent
                ? `<button class="tts-button" onclick="window.chatManager.handleTTS('${encodedMessage}')"><span>🔉</span></button>`
                : '';
            
            const displayButton = `<button class="display-button" onclick="window.chatManager.handleDisplay('${encodedMessage}')"><span>📌</span></button>`;
            
            const deleteButton = `<button class="delete-button" onclick="window.chatManager.deleteMessage(${index})"><span>🗑️</span></button>`;

            messageDiv.innerHTML = `
                <div class="message-main">
                    <span class="${authorClasses.join(' ')}">${this.cleanString(authorName)}:</span>
                    <span class="message-content">${displayContent}</span>
                </div>
                <div class="message-actions">
                    ${ttsButton}
                    ${displayButton}
                    ${deleteButton}
                </div>
            `;
            
            return messageDiv.outerHTML;
        } catch (error) {
            console.error('Error creating saved message element:', error);
            return `<div class="chat-message error">Error loading message</div>`;
        }
    }
} 