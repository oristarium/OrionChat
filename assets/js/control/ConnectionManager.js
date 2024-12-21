const dbName = 'OrionConnectionsDB';
const dbVersion = 2;
let db;

export class ConnectionManager {
    constructor() {
        this.ws = null;
        this.activeConnections = [];
        this.isDBReady = false;
        this.onConnectionsChange = null;
        this.onMessageReceived = null;
        this.showToast = null;
        this.chatManager = null;
        this.isConnecting = false;
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

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//chatsocket.oristarium.com/ws`;
            
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
                this.ws = null;
                setTimeout(() => {
                    console.log('Attempting WebSocket reconnection...');
                    this.initWebSocket()
                        .then(() => {
                            console.log('WebSocket reconnected, reloading saved connections...');
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
                    console.log('WebSocket received:', data);
                    
                    switch (data.type) {
                        case 'chat':
                            this.onMessageReceived?.(data);
                            break;
                            
                        case 'status':
                            this.handleStatusMessage(data);
                            break;
                            
                        case 'error':
                            console.error('WebSocket error message:', data);
                            this.showToast?.(data.error, 'error');
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
            const connection = this.activeConnections.find(c => c.id === data.liveId);
            if (!connection) {
                // Add new connection
                const [platform, identifierType, identifier] = data.liveId.split('-');
                const newConnection = {
                    id: data.liveId,
                    platform,
                    identifier,
                    identifierType: identifierType || 'username'
                };
                this.activeConnections.push(newConnection);
                this.onConnectionsChange?.(this.activeConnections);
                this.showToast?.(`Successfully connected to ${platform} chat: ${identifier}`, 'success');
            }
        } else if (data.status === 'unsubscribed') {
            // Remove connection
            const connection = this.activeConnections.find(c => c.id === data.liveId);
            if (connection) {
                this.showToast?.(`Disconnected from ${connection.platform} chat: ${connection.identifier}`, 'info');
            }
            this.activeConnections = this.activeConnections.filter(c => c.id !== data.liveId);
            this.onConnectionsChange?.(this.activeConnections);
        }
    }

    generateConnectionId(platform, identifier, identifierType = 'username') {
        return `${platform}-${identifierType}-${identifier}`.toLowerCase();
    }

    async connectNewChat(connectionDetails, existingId = null) {
        if (this.isConnecting) return;
        this.isConnecting = true;

        try {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                console.log('WebSocket not connected, attempting to connect...');
                await this.initWebSocket();
            }

            const connId = existingId || this.generateConnectionId(
                connectionDetails.platform,
                connectionDetails.identifier,
                connectionDetails.identifierType
            );

            if (this.activeConnections.some(conn => conn.id === connId)) {
                console.log('Already connected to:', connId);
                this.showToast?.('Already connected to this chat', 'info');
                this.isConnecting = false;
                return;
            }

            console.log('Subscribing to chat:', {
                platform: connectionDetails.platform,
                identifier: connectionDetails.identifier,
                identifierType: connectionDetails.identifierType
            });

            this.showToast?.(`Connecting to ${connectionDetails.platform} chat: ${connectionDetails.identifier}...`, 'info');

            this.ws.send(JSON.stringify({
                type: 'subscribe',
                platform: connectionDetails.platform,
                identifier: connectionDetails.identifier,
                identifierType: connectionDetails.identifierType
            }));

            if (this.isDBReady && !existingId) {
                const transaction = db.transaction(['connections'], 'readwrite');
                const store = transaction.objectStore('connections');
                await new Promise((resolve, reject) => {
                    const request = store.put({
                        id: connId,
                        ...connectionDetails
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
                
                for (const conn of connections) {
                    await this.connectNewChat(conn, conn.id);
                }
                resolve();
            };
            
            request.onerror = () => reject(request.error);
        });
    }

    async disconnectChat(connId) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

        try {
            // Unsubscribe from the chat
            this.ws.send(JSON.stringify({
                type: 'unsubscribe',
                liveId: connId
            }));

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

            this.showToast?.('Connection removed');
        } catch (error) {
            console.error('Failed to disconnect chat:', error);
            this.showToast?.('Failed to disconnect chat', 'error');
        }
    }

    async refreshChat(connId) {
        const connection = this.activeConnections.find(c => c.id === connId);
        if (!connection) return;

        await this.disconnectChat(connId);
        await this.connectNewChat(connection, connId);
    }
} 