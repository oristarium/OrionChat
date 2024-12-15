// Wait for DOM to be fully loaded
document.addEventListener('DOMContentLoaded', () => {
    const chatContainer = document.getElementById('chat-container');
    const scrollButton = document.getElementById('scroll-button');
    const languageSelect = document.getElementById('tts-language');
    let ws;

    let isUserScrolling = false;
    let scrollTimeout;

    // Scroll handling functions
    function isNearBottom() {
        const threshold = 100; // pixels from bottom
        const scrollPosition = Math.abs(chatContainer.scrollHeight - chatContainer.scrollTop - chatContainer.clientHeight);
        return scrollPosition < threshold;
    }

    function scrollToBottom() {
        chatContainer.scrollTop = chatContainer.scrollHeight;
        scrollButton.style.display = 'none';
        isUserScrolling = false;
    }

    function updateConnectionButtons(isConnected) {
        const connectBtn = document.getElementById('connect-btn');
        const disconnectBtn = document.getElementById('disconnect-btn');
        
        connectBtn.disabled = isConnected;
        disconnectBtn.disabled = !isConnected;
    }

    function connect() {
        // Clear previous chat
        chatContainer.innerHTML = '';
        
        // Disable connect button while attempting connection
        updateConnectionButtons(true);
        
        // Disconnect existing connection
        if (ws && ws.readyState === WebSocket.OPEN) {
            try {
                ws.send(JSON.stringify({ type: 'unsubscribe' }));
            } catch (error) {
                console.error('Error unsubscribing:', error);
            }
        }

        const channelId = document.getElementById('channel-id').value;
        const identifierType = document.getElementById('identifier-type').value;
        const platform = document.getElementById('platform-type').value;
        
        try {
            ws = new WebSocket('wss://chatsocket.oristarium.com/ws');
            
            ws.onopen = () => {
                document.getElementById('connection-status').className = 'connected';
                document.getElementById('connection-status').textContent = 'Connected';
                updateConnectionButtons(true);
                
                // Subscribe to the specified channel
                ws.send(JSON.stringify({
                    type: 'subscribe',
                    identifier: channelId,
                    identifierType: identifierType,
                    platform: platform
                }));
            };

            ws.onclose = () => {
                document.getElementById('connection-status').className = 'disconnected';
                document.getElementById('connection-status').textContent = 'Disconnected';
                updateConnectionButtons(false);
            };

            ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                document.getElementById('connection-status').className = 'error';
                document.getElementById('connection-status').textContent = 'Connection Error';
                updateConnectionButtons(false);
            };

            ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    const statusEl = document.getElementById('connection-status');
                    
                    switch (message.type) {
                        case 'chat':
                            if (message.data) {
                                appendMessage(message.data);
                            }
                            break;

                        case 'status':
                            switch (message.status) {
                                case 'started':
                                    statusEl.className = 'connected';
                                    statusEl.textContent = `Stream Started (${message.liveId})`;
                                    break;
                                case 'subscribed':
                                    statusEl.className = 'connected';
                                    statusEl.textContent = `Connected to ${message.identifier}`;
                                    updateConnectionButtons(true);
                                    break;
                                case 'unsubscribed':
                                    statusEl.className = 'disconnected';
                                    statusEl.textContent = 'Unsubscribed';
                                    updateConnectionButtons(false);
                                    break;
                            }
                            break;

                        case 'error':
                            statusEl.className = 'error';
                            let errorMessage = message.error;
                            
                            // Handle specific error codes
                            switch (message.code) {
                                case 'STREAM_NOT_LIVE':
                                    errorMessage = 'Stream is not live';
                                    break;
                                case 'STREAM_ENDED':
                                    errorMessage = 'Stream has ended';
                                    updateConnectionButtons(false);
                                    break;
                                case 'INVALID_MESSAGE_TYPE':
                                    errorMessage = 'Invalid message type';
                                    break;
                                case 'NO_ACTIVE_CHAT':
                                    errorMessage = 'No active chat';
                                    updateConnectionButtons(false);
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

                            statusEl.textContent = errorMessage;
                            break;
                    }
                } catch (error) {
                    console.error('Error processing message:', error);
                }
            };
        } catch (error) {
            console.error('Error creating WebSocket:', error);
            document.getElementById('connection-status').className = 'error';
            document.getElementById('connection-status').textContent = 'Connection Error';
            updateConnectionButtons(false);
        }
    }

    function disconnect() {
        if (ws) {
            try {
                ws.send(JSON.stringify({ type: 'unsubscribe' }));
                ws.close();
            } catch (error) {
                console.error('Error disconnecting:', error);
            }
            updateConnectionButtons(false);
        }
    }

    function appendMessage(message) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'chat-message';
        
        const authorName = message.data.author.display_name || message.data.author.name || 'Anonymous';
        
        // Create author name span with role classes
        let authorClasses = ['author-name'];
        if (message.data.author.roles) {
            if (message.data.author.roles.broadcaster) authorClasses.push('broadcaster');
            if (message.data.author.roles.moderator) authorClasses.push('moderator');
            if (message.data.author.roles.subscriber) authorClasses.push('subscriber');
        }
        
        // Handle different content structures
        let displayContent = '';
        let hasValidContent = false;
        
        if (typeof message.data.content === 'string') {
            displayContent = message.data.content;
            hasValidContent = false;
        } else {
            displayContent = message.data.content.rawHtml || message.data.content.formatted || message.data.content.raw || '';
            hasValidContent = message.data.content.sanitized && message.data.content.sanitized.trim().length > 0;
        }
        
        // Create buttons for TTS and Display
        const ttsButton = hasValidContent
            ? `<button class="tts-button" onclick='sendToTTS(${JSON.stringify(message).replace(/'/g, "&apos;")})'>🔉</button>`
            : '';
        
        const displayButton = `<button class="display-button" onclick='sendToDisplay(${JSON.stringify(message).replace(/'/g, "&apos;")})'>📌</button>`;
        
        messageDiv.innerHTML = `
            <span class="${authorClasses.join(' ')}">${authorName}:</span>
            <span class="message-content">${displayContent}</span>
            <div class="message-actions">
                ${ttsButton}
                ${displayButton}
            </div>
        `;
        
        chatContainer.appendChild(messageDiv);
        
        // Only auto-scroll if user isn't manually scrolling
        if (!isUserScrolling) {
            scrollToBottom();
        } else {
            scrollButton.style.display = 'flex';
        }
    }

    async function sendToDisplay(message) {
        try {
            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    message: message,
                    lang: languageSelect.value // Get current language at time of click
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
        } catch (error) {
            console.error('Error sending message:', error);
        }
    }

    async function sendToTTS(message) {
        try {
            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    message: message,
                    lang: languageSelect.value // Get current language at time of click
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
        } catch (error) {
            console.error('Error sending message to TTS:', error);
        }
    }

    // Make functions globally available
    window.connect = connect;
    window.disconnect = disconnect;
    window.sendToDisplay = sendToDisplay;
    window.sendToTTS = sendToTTS;

    // Update UI based on selected platform
    document.getElementById('platform-type').addEventListener('change', function() {
        const platform = this.value;
        const identifierType = document.getElementById('identifier-type');
        const youtubeHelp = document.getElementById('youtube-help');
        const tiktokHelp = document.getElementById('tiktok-help');
        const twitchHelp = document.getElementById('twitch-help');
        
        if (platform === 'tiktok' || platform === 'twitch') {
            identifierType.value = 'username';
            identifierType.disabled = true;
            document.getElementById('channel-id').placeholder = 
                `Enter ${platform === 'tiktok' ? 'TikTok' : 'Twitch'} username`;
            youtubeHelp.style.display = 'none';
            tiktokHelp.style.display = platform === 'tiktok' ? 'block' : 'none';
            twitchHelp.style.display = platform === 'twitch' ? 'block' : 'none';
        } else {
            identifierType.disabled = false;
            document.getElementById('channel-id').placeholder = 'Enter ID';
            youtubeHelp.style.display = 'block';
            tiktokHelp.style.display = 'none';
            twitchHelp.style.display = 'none';
        }
    });

    const platformSelect = document.getElementById('platform-type');
    const identifierType = document.getElementById('identifier-type');
    const channelInput = document.getElementById('channel-id');

    function updatePlaceholder() {
        const platform = platformSelect.value;
        const idType = identifierType.value;
        
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
        
        channelInput.placeholder = placeholder;
    }

    // Update platform color
    function updatePlatformStyle() {
        platformSelect.dataset.platform = platformSelect.value;
    }

    // Event listeners
    platformSelect.addEventListener('change', () => {
        updatePlatformStyle();
        updatePlaceholder();
        
        // Disable/enable identifier types based on platform
        if (platformSelect.value === 'tiktok' || platformSelect.value === 'twitch') {
            identifierType.value = 'username';
            identifierType.disabled = true;
        } else {
            identifierType.disabled = false;
        }
    });

    identifierType.addEventListener('change', updatePlaceholder);

    // Event listeners for scroll handling
    chatContainer.addEventListener('scroll', () => {
        if (!isNearBottom()) {
            isUserScrolling = true;
            scrollButton.style.display = 'flex';
        } else {
            isUserScrolling = false;
            scrollButton.style.display = 'none';
        }
    });

    chatContainer.addEventListener('wheel', () => {
        clearTimeout(scrollTimeout);
        
        scrollTimeout = setTimeout(() => {
            if (!isNearBottom()) {
                isUserScrolling = true;
                scrollButton.style.display = 'flex';
            }
        }, 150);
    });

    chatContainer.addEventListener('mouseleave', () => {
        if (isNearBottom()) {
            isUserScrolling = false;
            scrollButton.style.display = 'none';
        }
    });

    // Scroll button click handler
    scrollButton.addEventListener('click', scrollToBottom);

    // Initial setup
    updatePlatformStyle();
    updatePlaceholder();
}); 