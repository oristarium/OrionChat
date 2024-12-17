export class UIManager {
    constructor() {
        this.$platformSelect = $('#platform-type');
        this.$identifierType = $('#identifier-type');
        this.$channelInput = $('#channel-id');
        this.$ttsProvider = $('#tts-provider');
        this.$ttsLanguage = $('#tts-language');
        
        this.setupEventListeners();
    }

    setupEventListeners() {
        if (this.$ttsProvider) {
            this.$ttsProvider.on('change', () => {
                console.log('TTS provider changed:', this.$ttsProvider.val());
            });
        }
    }

    initializePlatformUI() {
        this.updatePlatformUI(this.$platformSelect.val());
    }

    updatePlatformUI(platform) {
        this.updatePlatformStyle();
        this.updatePlaceholder();
        this.updateIdentifierTypeState(platform);
    }

    updatePlatformStyle() {
        this.$platformSelect.attr('data-platform', this.$platformSelect.val());
    }

    updatePlaceholder() {
        const platform = this.$platformSelect.val();
        const idType = this.$identifierType.val();
        
        let placeholder = '';
        switch(idType) {
            case 'username':
                placeholder = `Enter ${platform} username`;
                break;
            case 'channelId':
                placeholder = platform === 'youtube' ? 'Enter channel ID (UC...)' : 'Enter channel ID';
                break;
            case 'liveId':
                placeholder = platform === 'youtube' ? 'Enter video ID (watch?v=...)' : 'Enter live ID';
                break;
        }
        
        this.$channelInput.attr('placeholder', placeholder);
    }

    updateIdentifierTypeState(platform) {
        this.$identifierType
            .val('username')
            .prop('disabled', platform === 'tiktok' || platform === 'twitch');
    }

    updateConnectionButtons(isConnected) {
        const connectBtn = document.getElementById('connect-btn');
        const disconnectBtn = document.getElementById('disconnect-btn');
        
        connectBtn.disabled = isConnected;
        disconnectBtn.disabled = !isConnected;
    }

    // ... other UI-related methods
} 