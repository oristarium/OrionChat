// Main control.js file
import { ChatManager } from './modules/ChatManager.js';
import { ConnectionManager } from './modules/ConnectionManager.js';
import { UIManager } from './modules/UIManager.js';
import { MessageHandler } from './modules/MessageHandler.js';
import { AvatarManager } from './modules/AvatarManager.js';
import { VoiceManager } from './modules/VoiceManager.js';

class Controller {
    constructor() {
        console.log('Initializing Controller');
        this.ui = new UIManager();
        this.chat = new ChatManager();
        this.connection = new ConnectionManager();
        this.messageHandler = new MessageHandler();
        this.avatarManager = new AvatarManager();
        this.voiceManager = new VoiceManager();
        window.voiceManager = this.voiceManager;
        
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

        // Use jQuery event delegation and chaining
        $('#connect-btn').on('click', () => this.connection.connect());
        $('#disconnect-btn').on('click', () => this.connection.disconnect());
        $('#platform-type').on('change', (e) => this.ui.updatePlatformUI(e.target.value));

        // Clear display and TTS queue buttons
        $('#clear-display').on('click', () => {
            console.log('Clear display button clicked - Controller handling');
            this.messageHandler.clearDisplay().catch(error => {
                console.error('Error in clearDisplay:', error);
            });
        });

        $('#clear-tts').on('click', () => {
            console.log('Clear TTS queue button clicked');
            this.messageHandler.clearTTSQueue().catch(error => {
                console.error('Error in clearTTSQueue:', error);
            });
        });

        // Make necessary methods available globally
        window.sendToDisplay = this.messageHandler.sendToDisplay.bind(this.messageHandler);
        window.sendToTTS = this.messageHandler.sendToTTS.bind(this.messageHandler);
        
        console.log('Event listeners setup completed');
    }
}

// Initialize the controller when DOM is loaded
$(document).ready(() => {
    console.log('DOM loaded - initializing Controller');
    new Controller();

    // Initialize TTS provider select
    $('#tts-provider').on('change', () => {
        console.log('TTS provider changed:', $('#tts-provider').val());
    });

    // Initialize tabs
    $("#settings-tabs").tabs();
    
    // Remove the avatar settings modal
    $("#open-avatar-settings").remove();
}); 