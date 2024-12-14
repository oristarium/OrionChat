// Wait for DOM to be fully loaded
document.addEventListener('DOMContentLoaded', () => {
    const chatContainer = document.getElementById('chat-container');
    const languageSelect = document.getElementById('tts-language');
    let ws;

    function connect() {
        // Clear previous chat
        chatContainer.innerHTML = '';
        
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
            };

            ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                document.getElementById('connection-status').className = 'error';
                document.getElementById('connection-status').textContent = 'Connection Error';
            };

            ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    switch (message.type) {
                        case 'chat':
                            if (message.data) {
                                appendMessage(message.data);
                            }
                            break;
                        case 'error':
                            const statusEl = document.getElementById('connection-status');
                            statusEl.textContent = `Error: ${message.error}`;
                            statusEl.className = 'error';
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
        }
    }

    function appendMessage(message) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'chat-message';
        
        const authorName = message.data.author.display_name || message.data.author.name || 'Anonymous';
        
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
            ? `<button class="tts-button" onclick='sendToTTS(${JSON.stringify({
                message: message,
                lang: languageSelect.value
              }).replace(/'/g, "&apos;")})'>🔉</button>`
            : '';
        
        const displayButton = `<button class="display-button" onclick='sendToDisplay(${JSON.stringify({
            message: message,
            lang: languageSelect.value
          }).replace(/'/g, "&apos;")})'>📌</button>`;
        
        messageDiv.innerHTML = `
            <span class="author-name">${authorName}:</span>
            <span class="message-content">${displayContent}</span>
            <div class="message-actions">
                ${ttsButton}
                ${displayButton}
            </div>
        `;
        
        chatContainer.appendChild(messageDiv);
        chatContainer.scrollTop = chatContainer.scrollHeight;
    }

    async function sendToDisplay(data) {
        try {
            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
        } catch (error) {
            console.error('Error sending message:', error);
        }
    }

    async function sendToTTS(data) {
        try {
            const response = await fetch('/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
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
}); 