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

    loadConfig() {
        const savedConfig = localStorage.getItem('ttsConfig');
        if (savedConfig) {
            const config = JSON.parse(savedConfig);
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
        }
    }

    saveConfig() {
        const config = {};
        Object.entries(this.configInputs).forEach(([key, control]) => {
            if (key === 'containerPosition' || key === 'stackingOrder') {
                config[key] = control.value;
            } else if (control.input) {
                config[key] = control.input.val();
            }
        });
        localStorage.setItem('ttsConfig', JSON.stringify(config));
    }

    getConfigInputs() {
        return this.configInputs;
    }
}

// Export the ConfigManager
window.ConfigManager = ConfigManager; 