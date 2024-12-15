// Main control.js file
import { ChatManager } from './modules/ChatManager.js';
import { ConnectionManager } from './modules/ConnectionManager.js';
import { UIManager } from './modules/UIManager.js';
import { MessageHandler } from './modules/MessageHandler.js';
import { AvatarManager } from './modules/AvatarManager.js';

class Controller {
    constructor() {
        console.log('Initializing Controller');
        this.ui = new UIManager();
        this.chat = new ChatManager();
        this.connection = new ConnectionManager();
        this.messageHandler = new MessageHandler();
        this.avatarManager = new AvatarManager();
        
        this.init();
    }

    init() {
        console.log('Controller initialization started');
        this.setupEventListeners();
        this.ui.initializePlatformUI();
        console.log('Controller initialization completed');
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