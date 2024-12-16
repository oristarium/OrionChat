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
        console.log('Setting up event listeners');

        // Connection events
        document.getElementById('connect-btn').addEventListener('click', () => this.connection.connect());
        document.getElementById('disconnect-btn').addEventListener('click', () => this.connection.disconnect());

        // Platform selection events
        document.getElementById('platform-type').addEventListener('change', (e) => {
            this.ui.updatePlatformUI(e.target.value);
        });

        // Add clear display button handler
        const clearBtn = document.getElementById('clear-display');
        if (clearBtn) {
            console.log('Clear display button found and handler attached');
            // Add logging to verify the messageHandler exists
            console.log('MessageHandler instance:', this.messageHandler);
            
            clearBtn.addEventListener('click', () => {
                console.log('Clear display button clicked - Controller handling');
                console.log('MessageHandler before call:', this.messageHandler);
                try {
                    this.messageHandler.clearDisplay().catch(error => {
                        console.error('Error in clearDisplay:', error);
                    });
                } catch (error) {
                    console.error('Error calling clearDisplay:', error);
                    console.error('this.messageHandler:', this.messageHandler);
                }
            });
        } else {
            console.error('Clear display button not found in Controller');
        }

        // Add clear TTS queue button handler
        const clearTTSBtn = document.getElementById('clear-tts');
        if (clearTTSBtn) {
            console.log('Clear TTS queue button found and handler attached');
            clearTTSBtn.addEventListener('click', () => {
                console.log('Clear TTS queue button clicked');
                try {
                    this.messageHandler.clearTTSQueue().catch(error => {
                        console.error('Error in clearTTSQueue:', error);
                    });
                } catch (error) {
                    console.error('Error calling clearTTSQueue:', error);
                }
            });
        } else {
            console.error('Clear TTS queue button not found');
        }

        // Make necessary methods available globally
        window.sendToDisplay = this.messageHandler.sendToDisplay.bind(this.messageHandler);
        window.sendToTTS = this.messageHandler.sendToTTS.bind(this.messageHandler);
        
        console.log('Event listeners setup completed');
    }
}

// Initialize the controller when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded - initializing Controller');
    new Controller();
}); 