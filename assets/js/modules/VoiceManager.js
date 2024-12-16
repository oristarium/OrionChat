export class VoiceManager {
    constructor() {
        this.voiceData = null;
        this.providerSelect = document.getElementById('tts-provider');
        this.googleSelect = document.getElementById('google-voice-select');
        this.tiktokSelects = document.getElementById('tiktok-voice-selects');
        this.tiktokLangSelect = document.getElementById('tiktok-language');
        this.tiktokVoiceSelect = document.getElementById('tiktok-voice');
        
        this.init();
    }

    async init() {
        await this.loadVoiceData();
        this.setupEventListeners();
        this.updateVoiceUI(this.providerSelect.value);
    }

    async loadVoiceData() {
        try {
            const response = await fetch('/data/tiktok_voice_ids.csv');
            const text = await response.text();
            this.voiceData = this.parseCSV(text);
            this.populateLanguageSelect();
        } catch (error) {
            console.error('Failed to load voice data:', error);
        }
    }

    parseCSV(text) {
        const lines = text.split('\n');
        const headers = lines[0].split(',');
        
        return lines.slice(1)
            .filter(line => line.trim())
            .map(line => {
                const values = line.split(',');
                return {
                    lang_label: values[0],
                    voice_name: values[1],
                    voice_id: values[2].trim()
                };
            });
    }

    populateLanguageSelect() {
        // Get unique languages
        const languages = [...new Set(this.voiceData.map(v => v.lang_label))];
        
        this.tiktokLangSelect.innerHTML = languages
            .map(lang => `<option value="${lang}">${lang}</option>`)
            .join('');
            
        // Trigger voice population for initial language
        this.populateVoiceSelect(languages[0]);
    }

    populateVoiceSelect(language) {
        const voices = this.voiceData.filter(v => v.lang_label === language);
        
        this.tiktokVoiceSelect.innerHTML = 
            '<option value="random">Random</option>' +
            voices.map(v => `<option value="${v.voice_id}">${v.voice_name}</option>`)
            .join('');
    }

    setupEventListeners() {
        this.providerSelect.addEventListener('change', (e) => {
            this.updateVoiceUI(e.target.value);
        });

        this.tiktokLangSelect.addEventListener('change', (e) => {
            this.populateVoiceSelect(e.target.value);
        });
    }

    updateVoiceUI(provider) {
        if (provider === 'tiktok') {
            this.googleSelect.style.display = 'none';
            this.tiktokSelects.style.display = 'block';
        } else {
            this.googleSelect.style.display = 'block';
            this.tiktokSelects.style.display = 'none';
        }
    }

    getCurrentVoiceId() {
        const provider = this.providerSelect.value;
        if (provider === 'tiktok') {
            if (this.tiktokVoiceSelect.value === 'random') {
                // Get all voice IDs for current language
                const currentLang = this.tiktokLangSelect.value;
                const voices = this.voiceData.filter(v => v.lang_label === currentLang);
                // Pick a random voice
                const randomIndex = Math.floor(Math.random() * voices.length);
                return voices[randomIndex].voice_id;
            } else {
                return this.tiktokVoiceSelect.value;
            }
        }
        return document.getElementById('tts-language').value;
    }
} 