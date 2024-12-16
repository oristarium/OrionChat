import { MessageHandler } from './MessageHandler.js';

export class ChatManager {
    constructor() {
        this.container = document.getElementById('chat-container');
        this.scrollButton = document.getElementById('scroll-button');
        this.isUserScrolling = false;
        this.scrollTimeout = null;
        this.messageHandler = new MessageHandler();
        this.ttsAllChat = false;
        
        this.setupScrollHandlers();
        this.setupMessageHandler();
        this.setupTTSToggle();
        this.setupCustomEventHandlers();
    }

    setupScrollHandlers() {
        this.container.addEventListener('scroll', () => this.handleScroll());
        this.container.addEventListener('wheel', () => this.handleWheel());
        this.container.addEventListener('mouseleave', () => this.handleMouseLeave());
        this.scrollButton.addEventListener('click', () => this.scrollToBottom());
    }

    setupMessageHandler() {
        window.addEventListener('chatMessage', (event) => {
            this.handleChatMessage(event.detail);
        });
    }

    setupTTSToggle() {
        const ttsToggle = document.getElementById('tts-all-chat');
        if (ttsToggle) {
            ttsToggle.addEventListener('change', (e) => {
                this.ttsAllChat = e.target.checked;
                console.log('TTS All Chat:', this.ttsAllChat ? 'enabled' : 'disabled');
            });
        }
    }

    setupCustomEventHandlers() {
        window.addEventListener('sendToTTS', (event) => {
            this.messageHandler.sendToTTS(event.detail);
        });

        window.addEventListener('sendToDisplay', (event) => {
            this.messageHandler.sendToDisplay(event.detail);
        });
    }

    handleScroll() {
        if (!this.isNearBottom()) {
            this.isUserScrolling = true;
            this.scrollButton.classList.remove('hidden');
        } else {
            this.isUserScrolling = false;
            this.scrollButton.classList.add('hidden');
        }
    }

    handleWheel() {
        clearTimeout(this.scrollTimeout);
        this.scrollTimeout = setTimeout(() => {
            if (!this.isNearBottom()) {
                this.isUserScrolling = true;
                this.scrollButton.classList.remove('hidden');
            }
        }, 150);
    }

    handleMouseLeave() {
        if (this.isNearBottom()) {
            this.isUserScrolling = false;
            this.scrollButton.classList.add('hidden');
        }
    }

    isNearBottom() {
        const threshold = 100;
        return Math.abs(
            this.container.scrollHeight - 
            this.container.scrollTop - 
            this.container.clientHeight
        ) < threshold;
    }

    scrollToBottom() {
        this.container.scrollTop = this.container.scrollHeight;
        this.scrollButton.classList.add('hidden');
        this.isUserScrolling = false;
    }

    appendMessage(message) {
        const messageEl = this.createMessageElement(message);
        this.container.appendChild(messageEl);
        
        if (!this.isUserScrolling) {
            this.scrollToBottom();
        } else {
            this.scrollButton.classList.remove('hidden');
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
        return message.data.author.display_name || message.data.author.name || 'Anonymous';
    }

    getAuthorClasses(message) {
        const classes = ['author-name'];
        const roles = message.data.author.roles || {};
        
        if (roles.broadcaster) classes.push('broadcaster');
        if (roles.moderator) classes.push('moderator');
        if (roles.subscriber) classes.push('subscriber');
        
        return classes;
    }

    processMessageContent(message) {
        let displayContent = '';
        let hasValidContent = false;
        
        if (typeof message.data.content === 'string') {
            displayContent = message.data.content;
            hasValidContent = displayContent.trim().length > 0;
        } else {
            displayContent = message.data.content.rawHtml || 
                           message.data.content.formatted || 
                           message.data.content.raw || '';
            hasValidContent = message.data.content.sanitized && 
                            message.data.content.sanitized.trim().length > 0;
        }
        
        return { displayContent, hasValidContent };
    }

    generateMessageHTML(message, authorName, authorClasses, displayContent, hasValidContent) {
        const ttsButton = hasValidContent
            ? `<button class="tts-button" onclick='window.dispatchEvent(new CustomEvent("sendToTTS", { detail: ${JSON.stringify(message).replace(/'/g, "&apos;")} }))'><span>🔉</span></button>`
            : '';
        
        const displayButton = `<button class="display-button" onclick='window.dispatchEvent(new CustomEvent("sendToDisplay", { detail: ${JSON.stringify(message).replace(/'/g, "&apos;")} }))'><span>📌</span></button>`;
        
        return `
            <div class="message-main">
                <span class="${authorClasses.join(' ')}">${authorName}:</span>
                <span class="message-content">${displayContent}</span>
            </div>
            <div class="message-actions">
                ${ttsButton}
                ${displayButton}
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
                await this.messageHandler.sendToTTS({
                    type: 'tts',
                    data: {
                        message: message,
                        voice_id: this.messageHandler.languageSelect.value || 'en',
                        voice_provider: this.messageHandler.providerSelect.value || 'google'
                    }
                });
            }

            this.appendMessage(message);
        } catch (error) {
            console.error('Error handling chat message:', error);
        }
    }
} 