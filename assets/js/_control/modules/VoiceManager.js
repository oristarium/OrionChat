export class VoiceManager {
    constructor(selectors) {
        this.voiceData = null;
        this.selectedVoices = new Set();
        
        // Store jQuery elements using passed selectors
        this.$providerSelect = $(selectors.providerSelect);
        this.$googleSelect = $(selectors.googleSelect);
        this.$tiktokSelects = $(selectors.tiktokSelects);
        this.$voiceSearch = $(selectors.voiceSearch);
        this.$languageFilter = $(selectors.languageFilter);
        this.$genderFilter = $(selectors.genderFilter);
        this.$voiceList = $(selectors.voiceList);
        this.$selectAllCheckbox = $(selectors.selectAllCheckbox);
        this.$selectedVoiceCount = $(selectors.selectedVoiceCount);
        this.$clearVoicesBtn = $(selectors.clearVoicesBtn);
        
        this.init();
    }

    async init() {
        await this.loadVoiceData();
        this.setupEventListeners();
        this.updateVoiceUI(this.$providerSelect.val());
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
                // console.log('Parsed voice entry:', result);
                return result;
            });
        return parsed;
    }

    populateLanguageFilter() {
        // Get unique languages
        const languages = [...new Set(this.voiceData.map(v => v.lang_label))];
        
        this.$languageFilter.html('<option value="all">All Languages</option>' + languages
            .map(lang => `<option value="${lang}">${lang}</option>`)
            .join(''));
    }

    populateVoiceList() {
        this.$voiceList.html(this.voiceData
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
            `).join(''));

        // Delegate the click event to the table body
        $(this.$voiceList).off('click', '.voice-row td').on('click', '.voice-row td', (e) => {
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
        this.$providerSelect.on('change', (e) => {
            this.updateVoiceUI(e.target.value);
        });

        this.$voiceSearch.on('input', () => this.filterVoices());
        this.$languageFilter.on('change', () => this.filterVoices());
        this.$genderFilter.on('change', () => this.filterVoices());
        
        this.$selectAllCheckbox.on('change', (e) => {
            const checkboxes = this.$voiceList.find('input[type="checkbox"]');
            checkboxes.each((index, checkbox) => {
                checkbox.checked = e.target.checked;
                this.toggleVoice(checkbox.value, e.target.checked);
            });
            this.updateVoiceCount();
        });

        if (this.$clearVoicesBtn) {
            this.$clearVoicesBtn.on('click', () => {
                this.selectedVoices.clear();
                this.$selectAllCheckbox.prop('checked', false);
                const checkboxes = this.$voiceList.find('input[type="checkbox"]');
                checkboxes.each((index, checkbox) => {
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
        if (this.$selectedVoiceCount) {
            this.$selectedVoiceCount.text(this.selectedVoices.size);
        }
    }

    filterVoices() {
        const searchTerm = this.$voiceSearch.val().toLowerCase();
        const languageFilter = this.$languageFilter.val();
        const genderFilter = this.$genderFilter.val();
        
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
        this.$voiceList.html(data
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
            `).join(''));

        // Delegate the click event to the table body
        $(this.$voiceList).off('click', '.voice-row td').on('click', '.voice-row td', (e) => {
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
            this.$googleSelect.css('display', 'none');
            this.$tiktokSelects.css('display', 'block');
        } else {
            this.$googleSelect.css('display', 'block');
            this.$tiktokSelects.css('display', 'none');
        }
    }

    getCurrentVoiceId() {
        const provider = this.$providerSelect.val();
        if (provider === 'tiktok') {
            if (this.selectedVoices.size === 0) {
                return this.voiceData[0].voice_id; // Default to first voice if none selected
            }
            const voiceIds = Array.from(this.selectedVoices);
            return voiceIds[Math.floor(Math.random() * voiceIds.length)];
        }
        return $('#tts-language').val();
    }

    setSelectedVoices(voices) {
        this.selectedVoices.clear();
        voices.forEach(voice => {
            if (voice.provider === 'tiktok') {
                this.selectedVoices.add(voice.voice_id);
            }
        });
        
        // Update UI
        this.updateVoiceUI(this.$providerSelect.val());
        this.updateVoiceCount();
        
        // Update checkboxes
        this.$voiceList.find('input[type="checkbox"]').each((_, checkbox) => {
            checkbox.checked = this.selectedVoices.has(checkbox.value);
        });
    }

    getSelectedVoices() {
        const voices = [];
        
        // Add TikTok voices
        this.selectedVoices.forEach(voiceId => {
            voices.push({
                voice_id: voiceId,
                provider: 'tiktok'
            });
        });
        
        // Add Google voice if selected
        if (this.$providerSelect.val() === 'google') {
            voices.push({
                voice_id: $('#tts-language').val(),
                provider: 'google'
            });
        }
        
        return voices;
    }
} 