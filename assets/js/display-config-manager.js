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
                value: { justify: 'end', align: 'end' },
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

    setupToggleButton() {
        const $configToggle = $('#configToggle');
        const $configPanel = $('#configPanel');
        
        $configToggle.on('click', () => {
            $(this).toggleClass('rotate-180');
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

    loadConfig() {
        const savedConfig = localStorage.getItem('displayConfig');
        if (savedConfig) {
            const config = JSON.parse(savedConfig);
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
        }
    }

    saveConfig() {
        const config = {};
        Object.entries(this.configInputs).forEach(([key, control]) => {
            if (key === 'containerPosition') {
                config[key] = control.value;
            } else if (control.input) {
                config[key] = control.input.val();
            }
        });
        localStorage.setItem('displayConfig', JSON.stringify(config));
    }
}

// Initialize when document is ready
$(document).ready(() => {
    const displayConfig = new DisplayConfigManager();
    
    // Load saved configuration
    displayConfig.loadConfig();
    
    // Setup input event listeners
    Object.entries(displayConfig.configInputs).forEach(([key, config]) => {
        if (config.input && config.value) {
            config.input.on('input', function() {
                const value = $(this).val();
                config.value.text(value);
                config.apply(value);
                displayConfig.saveConfig();
            });
        }
    });
}); 