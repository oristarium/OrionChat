class DisplayConfigManager {
    constructor() {
        // Create a style element for dynamic styles
        this.styleElement = document.createElement('style');
        document.head.appendChild(this.styleElement);

        this.configInputs = {
            messageFontSize: {
                input: $('#messageFontSize'),
                value: $('#messageFontSizeValue'),
                apply: (value) => {
                    this.updateStyle('.chat-message__text', `font-size: ${value}px`);
                }
            },
            authorFontSize: {
                input: $('#authorFontSize'),
                value: $('#authorFontSizeValue'),
                apply: (value) => {
                    this.updateStyle('.chat-message__author_name', `font-size: ${value}px`);
                }
            },
            avatarSize: {
                input: $('#avatarSize'),
                value: $('#avatarSizeValue'),
                apply: (value) => {
                    this.updateStyle(':root', `--avatar-size: ${value}px`);
                    this.updateStyle('.chat-message__avatar', `
                        width: var(--avatar-size);
                        height: var(--avatar-size);
                        min-width: var(--avatar-size);
                        min-height: var(--avatar-size);
                        flex-shrink: 0
                    `);
                }
            },
            messageHPadding: {
                input: $('#messageHPadding'),
                value: $('#messageHPaddingValue'),
                apply: (value) => {
                    this.updateStyle('.chat-message', `padding-left: ${value}em; padding-right: ${value}em`);
                }
            },
            messageVPadding: {
                input: $('#messageVPadding'),
                value: $('#messageVPaddingValue'),
                apply: (value) => {
                    this.updateStyle('.chat-message', `padding-top: ${value}em; padding-bottom: ${value}em`);
                }
            },
            containerPosition: {
                value: { justify: 'center', align: 'end' },
                apply: (value) => {
                    this.updateStyle('#viewport-container', 
                        `justify-content: ${value.justify}; align-items: ${value.align}`
                    );
                }
            },
            authorMarginBottom: {
                input: $('#authorMarginBottom'),
                value: $('#authorMarginBottomValue'),
                apply: (value) => {
                    this.updateStyle('.chat-message__content_header', `margin-bottom: ${value}em`);
                }
            }
        };
        
        // Apply default styles immediately
        this.applyDefaultStyles();
        
        // Setup toggle button functionality
        this.setupToggleButton();
        this.setupPositionButtons();
    }

    // Add new method to manage dynamic styles
    updateStyle(selector, rules) {
        // Get all current styles
        const styleRules = Array.from(this.styleElement.sheet?.cssRules || []);
        const existingRuleIndex = styleRules.findIndex(rule => 
            rule.selectorText === selector
        );

        // Get existing styles for this selector
        let existingStyles = '';
        if (existingRuleIndex >= 0) {
            existingStyles = styleRules[existingRuleIndex].style.cssText;
            this.styleElement.sheet.deleteRule(existingRuleIndex);
        }

        // Merge new rules with existing ones
        const newRules = rules.split(';').filter(rule => rule.trim());
        const existingRules = existingStyles.split(';').filter(rule => rule.trim());
        
        // Combine rules, new rules overwriting existing ones
        const combinedRules = [...existingRules, ...newRules]
            .map(rule => rule.trim())
            .filter(rule => rule)
            .reduce((acc, rule) => {
                const [property] = rule.split(':');
                acc[property.trim()] = rule;
                return acc;
            }, {});

        // Create final rule
        const finalRule = `${selector} { ${Object.values(combinedRules).join('; ')} }`;
        
        // Add the new rule
        if (this.styleElement.sheet) {
            this.styleElement.sheet.insertRule(finalRule, this.styleElement.sheet.cssRules.length);
        }
    }

    // Add new method to apply default styles
    applyDefaultStyles() {
        // Apply all default values from the config inputs
        Object.values(this.configInputs).forEach(config => {
            if (config.apply) {
                if (config.input) {
                    // Use the default value from the HTML input
                    config.apply(config.input.val());
                } else {
                    // Use the default value from the config object
                    config.apply(config.value);
                }
            }
        });

        // Add any additional base styles needed
        this.updateStyle('.chat-message', `
            display: flex;
            align-items: center;
            gap: 32px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            animation: fadeInUp 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            transform-origin: center center;
            will-change: transform, opacity;
            max-width: 800px;
        `);

        this.updateStyle('.chat-message__content', `
            flex-grow: 1;
            min-width: 0;
        `);

        this.updateStyle('.chat-message__text', `
            word-wrap: break-word;
            line-height: 1.4;
        `);
    }

    setupToggleButton() {
        const $configToggle = $('#configToggle');
        const $configPanel = $('#configPanel');
        
        $configToggle.on('click', () => {
            $configToggle.toggleClass('rotate-180');
            $configPanel.toggleClass('hidden');
            
            // Show/hide sample message when config panel is toggled
            const isConfiguring = !$configPanel.hasClass('hidden');
            if (isConfiguring) {
                const mockMessage = document.createElement('div');
                mockMessage.className = 'chat-message chat-message_twitch';
                mockMessage.innerHTML = `
                    <div class="chat-message__avatar" style="
                        background-color: #9146FF;
                        min-width: var(--avatar-size, 64px);
                        min-height: var(--avatar-size, 64px);
                        border-radius: 50%;
                        flex-shrink: 0;
                    "></div>
                    <div class="chat-message__content">
                        <div class="chat-message__content_header">
                            <span class="chat-message__author_name">Sample Username</span>
                            <div class="badges-container">
                                <img class="badge" src="data:image/svg+xml,%3Csvg width='18' height='18' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='9' cy='9' r='8' fill='%239146FF'/%3E%3C/svg%3E" alt="Sample Badge">
                            </div>
                        </div>
                        <div class="chat-message__text">Sample message text here. Adjust the size using the controls!</div>
                    </div>
                `;
                
                // Clear any existing message
                $('#viewport-container').empty().append(mockMessage);
                messageContainer = mockMessage;
                
            } else {
                // Clear the sample message when closing config
                $('#viewport-container').empty();
                messageContainer = null;
            }
            
            // Apply current configuration to the sample message
            if (isConfiguring) {
                Object.values(this.configInputs).forEach(config => {
                    if (config.apply) {
                        if (config.input) {
                            config.apply(config.input.val());
                        } else {
                            config.apply(config.value);
                        }
                    }
                });
            }
        });
    }

    setupPositionButtons() {
        $('.position-btn').on('click', (e) => {
            $('.position-btn').removeClass('bg-white/20');
            $(e.currentTarget).addClass('bg-white/20');

            const justify = $(e.currentTarget).data('justify');
            const align = $(e.currentTarget).data('align');
            
            this.configInputs.containerPosition.value = { justify, align };
            this.configInputs.containerPosition.apply({ justify, align });
            this.saveConfig();
        });
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/kv/displayConfig');
            if (!response.ok) {
                if (response.status === 404) {
                    console.log('No saved config found, using defaults');
                    return; // This is fine, we'll use default values
                }
                console.error('Failed to load display config:', response.statusText);
                return;
            }
            
            const config = await response.json();
            Object.entries(config).forEach(([key, value]) => {
                if (this.configInputs[key]) {
                    if (key === 'containerPosition') {
                        this.configInputs[key].value = value;
                        this.configInputs[key].apply(value);
                        $(`.position-btn[data-justify="${value.justify}"][data-align="${value.align}"]`)
                            .addClass('bg-white/20');
                    } else if (this.configInputs[key].input) {
                        this.configInputs[key].input.val(value);
                        this.configInputs[key].value.text(value);
                        this.configInputs[key].apply(value);
                    }
                }
            });
        } catch (error) {
            console.error('Error loading display config:', error);
        }
    }

    async saveConfig() {
        const config = {};
        Object.entries(this.configInputs).forEach(([key, control]) => {
            if (key === 'containerPosition') {
                config[key] = control.value;
            } else if (control.input) {
                config[key] = control.input.val();
            }
        });

        try {
            const response = await fetch('/api/kv/displayConfig', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (!response.ok) {
                throw new Error(`Failed to save display config: ${response.statusText}`);
            }
        } catch (error) {
            console.error('Error saving display config:', error);
        }
    }
}

// Remove the jQuery initialization block and just export the class
window.DisplayConfigManager = DisplayConfigManager; 