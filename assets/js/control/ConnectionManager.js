const dbName = 'OrionConnectionsDB';
const dbVersion = 2;
let db;

export class ConnectionManager {
    constructor() {
        this.ws = null;
        this.savedConnections = [];
        this.isDBReady = false;
        this.onConnectionsChange = null;
        this.onMessageReceived = null;
        this.showToast = null;
        this.chatManager = null;
        this.isConnecting = false;
    }

    get activeConnections() {
        return this.savedConnections.filter(conn => conn.status === 'connected');
    }

    async init(chatManager) {
        try {
            this.chatManager = chatManager;
            await this.initDB();
            this.isDBReady = true;
            await this.initWebSocket();
            if (this.ws?.readyState === WebSocket.OPEN) {
                await this.loadSavedConnections();
            }
        } catch (error) {
            console.error('Failed to initialize ConnectionManager:', error);
        }
    }

    async initDB() {
        return new Promise((resolve, reject) => {
            console.log('Initializing Connections IndexedDB...');
            const request = indexedDB.open(dbName, dbVersion);

            request.onerror = (event) => {
                console.error('Connections IndexedDB error:', event.target.error);
                reject(event.target.error);
            };

            request.onsuccess = (event) => {
                console.log('Connections IndexedDB initialized successfully');
                db = event.target.result;
                resolve(db);
            };

            request.onupgradeneeded = (event) => {
                console.log('Creating/upgrading Connections IndexedDB structure...');
                const db = event.target.result;
                
                if (!db.objectStoreNames.contains('connections')) {
                    const store = db.createObjectStore('connections', { keyPath: 'id' });
                    console.log('Connections store created');
                }
            };
        });
    }

    initWebSocket() {
        return new Promise((resolve, reject) => {
            if (this.ws?.readyState === WebSocket.OPEN) {
                resolve(this.ws);
                return;
            }

            // const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `wss://chatsocket.oristarium.com/ws`;
            
            console.log('Connecting to WebSocket:', wsUrl);
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected successfully');
                resolve(this.ws);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket connection error:', error);
                this.showToast?.('WebSocket connection error', 'error');
                reject(error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected, attempting to reconnect in 5 seconds...');
                // Mark all connections as disconnected
                this.savedConnections.forEach(conn => {
                    conn.status = 'disconnected';
                });
                this.onConnectionsChange?.(this.savedConnections);
                
                this.ws = null;
                setTimeout(() => {
                    console.log('Attempting WebSocket reconnection...');
                    this.initWebSocket()
                        .then(() => {
                            console.log('WebSocket reconnected, resubscribing to saved connections...');
                            this.loadSavedConnections();
                        })
                        .catch(error => {
                            console.error('WebSocket reconnection failed:', error);
                        });
                }, 5000);
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    // console.log('WebSocket received:', data);
                    
                    switch (data.type) {
                        case 'chat':
                            this.onMessageReceived?.(data);
                            break;
                            
                        case 'status':
                            this.handleStatusMessage(data);
                            break;
                            
                        case 'error':
                            if (data.code === 'ALREADY_SUBSCRIBED') {
                                this.showToast?.(data.error, 'info');
                            } else {
                                console.error('WebSocket error message:', data);
                                // only show error if code is provided
                                if (data.code) {
                                    this.showToast?.(data.error, 'error');
                                }
                                
                                // Find any connecting connections and mark them as error
                                const connection = this.savedConnections.find(c => c.status === 'connecting');
                                if (connection) {
                                    connection.status = data.error || 'error';
                                    this.onConnectionsChange?.(this.savedConnections);
                                }
                            }
                            break;

                        default:
                            console.log('Unknown message type:', data.type);
                    }
                } catch (error) {
                    console.error('Error processing WebSocket message:', error);
                }
            };
        });
    }

    handleStatusMessage(data) {
        console.log('Status message:', data);
        
        if (data.status === 'subscribed') {
            const connection = this.savedConnections.find(c => c.id.toLowerCase() === data.liveId.toLowerCase());
            if (connection) {
                connection.status = 'connected';
                this.onConnectionsChange?.(this.savedConnections);
                this.showToast?.(`Successfully connected to ${connection.platform} chat: ${connection.identifier}`, 'success');
            }
        } else if (data.status === 'unsubscribed') {
            const connection = this.savedConnections.find(c => c.id.toLowerCase() === data.liveId.toLowerCase());
            if (connection) {
                connection.status = 'disconnected';
                this.onConnectionsChange?.(this.savedConnections);
                this.showToast?.(`Disconnected from ${connection.platform} chat: ${connection.identifier}`, 'info');
            }
        }
    }

    generateConnectionId(platform, identifier, identifierType = 'username') {
        return `${platform}-${identifierType}-${identifier}`;
    }

    async connectNewChat(connectionDetails) {
        if (this.isConnecting) return;
        this.isConnecting = true;

        try {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                console.log('WebSocket not connected, attempting to connect...');
                await this.initWebSocket();
            }

            const connId = this.generateConnectionId(
                connectionDetails.platform,
                connectionDetails.identifier,
                connectionDetails.identifierType
            );

            // Check if connection exists
            const existingConnection = this.savedConnections.find(conn => conn.id === connId);
            if (existingConnection) {
                if (existingConnection.status === 'connected') {
                    console.log('Already connected to:', connId);
                    this.showToast?.('Already connected to this chat', 'info');
                    this.isConnecting = false;
                    return;
                }
                existingConnection.status = 'connecting';
                this.onConnectionsChange?.(this.savedConnections);
            } else {
                // Add new connection with connecting status
                const newConnection = {
                    id: connId,
                    ...connectionDetails,
                    status: 'connecting'
                };
                this.savedConnections.push(newConnection);
                this.onConnectionsChange?.(this.savedConnections);
            }

            // Send subscription request
            this.ws.send(JSON.stringify({
                type: 'subscribe',
                platform: connectionDetails.platform,
                identifier: connectionDetails.identifier,
                identifierType: connectionDetails.identifierType
            }));

            // Save to database
            if (this.isDBReady) {
                const transaction = db.transaction(['connections'], 'readwrite');
                const store = transaction.objectStore('connections');
                await new Promise((resolve, reject) => {
                    const request = store.put({
                        id: connId,
                        ...connectionDetails,
                        status: 'connecting'
                    });
                    request.onsuccess = () => resolve();
                    request.onerror = () => reject(request.error);
                });
            }
        } catch (error) {
            console.error('Failed to connect to chat:', error);
            this.showToast?.('Failed to connect to chat', 'error');
        } finally {
            this.isConnecting = false;
        }
    }

    async loadSavedConnections() {
        if (!this.isDBReady) return;

        const transaction = db.transaction(['connections'], 'readonly');
        const store = transaction.objectStore('connections');
        
        return new Promise((resolve, reject) => {
            const request = store.getAll();
            
            request.onsuccess = async () => {
                const connections = request.result;
                console.log('Loading saved connections:', connections);
                
                // Initialize all loaded connections as disconnected
                this.savedConnections = connections.map(conn => ({
                    ...conn,
                    status: 'disconnected'
                }));
                this.onConnectionsChange?.(this.savedConnections);
                
                // Try to connect to each saved connection
                for (const conn of connections) {
                    await this.connectNewChat(conn);
                }
                resolve();
            };
            
            request.onerror = () => reject(request.error);
        });
    }

    async disconnectChat(connId) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

        try {
            // Send unsubscribe request
            const unsubMessage = {
                type: 'unsubscribe',
                liveId: connId
            };
            console.log('Sending unsubscribe message:', unsubMessage);
            this.ws.send(JSON.stringify(unsubMessage));

            // Remove from savedConnections
            this.savedConnections = this.savedConnections.filter(c => c.id !== connId);
            this.onConnectionsChange?.(this.savedConnections);

            // Remove from database
            if (this.isDBReady) {
                const transaction = db.transaction(['connections'], 'readwrite');
                const store = transaction.objectStore('connections');
                await new Promise((resolve, reject) => {
                    const request = store.delete(connId);
                    request.onsuccess = () => resolve();
                    request.onerror = () => reject(request.error);
                });
            }
        } catch (error) {
            console.error('Failed to disconnect chat:', error);
            this.showToast?.('Failed to disconnect chat', 'error');
        }
    }

    async refreshChat(connId) {
        const connection = this.savedConnections.find(c => c.id === connId);
        if (!connection) return;

        try {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                console.log('WebSocket not connected, attempting to connect...');
                await this.initWebSocket();
            }

            // Update connection status to connecting
            connection.status = 'connecting';
            this.onConnectionsChange?.(this.savedConnections);

            // Send subscription request
            this.ws.send(JSON.stringify({
                type: 'subscribe',
                platform: connection.platform,
                identifier: connection.identifier,
                identifierType: connection.identifierType
            }));

            this.showToast?.(`Refreshing ${connection.platform} chat: ${connection.identifier}...`, 'info');
        } catch (error) {
            console.error('Failed to refresh chat:', error);
            this.showToast?.('Failed to refresh chat', 'error');
            connection.status = 'error';
            this.onConnectionsChange?.(this.savedConnections);
        }
    }
} 