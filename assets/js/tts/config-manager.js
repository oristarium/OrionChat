class ConfigManager {
    constructor() {
        this.configInputs = {
            messageFontSize: {
                input: $('#messageFontSize'),
                value: $('#messageFontSizeValue'),
                apply: (value) => {
                    $('.message-bubble').css('fontSize', `${value}px`);
                }
            },
            authorFontSize: {
                input: $('#authorFontSize'),
                value: $('#authorFontSizeValue'),
                apply: (value) => {
                    $('.username').css('fontSize', `${value}px`);
                }
            },
            avatarSize: {
                input: $('#avatarSize'),
                value: $('#avatarSizeValue'),
                apply: (value) => {
                    $('.avatar-container').css('width', `${value}px`);
                }
            },
            avatarGap: {
                input: $('#avatarGap'),
                value: $('#avatarGapValue'),
                apply: (value) => {
                    const justify = this.configInputs.containerPosition.value.justify;
                    $('.avatar-container').css({
                        'margin-left': justify === 'center' ? `${value/2}px` : 
                                     justify === 'end' ? `${value}px` : '0',
                        'margin-right': justify === 'center' ? `${value/2}px` : 
                                      justify === 'start' ? `${value}px` : '0'
                    });
                }
            },
            messageHPadding: {
                input: $('#messageHPadding'),
                value: $('#messageHPaddingValue'),
                apply: (value) => {
                    $('.message-bubble').css({
                        'padding-left': `${value}em`,
                        'padding-right': `${value}em`
                    });
                }
            },
            messageVPadding: {
                input: $('#messageVPadding'),
                value: $('#messageVPaddingValue'),
                apply: (value) => {
                    $('.message-bubble').css({
                        'padding-top': `${value}em`,
                        'padding-bottom': `${value}em`
                    });
                }
            },
            containerPosition: {
                value: { justify: 'end', align: 'end' },
                apply: (value) => {
                    $('#avatars-wrapper').css({
                        'justify-content': value.justify,
                        'align-items': value.align
                    });
                    // Reapply gap with new justification
                    const gapValue = $('#avatarGap').val();
                    this.configInputs.avatarGap.apply(gapValue);
                }
            },
            stackingOrder: {
                value: false,
                apply: (value) => {
                    const $avatars = $('.avatar-container');
                    if (value) {
                        $avatars.removeClass('[&:nth-child(1)]:z-50 [&:nth-child(2)]:z-40 [&:nth-child(3)]:z-30 [&:nth-child(4)]:z-20 [&:nth-child(5)]:z-10')
                               .addClass('[&:nth-child(1)]:z-10 [&:nth-child(2)]:z-20 [&:nth-child(3)]:z-30 [&:nth-child(4)]:z-40 [&:nth-child(5)]:z-50');
                    } else {
                        $avatars.removeClass('[&:nth-child(1)]:z-10 [&:nth-child(2)]:z-20 [&:nth-child(3)]:z-30 [&:nth-child(4)]:z-40 [&:nth-child(5)]:z-50')
                               .addClass('[&:nth-child(1)]:z-50 [&:nth-child(2)]:z-40 [&:nth-child(3)]:z-30 [&:nth-child(4)]:z-20 [&:nth-child(5)]:z-10');
                    }
                }
            }
        };
        
        // Setup toggle button functionality
        this.setupToggleButton();
        
        // Setup event listeners for all inputs
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Setup range input listeners
        Object.entries(this.configInputs).forEach(([key, config]) => {
            if (config.input) {
                config.input.on('input', (e) => {
                    const value = e.target.value;
                    config.value.text(value);
                    config.apply(value);
                    this.saveConfig();
                });
            }
        });

        // Setup position button listeners
        $('.position-btn').on('click', (e) => {
            const $btn = $(e.currentTarget);
            $('.position-btn').removeClass('bg-white/20');
            $btn.addClass('bg-white/20');
            
            const justify = $btn.data('justify');
            const align = $btn.data('align');
            this.configInputs.containerPosition.value = { justify, align };
            this.configInputs.containerPosition.apply({ justify, align });
            this.saveConfig();
        });

        // Setup stacking order toggle
        $('#stackingOrderToggle').on('click', (e) => {
            const $toggle = $(e.currentTarget);
            const currentValue = $toggle.data('reversed');
            const newValue = !currentValue;
            
            $toggle.data('reversed', newValue);
            $toggle.find('span').text(newValue ? 'Last Avatar on Top' : 'First Avatar on Top');
            $toggle.find('svg').toggleClass('rotate-180', newValue);
            
            this.configInputs.stackingOrder.value = newValue;
            this.configInputs.stackingOrder.apply(newValue);
            this.saveConfig();
        });

        // Setup increase/decrease all buttons
        $('#increaseAll').on('click', () => {
            Object.entries(this.configInputs).forEach(([key, config]) => {
                if (config.input) {
                    const input = config.input[0];
                    const step = parseFloat(input.step) || 1;
                    const newValue = Math.min(parseFloat(input.value) + step, input.max);
                    input.value = newValue;
                    config.value.text(newValue);
                    config.apply(newValue);
                }
            });
            this.saveConfig();
        });

        $('#decreaseAll').on('click', () => {
            Object.entries(this.configInputs).forEach(([key, config]) => {
                if (config.input) {
                    const input = config.input[0];
                    const step = parseFloat(input.step) || 1;
                    const newValue = Math.max(parseFloat(input.value) - step, input.min);
                    input.value = newValue;
                    config.value.text(newValue);
                    config.apply(newValue);
                }
            });
            this.saveConfig();
        });
    }

    setupToggleButton() {
        const $configToggle = $('#configToggle');
        const $configPanel = $('#configPanel');
        
        $configToggle.on('click', function() {
            $(this).toggleClass('rotate-180');
            $configPanel.toggleClass('hidden');
            
            // Show sample text when config panel is visible
            const isConfiguring = !$configPanel.hasClass('hidden');
            $('.avatar-container').each(function() {
                const $container = $(this);
                const $messageBubble = $container.find('.message-bubble');
                const $username = $container.find('.username');
                
                if (isConfiguring) {
                    $messageBubble
                        .text("Sample message text here. Adjust the size using the controls!")
                        .addClass('show');
                    $username
                        .text("Sample Username")
                        .addClass('show');
                } else {
                    $messageBubble.removeClass('show');
                    $username.removeClass('show');
                }
            });
        });
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/kv/ttsConfig');
            if (!response.ok) {
                if (response.status === 404) {
                    console.log('No saved config found, using defaults');
                    // Apply default values
                    Object.values(this.configInputs).forEach(config => {
                        if (config.apply) {
                            if (config.input) {
                                config.apply(config.input.val());
                            } else {
                                config.apply(config.value);
                            }
                        }
                    });
                    return;
                }
                console.error('Failed to load config:', response.statusText);
                return;
            }
            
            const config = await response.json();
            Object.entries(config).forEach(([key, value]) => {
                if (this.configInputs[key]) {
                    if (key === 'containerPosition' || key === 'stackingOrder') {
                        this.configInputs[key].value = value;
                        this.configInputs[key].apply(value);
                        
                        if (key === 'containerPosition') {
                            $(`.position-btn[data-justify="${value.justify}"][data-align="${value.align}"]`)
                                .addClass('bg-white/20');
                        } else if (key === 'stackingOrder') {
                            const $toggle = $('#stackingOrderToggle');
                            $toggle.data('reversed', value);
                            $toggle.find('span').text(value ? 'Last Avatar on Top' : 'First Avatar on Top');
                            $toggle.find('svg').toggleClass('rotate-180', value);
                        }
                    } else if (this.configInputs[key].input) {
                        this.configInputs[key].input.val(value);
                        this.configInputs[key].value.text(value);
                        this.configInputs[key].apply(value);
                    }
                }
            });
        } catch (error) {
            console.error('Error loading config:', error);
            // Apply default values on error
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
    }

    async saveConfig() {
        const config = {};
        Object.entries(this.configInputs).forEach(([key, control]) => {
            if (key === 'containerPosition' || key === 'stackingOrder') {
                config[key] = control.value;
            } else if (control.input) {
                config[key] = control.input.val();
            }
        });

        try {
            const response = await fetch('/api/kv/ttsConfig', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (!response.ok) {
                throw new Error(`Failed to save config: ${response.statusText}`);
            }
        } catch (error) {
            console.error('Error saving config:', error);
        }
    }

    getConfigInputs() {
        return this.configInputs;
    }
}

// Export the ConfigManager
window.ConfigManager = ConfigManager; 