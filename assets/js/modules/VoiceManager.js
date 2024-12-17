export class VoiceManager {
    constructor() {
        this.voiceData = null;
        this.selectedVoices = new Set();
        this.providerSelect = document.getElementById('tts-provider');
        this.googleSelect = document.getElementById('google-voice-select');
        this.tiktokSelects = document.getElementById('tiktok-voice-selects');
        this.voiceSearch = document.getElementById('voice-search');
        this.languageFilter = document.getElementById('language-filter');
        this.genderFilter = document.getElementById('gender-filter');
        this.voiceList = document.getElementById('voice-list');
        this.selectAllCheckbox = document.getElementById('select-all-voices');
        this.selectedVoiceCount = document.getElementById('selected-voice-count');
        this.clearVoicesBtn = document.getElementById('clear-voices');
        
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
            this.populateLanguageFilter();
            this.populateVoiceList();
        } catch (error) {
            console.error('Failed to load voice data:', error);
        }
    }

    parseCSV(text) {
        const lines = text.split('\n');
        const headers = lines[0].split(',');
        
        const parsed = lines.slice(1)
            .filter(line => line.trim())
            .map(line => {
                // Handle CSV values that might contain commas within quotes
                const values = line.match(/(?:^|,)("(?:[^"]|"")*"|[^,]*)/g)
                    .map(value => {
                        // Remove leading comma and quotes, handle escaped quotes
                        value = value.replace(/^,/, '');
                        if (value.startsWith('"') && value.endsWith('"')) {
                            value = value.slice(1, -1).replace(/""/g, '"');
                        }
                        return value.trim();
                    });

                const result = {
                    lang_label: values[0],
                    voice_name: values[1],
                    voice_gender: values[2],
                    voice_id: values[3],
                    sample_sentence: values[4] || '',
                    sample_voice: values[5] ? values[5].replace(/\\n/g, '\n').replace(/\\"/g, '"') : ''
                };
                console.log('Parsed voice entry:', result);
                return result;
            });
        return parsed;
    }

    populateLanguageFilter() {
        // Get unique languages
        const languages = [...new Set(this.voiceData.map(v => v.lang_label))];
        
        this.languageFilter.innerHTML = '<option value="all">All Languages</option>' + languages
            .map(lang => `<option value="${lang}">${lang}</option>`)
            .join('');
    }

    populateVoiceList() {
        this.voiceList.innerHTML = this.voiceData
            .map(voice => `
                <tr class="voice-row" data-voice-id="${voice.voice_id}">
                    <td>
                        <input type="checkbox" 
                               value="${voice.voice_id}"
                               ${this.selectedVoices.has(voice.voice_id) ? 'checked' : ''}
                               onchange="window.voiceManager.toggleVoice('${voice.voice_id}', this.checked)">
                    </td>
                    <td>${voice.lang_label}</td>
                    <td>${voice.voice_name}</td>
                    <td>${voice.voice_gender}</td>
                    <td class="sample-cell">
                        <audio controls preload="none" class="voice-sample">
                            <source src="${voice.sample_voice}" type="audio/mpeg">
                        </audio>
                    </td>
                </tr>
            `).join('');

        // Delegate the click event to the table body
        $(this.voiceList).off('click', '.voice-row td').on('click', '.voice-row td', (e) => {
            // Skip if clicking the sample cell or checkbox
            if ($(e.target).closest('.sample-cell').length || $(e.target).is(':checkbox')) {
                return;
            }
            
            const $row = $(e.target).closest('.voice-row');
            const $checkbox = $row.find(':checkbox');
            const newState = !$checkbox.prop('checked');
            
            $checkbox.prop('checked', newState);
            this.toggleVoice($row.data('voice-id'), newState);
        });
    }

    setupEventListeners() {
        this.providerSelect.addEventListener('change', (e) => {
            this.updateVoiceUI(e.target.value);
        });

        this.voiceSearch.addEventListener('input', () => this.filterVoices());
        this.languageFilter.addEventListener('change', () => this.filterVoices());
        this.genderFilter.addEventListener('change', () => this.filterVoices());
        
        this.selectAllCheckbox.addEventListener('change', (e) => {
            const checkboxes = this.voiceList.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach(checkbox => {
                checkbox.checked = e.target.checked;
                this.toggleVoice(checkbox.value, e.target.checked);
            });
            this.updateVoiceCount();
        });

        if (this.clearVoicesBtn) {
            this.clearVoicesBtn.addEventListener('click', () => {
                this.selectedVoices.clear();
                this.selectAllCheckbox.checked = false;
                const checkboxes = this.voiceList.querySelectorAll('input[type="checkbox"]');
                checkboxes.forEach(checkbox => {
                    checkbox.checked = false;
                });
                this.updateVoiceCount();
            });
        }
    }

    toggleVoice(voiceId, selected) {
        if (selected) {
            this.selectedVoices.add(voiceId);
        } else {
            this.selectedVoices.delete(voiceId);
        }
        this.updateVoiceCount();
    }

    updateVoiceCount() {
        if (this.selectedVoiceCount) {
            this.selectedVoiceCount.textContent = this.selectedVoices.size;
        }
    }

    filterVoices() {
        const searchTerm = this.voiceSearch.value.toLowerCase();
        const languageFilter = this.languageFilter.value;
        const genderFilter = this.genderFilter.value;
        
        // Filter the data
        const filteredData = this.voiceData.filter(voice => {
            const matchesSearch = 
                voice.voice_name.toLowerCase().includes(searchTerm) ||
                voice.lang_label.toLowerCase().includes(searchTerm);
            const matchesLanguage = 
                languageFilter === 'all' || 
                voice.lang_label === languageFilter;
            const matchesGender =
                genderFilter === 'all' ||
                voice.voice_gender.toLowerCase() === genderFilter;
            return matchesSearch && matchesLanguage && matchesGender;
        });
        
        // Update the voiceData reference and repopulate
        this.populateVoiceList(filteredData);
    }

    populateVoiceList(data = this.voiceData) {
        this.voiceList.innerHTML = data
            .map(voice => `
                <tr class="voice-row" data-voice-id="${voice.voice_id}">
                    <td>
                        <input type="checkbox" 
                               value="${voice.voice_id}"
                               ${this.selectedVoices.has(voice.voice_id) ? 'checked' : ''}
                               onchange="window.voiceManager.toggleVoice('${voice.voice_id}', this.checked)">
                    </td>
                    <td>${voice.lang_label}</td>
                    <td>${voice.voice_name}</td>
                    <td>${voice.voice_gender}</td>
                    <td class="sample-cell">
                        <audio controls preload="none" class="voice-sample">
                            <source src="${voice.sample_voice}" type="audio/mpeg">
                        </audio>
                    </td>
                </tr>
            `).join('');

        // Delegate the click event to the table body
        $(this.voiceList).off('click', '.voice-row td').on('click', '.voice-row td', (e) => {
            // Skip if clicking the sample cell or checkbox
            if ($(e.target).closest('.sample-cell').length || $(e.target).is(':checkbox')) {
                return;
            }
            
            const $row = $(e.target).closest('.voice-row');
            const $checkbox = $row.find(':checkbox');
            const newState = !$checkbox.prop('checked');
            
            $checkbox.prop('checked', newState);
            this.toggleVoice($row.data('voice-id'), newState);
        });
    }

    async playSample(samplePath) {
        console.log('Attempting to play sample:', {
            rawPath: samplePath,
            type: typeof samplePath
        });
        const audio = new Audio(samplePath);
        try {
            await audio.play();
        } catch (error) {
            console.error('Error playing sample:', error);
        }
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
            if (this.selectedVoices.size === 0) {
                return this.voiceData[0].voice_id; // Default to first voice if none selected
            }
            const voiceIds = Array.from(this.selectedVoices);
            return voiceIds[Math.floor(Math.random() * voiceIds.length)];
        }
        return document.getElementById('tts-language').value;
    }
} 