// Main control.js file
import { ChatManager } from './modules/ChatManager.js';
import { ConnectionManager } from './modules/ConnectionManager.js';
import { UIManager } from './modules/UIManager.js';
import { MessageHandler } from './modules/MessageHandler.js';

class Controller {
    constructor() {
        this.ui = new UIManager();
        this.chat = new ChatManager();
        this.connection = new ConnectionManager();
        this.messageHandler = new MessageHandler();
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.ui.initializePlatformUI();
    }

    setupEventListeners() {
        // Connection events
        document.getElementById('connect-btn').addEventListener('click', () => this.connection.connect());
        document.getElementById('disconnect-btn').addEventListener('click', () => this.connection.disconnect());

        // Platform selection events
        document.getElementById('platform-type').addEventListener('change', (e) => {
            this.ui.updatePlatformUI(e.target.value);
        });

        // Make necessary methods available globally
        window.sendToDisplay = this.messageHandler.sendToDisplay.bind(this.messageHandler);
        window.sendToTTS = this.messageHandler.sendToTTS.bind(this.messageHandler);
    }
}

// Initialize the controller when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new Controller();
}); 