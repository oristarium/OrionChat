export class UIManager {
    constructor() {
        this.platformSelect = document.getElementById('platform-type');
        this.identifierType = document.getElementById('identifier-type');
        this.channelInput = document.getElementById('channel-id');
    }

    initializePlatformUI() {
        this.updatePlatformUI(this.platformSelect.value);
    }

    updatePlatformUI(platform) {
        this.updatePlatformStyle();
        this.updatePlaceholder();
        this.updateIdentifierTypeState(platform);
    }

    updatePlatformStyle() {
        this.platformSelect.dataset.platform = this.platformSelect.value;
    }

    updatePlaceholder() {
        const platform = this.platformSelect.value;
        const idType = this.identifierType.value;
        
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
        
        this.channelInput.placeholder = placeholder;
    }

    updateIdentifierTypeState(platform) {
        if (platform === 'tiktok' || platform === 'twitch') {
            this.identifierType.value = 'username';
            this.identifierType.disabled = true;
        } else {
            this.identifierType.disabled = false;
        }
    }

    updateConnectionButtons(isConnected) {
        const connectBtn = document.getElementById('connect-btn');
        const disconnectBtn = document.getElementById('disconnect-btn');
        
        connectBtn.disabled = isConnected;
        disconnectBtn.disabled = !isConnected;
    }

    // ... other UI-related methods
} 