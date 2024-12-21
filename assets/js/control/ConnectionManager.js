const dbName = 'OrionConnectionsDB';
const dbVersion = 2;
let db;

export class ConnectionManager {
    constructor() {
        this.activeConnections = [];
        this.isDBReady = false;
        this.onConnectionsChange = null;
        this.onMessageReceived = null;
        this.showToast = null;
    }

    async init() {
        try {
            await this.initDB();
            this.isDBReady = true;
            await this.loadSavedConnections();
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

    generateConnectionId(platform, identifier, identifierType = 'username') {
        return `${platform}-${identifierType}-${identifier}`.toLowerCase();
    }

    async connectNewChat(connectionDetails, existingId = null) {
        const generatedId = this.generateConnectionId(
            connectionDetails.platform,
            connectionDetails.identifier,
            connectionDetails.identifierType
        );
        const connId = existingId || generatedId;

        if (!existingId && this.activeConnections.some(conn => conn.id === connId)) {
            this.showToast?.('This connection already exists', 'error');
            return;
        }

        const ws = new WebSocket('wss://chatsocket.oristarium.com/ws');
        let subscriptionConfirmed = false;

        ws.onopen = () => {
            const subscribeMessage = {
                type: 'subscribe',
                identifier: connectionDetails.identifier,
                platform: connectionDetails.platform,
                ...(connectionDetails.platform === 'youtube' && { 
                    identifierType: connectionDetails.identifierType 
                })
            };

            console.log('WebSocket connected, sending subscribe message:', {
                connectionId: connId,
                message: subscribeMessage
            });

            ws.send(JSON.stringify(subscribeMessage));

            setTimeout(() => {
                if (!subscriptionConfirmed) {
                    console.error('Subscription confirmation timeout for:', {
                        connectionId: connId,
                        platform: connectionDetails.platform,
                        identifier: connectionDetails.identifier,
                        identifierType: connectionDetails.identifierType
                    });
                    this.showToast?.('Connection failed: No response from server', 'error');
                    ws.close();
                }
            }, 5000);
        };

        ws.onmessage = async (event) => {
            const data = JSON.parse(event.data);
            
            if (data.type !== 'chat') {
                console.log('WebSocket received message:', {
                    connectionId: connId,
                    type: data.type,
                    data: data
                });
            }
            
            if (data.type === 'status' && data.status === 'subscribed') {
                subscriptionConfirmed = true;
                console.log('Subscription confirmed for:', {
                    connectionId: connId,
                    platform: connectionDetails.platform,
                    identifier: connectionDetails.identifier,
                    identifierType: connectionDetails.identifierType,
                    response: data
                });

                const connection = {
                    id: connId,
                    ws,
                    platform: connectionDetails.platform,
                    identifier: connectionDetails.identifier,
                    identifierType: connectionDetails.identifierType
                };

                this.activeConnections.push(connection);
                this.onConnectionsChange?.(this.activeConnections);

                try {
                    await this.saveConnectionsToDb();
                } catch (error) {
                    console.error('Error saving connections:', error);
                }

                if (!existingId) {
                    this.showToast?.('New connection added');
                }
                return;
            }

            if (data.type === 'chat' && subscriptionConfirmed) {
                const messageWithConn = {
                    ...data,
                    connectionId: connId
                };
                this.onMessageReceived?.(messageWithConn);
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error for connection:', {
                connectionId: connId,
                error: error
            });
            this.showToast?.('Connection error', 'error');
            this.disconnectChat(connId);
        };

        ws.onclose = () => {
            console.log('WebSocket closed for connection:', {
                connectionId: connId,
                wasConfirmed: subscriptionConfirmed
            });
            if (subscriptionConfirmed) {
                this.disconnectChat(connId);
            }
        };
    }

    async refreshChat(connId) {
        const conn = this.activeConnections.find(c => c.id === connId);
        if (conn) {
            const details = {
                platform: conn.platform,
                identifier: conn.identifier,
                identifierType: conn.identifierType
            };
            
            await this.disconnectChat(connId);
            await this.connectNewChat(details, connId);
        }
    }

    async disconnectChat(connId) {
        const connIndex = this.activeConnections.findIndex(conn => conn.id === connId);
        if (connIndex !== -1) {
            const conn = this.activeConnections[connIndex];
            try {
                conn.ws.send(JSON.stringify({ type: 'unsubscribe' }));
            } catch (e) {
                console.warn('Could not send unsubscribe message:', e);
            }
            conn.ws.close();
            this.activeConnections.splice(connIndex, 1);
            this.onConnectionsChange?.(this.activeConnections);

            if (this.isDBReady) {
                await this.saveConnectionsToDb();
            }

            this.showToast?.('Connection removed');
        }
    }

    async saveConnectionsToDb() {
        if (!db) return;
        
        const transaction = db.transaction(['connections'], 'readwrite');
        const store = transaction.objectStore('connections');

        await store.clear();

        for (const conn of this.activeConnections) {
            const connData = {
                id: conn.id,
                platform: conn.platform,
                identifier: conn.identifier,
                identifierType: conn.identifierType
            };
            await store.add(connData);
        }
    }

    async loadSavedConnections() {
        if (!db) return;
        
        return new Promise((resolve, reject) => {
            const transaction = db.transaction(['connections'], 'readonly');
            const store = transaction.objectStore('connections');
            
            const request = store.getAll();
            
            request.onsuccess = async () => {
                const connections = request.result;
                if (connections.length > 0) {
                    for (const conn of connections) {
                        await this.connectNewChat({
                            platform: conn.platform,
                            identifier: conn.identifier,
                            identifierType: conn.identifierType
                        }, conn.id);
                    }
                }
                resolve(connections);
            };
            
            request.onerror = () => {
                reject(request.error);
            };
        });
    }
} 