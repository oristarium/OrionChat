// Import all required modules
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
        
        // Initialize VoiceManager with modal selectors
        this.voiceManager = new VoiceManager({
            providerSelect: '#voice-settings-modal #tts-provider',
            googleSelect: '#voice-settings-modal #google-voice-select',
            tiktokSelects: '#voice-settings-modal #tiktok-voice-selects',
            voiceSearch: '#voice-settings-modal #voice-search',
            languageFilter: '#voice-settings-modal #language-filter',
            genderFilter: '#voice-settings-modal #gender-filter',
            voiceList: '#voice-settings-modal #voice-list',
            selectAllCheckbox: '#voice-settings-modal #select-all-voices',
            selectedVoiceCount: '#voice-settings-modal #selected-voice-count',
            clearVoicesBtn: '#voice-settings-modal #clear-voices'
        });
        
        this.avatarManager = new AvatarManager(this.voiceManager);
        window.voiceManager = this.voiceManager;
        
        // Make chatManager globally available
        window.chatManager = this.chat;
        
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
                $.toast('Error clearing display');
            });
            $.toast('Display cleared');
        });

        $('#clear-tts').on('click', () => {
            console.log('Clear TTS queue button clicked');
            this.messageHandler.clearTTSQueue().catch(error => {
                console.error('Error in clearTTSQueue:', error);
                $.toast('Error clearing TTS queue');
            });
            $.toast('TTS queue cleared');
        });

        // Make necessary methods available globally
        window.sendToDisplay = this.messageHandler.sendToDisplay.bind(this.messageHandler);
        window.sendToTTS = this.messageHandler.sendToTTS.bind(this.messageHandler);
        
        // Add manual control event listeners
        $('#manual-tts').on('click', async () => {
            const message = $('#manual-input').val().trim();
            if (message) {
                try {
                    await this.messageHandler.sendToTTS(message);
                    console.log('Manual TTS message sent successfully');
                    $.toast('Message sent to TTS');
                } catch (error) {
                    console.error('Error sending manual TTS message:', error);
                    $.toast('Error sending message to TTS');
                }
            }
        });

        $('#manual-display').on('click', async () => {
            const message = $('#manual-input').val().trim();
            if (message) {
                try {
                    await this.messageHandler.sendToDisplay(message);
                    console.log('Manual display message sent successfully');
                    $.toast('Message sent to display');
                } catch (error) {
                    console.error('Error sending manual display message:', error);
                    $.toast('Error sending message to display');
                }
            }
        });

        // Add keyboard shortcut (Ctrl/Cmd + Enter) for TTS
        $('#manual-input').on('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.keyCode === 13) {
                $('#manual-tts').click();
                e.preventDefault();
            }
        });

        console.log('Event listeners setup completed');
    }
}

// Initialize the controller when DOM is loaded
$(document).ready(() => {
    console.log('DOM loaded - initializing Controller');
    new Controller();

    // Initialize tabs
    $("#settings-tabs").tabs();
});