class AvatarManager {
    constructor() {
        this.avatars = new Map();
    }

    async initialize() {
        try {
            const { avatars } = await $.get('/api/avatars/active');
            this.updateAvatars(avatars);
        } catch (error) {
            console.error('Error initializing avatars:', error);
        }
    }

    updateAvatars(avatarsData) {
        window.ELEMENTS.$avatarsWrapper.empty();
        this.avatars.clear();

        avatarsData.forEach(avatarData => {
            const avatar = new Avatar(avatarData);
            const $element = avatar.createElements();
            window.ELEMENTS.$avatarsWrapper.append($element);
            this.avatars.set(avatar.id, avatar);
        });
    }

    getRandomAvatar() {
        const avatars = Array.from(this.avatars.values());
        return avatars[Math.floor(Math.random() * avatars.length)];
    }

    async processTTSMessage(chatMessage) {
        const selectedAvatar = this.getRandomAvatar();
        if (!selectedAvatar) {
            console.error('No avatars available');
            return;
        }

        await ttsManager.processTTSMessage(chatMessage, selectedAvatar);
    }
}

class Avatar {
    constructor(data) {
        this.id = data.id;
        this.name = data.name;
        this.states = data.states;
        this.tts_voices = data.tts_voices || [];
        this.element = null;
        this.audioPlayer = new Audio();
        this.setupAudioPlayer();
    }

    setupAudioPlayer() {
        $(this.audioPlayer)
            .on('play', () => {
                console.log(`Avatar ${this.id} started speaking`);
                this.updateUIState(true);
            })
            .on('ended', () => {
                console.log(`Avatar ${this.id} finished speaking`);
                this.updateUIState(false);
            })
            .on('pause', () => {
                console.log(`Avatar ${this.id} paused speaking`);
                this.updateUIState(false);
            })
            .on('error', () => {
                console.error(`Avatar ${this.id} audio error`);
                this.updateUIState(false);
            });
    }

    updateMessageContent(message) {
        if (!this.element) return;

        // Validate required content field
        if (!message.data?.content?.sanitized) {
            console.error('Invalid message structure - missing required content');
            return;
        }

        // Get the display text - prioritize rawHtml, then fall back to other formats
        const content = message.data.content;
        if (content.rawHtml) {
            this.element.$messageBubble.html(content.rawHtml);
        } else {
            const displayText = content.formatted || content.raw || content.sanitized;
            this.element.$messageBubble.text(displayText);
        }

        // Get author information
        const author = message.data?.author || {};
        const authorName = author.display_name || author.username || '';

        // Handle author display
        if (authorName) {
            this.element.$username.text(authorName);
            this.element.$username.removeClass('hidden');
        } else {
            this.element.$username.addClass('hidden');
        }
    }

    createElements() {
        const $template = window.ELEMENTS.$avatarTemplate.html();
        const $container = $($.parseHTML($template));
        $container.attr('data-avatar-id', this.id);
        
        // Get references to elements
        const $messageBubble = $container.find('.message-bubble');
        const $username = $container.find('.username');
        const $avatar = $container.find('.avatar-image');
        const $avatarWrapper = $container.find('.avatar-wrapper');

        // Get configuration values
        const messageFontSize = $('#messageFontSize').val();
        const authorFontSize = $('#authorFontSize').val();
        const avatarSize = $('#avatarSize').val();
        const messageHPadding = $('#messageHPadding').val();
        const messageVPadding = $('#messageVPadding').val();
        const justify = window.configManager?.configInputs.containerPosition.value.justify || 'end';
        const avatarGap = $('#avatarGap').val();

        // Apply styles
        $container.css({
            'width': `${avatarSize}px`,
            'margin-left': justify === 'center' ? `${avatarGap/2}px` : 
                        justify === 'end' ? `${avatarGap}px` : '0',
            'margin-right': justify === 'center' ? `${avatarGap/2}px` : 
                        justify === 'start' ? `${avatarGap}px` : '0'
        });

        $messageBubble.css({
            'fontSize': `${messageFontSize}px`,
            'padding-left': `${messageHPadding}em`,
            'padding-right': `${messageHPadding}em`,
            'padding-top': `${messageVPadding}em`,
            'padding-bottom': `${messageVPadding}em`
        });

        $username.css('fontSize', `${authorFontSize}px`);
        $avatar.attr('src', this.states.idle);

        // Store element references
        this.element = {
            $container,
            $messageBubble,
            $username,
            $avatar,
            $avatarWrapper
        };

        return $container;
    }

    updateUIState(isPlaying) {
        this.element.$avatar.attr('src', this.states[isPlaying ? 'talking' : 'idle']);
        this.element.$avatarWrapper.toggleClass('animate-talking', isPlaying);
        this.element.$messageBubble.toggleClass('show', isPlaying);
        // Only toggle username if it has content and is not hidden
        if (this.element.$username.text() && !this.element.$username.hasClass('hidden')) {
            this.element.$username.toggleClass('show', isPlaying);
        }
    }

    getRandomVoice() {
        if (!this.tts_voices || this.tts_voices.length === 0) {
            console.error('No TTS voices configured for avatar:', this.id);
            return null;
        }

        const randomVoice = this.tts_voices[Math.floor(Math.random() * this.tts_voices.length)];
        console.log('Selected voice for avatar:', this.id, randomVoice);
        
        return {
            voice_id: randomVoice.voice_id,
            provider: randomVoice.provider
        };
    }
}

// Export the AvatarManager
window.AvatarManager = AvatarManager; 