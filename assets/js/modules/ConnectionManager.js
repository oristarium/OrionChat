export class ConnectionManager {
    constructor() {
        this.ws = null;
        this.statusElement = document.getElementById('connection-status');
    }

    async connect() {
        const channelId = document.getElementById('channel-id').value;
        const identifierType = document.getElementById('identifier-type').value;
        const platform = document.getElementById('platform-type').value;
        
        try {
            await this.initializeWebSocket(channelId, identifierType, platform);
        } catch (error) {
            this.handleConnectionError(error);
        }

        document.getElementById('connect-btn').classList.add('hidden');
        document.getElementById('disconnect-btn').classList.remove('hidden');
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
        this.updateConnectionState(true);
        
        this.ws.send(JSON.stringify({
            type: 'subscribe',
            identifier: channelId,
            identifierType: identifierType,
            platform: platform
        }));
    }

    handleClose() {
        this.updateStatus('disconnected', 'Disconnected');
        this.updateConnectionState(false);
    }

    handleError(error) {
        console.error('WebSocket error:', error);
        this.updateStatus('error', 'Connection Error');
        this.updateConnectionState(false);
    }

    handleConnectionError(error) {
        console.error('Connection error:', error);
        this.updateStatus('error', 'Connection Error');
        this.updateConnectionState(false);
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
        this.statusElement.className = `status-indicator ${className}`;
        this.statusElement.textContent = text;
    }

    updateConnectionState(isConnected) {
        const connectBtn = document.getElementById('connect-btn');
        const disconnectBtn = document.getElementById('disconnect-btn');
        
        connectBtn.disabled = isConnected;
        disconnectBtn.disabled = !isConnected;
    }

    disconnect() {
        if (this.ws) {
            try {
                this.ws.send(JSON.stringify({ type: 'unsubscribe' }));
                this.ws.close();
            } catch (error) {
                console.error('Error disconnecting:', error);
            }
            this.updateConnectionState(false);
        }

        document.getElementById('connect-btn').classList.remove('hidden');
        document.getElementById('disconnect-btn').classList.add('hidden');
    }

    handleStatusMessage(message) {
        switch (message.status) {
            case 'started':
                this.updateStatus('connected', `Stream Started (${message.liveId})`);
                break;
            case 'subscribed':
                this.updateStatus('connected', `Connected to ${message.identifier}`);
                this.updateConnectionState(true);
                break;
            case 'unsubscribed':
                this.updateStatus('disconnected', 'Unsubscribed');
                this.updateConnectionState(false);
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
                this.updateConnectionState(false);
                break;
            case 'INVALID_MESSAGE_TYPE':
                errorMessage = 'Invalid message type';
                break;
            case 'NO_ACTIVE_CHAT':
                errorMessage = 'No active chat';
                this.updateConnectionState(false);
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

        this.updateStatus('error', errorMessage);
    }
} 