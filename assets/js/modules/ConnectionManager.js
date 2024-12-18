export class ConnectionManager {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.$connectBtn = $('#connect-btn');
        this.$disconnectBtn = $('#disconnect-btn');
        this.$status = $('#connection-status');
    }

    toggleConnectionButtons(isConnected) {
        if (isConnected) {
            this.$connectBtn.addClass('hidden');
            this.$disconnectBtn.removeClass('hidden');
        } else {
            this.$connectBtn.removeClass('hidden').prop('disabled', false);
            this.$disconnectBtn.addClass('hidden');
        }
        this.isConnected = isConnected;
    }

    async connect() {
        const channelId = $('#channel-id').val();
        const identifierType = $('#identifier-type').val();
        const platform = $('#platform-type').val();
        
        try {
            await this.initializeWebSocket(channelId, identifierType, platform);
        } catch (error) {
            this.handleConnectionError(error);
        }
    }

    async initializeWebSocket(channelId, identifierType, platform) {
        // Clear previous connection
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify({ type: 'unsubscribe' }));
            } catch (error) {
                console.error('Error unsubscribing:', error);
            }
        }

        this.ws = new WebSocket('wss://chatsocket.oristarium.com/ws');
        
        this.ws.onopen = () => this.handleOpen(channelId, identifierType, platform);
        this.ws.onclose = () => this.handleClose();
        this.ws.onerror = (error) => this.handleError(error);
        this.ws.onmessage = (event) => this.handleMessage(event);
    }

    handleOpen(channelId, identifierType, platform) {
        this.updateStatus('connected', 'Connected');
        this.toggleConnectionButtons(true);
        
        this.ws.send(JSON.stringify({
            type: 'subscribe',
            identifier: channelId,
            identifierType: identifierType,
            platform: platform
        }));
    }

    handleClose() {
        this.updateStatus('disconnected', 'Disconnected');
        this.toggleConnectionButtons(false);
    }

    handleError(error) {
        console.error('WebSocket error:', error);
        this.updateStatus('disconnected', 'Disconnected');
        this.toggleConnectionButtons(false);
        $.toast('Connection Error');
    }

    handleConnectionError(error) {
        console.error('Connection error:', error);
        this.updateStatus('disconnected', 'Disconnected');
        this.toggleConnectionButtons(false);
        $.toast('Connection Error');
    }

    handleMessage(event) {
        try {
            const message = JSON.parse(event.data);
            this.processMessage(message);
        } catch (error) {
            console.error('Error processing message:', error);
        }
    }

    processMessage(message) {
        switch (message.type) {
            case 'chat':
                if (message.data) {
                    // Emit event for chat manager to handle
                    window.dispatchEvent(new CustomEvent('chatMessage', { detail: message.data }));
                }
                break;
            case 'status':
                this.handleStatusMessage(message);
                break;
            case 'error':
                this.handleErrorMessage(message);
                break;
        }
    }

    updateStatus(className, text) {
        this.$status
            .removeClass()
            .addClass(`status-indicator ${className}`)
            .text(text);
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        
        this.toggleConnectionButtons(false);
        this.$status.text('Disconnected').removeClass('connected').addClass('disconnected');
        $.toast('Disconnected from chat');
    }

    handleStatusMessage(message) {
        switch (message.status) {
            case 'started':
                this.updateStatus('connected', 'Connected');
                $.toast(`Stream Started (${message.liveId})`);
                break;
            case 'subscribed':
                this.updateStatus('connected', 'Connected');
                $.toast(`Connected to ${message.identifier}`);
                this.toggleConnectionButtons(true);
                break;
            case 'unsubscribed':
                this.updateStatus('disconnected', 'Disconnected');
                $.toast('Unsubscribed from chat');
                this.toggleConnectionButtons(false);
                break;
            default:
                console.warn('Unknown status message:', message);
        }
    }

    handleErrorMessage(message) {
        let errorMessage = message.error;
        
        // Handle specific error codes
        switch (message.code) {
            case 'STREAM_NOT_LIVE':
                errorMessage = 'Stream is not live';
                break;
            case 'STREAM_ENDED':
                errorMessage = 'Stream has ended';
                break;
            case 'INVALID_MESSAGE_TYPE':
                errorMessage = 'Invalid message type';
                break;
            case 'NO_ACTIVE_CHAT':
                errorMessage = 'No active chat';
                break;
            case 'STREAM_NOT_FOUND':
                errorMessage = 'Stream not found';
                break;
            case 'UNSUPPORTED_PLATFORM':
                errorMessage = 'Platform not supported';
                break;
        }

        // Add details if present
        if (message.details) {
            errorMessage += `: ${message.details}`;
        }

        this.updateStatus('disconnected', 'Disconnected');
        this.toggleConnectionButtons(false);
        $.toast(errorMessage);
    }
} 