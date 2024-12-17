export class AvatarManager {
    constructor() {
        console.log('AvatarManager initialized');
        this.avatars = [];
        this.currentAvatar = null;
        this.initializeUI();
        this.setupEventListeners();
        this.loadAvatars();
    }

    async loadAvatars() {
        try {
            // Get all uploaded avatar images
            const listResponse = await fetch('/list-avatar-images');
            const { avatars: avatarImages } = await listResponse.json();
            
            // Get configured avatars (sets of idle/talking states)
            const avatarResponse = await fetch('/api/avatars');
            const { avatars: configuredAvatars, current_id } = await avatarResponse.json();
            
            // Store current avatar ID
            this.currentAvatarId = current_id;
            
            // Populate the grid with all available images
            this.populateAvatarGrid(avatarImages, configuredAvatars);
        } catch (error) {
            console.error('Error loading avatars:', error);
        }
    }

    initializeUI() {
        // Create modal dialog
        this.modal = $('<div>', {
            title: 'Edit Avatar States',
            class: 'avatar-edit-modal'
        }).dialog({
            autoOpen: false,
            modal: true,
            width: 500,
            buttons: {
                Save: () => this.saveAvatarStates(),
                Cancel: () => this.modal.dialog('close')
            }
        });

        // Create modal content
        this.modal.html(`
            <div class="avatar-states-editor">
                <div class="state-section">
                    <h3>Idle State</h3>
                    <input type="file" class="state-file" id="idle-state" accept="image/*">
                    <div class="preview">
                        <img id="idle-preview" src="" alt="Idle preview">
                    </div>
                </div>
                <div class="state-section">
                    <h3>Talking State</h3>
                    <input type="file" class="state-file" id="talking-state" accept="image/*">
                    <div class="preview">
                        <img id="talking-preview" src="" alt="Talking preview">
                    </div>
                </div>
            </div>
        `);
    }

    setupEventListeners() {
        // Handle avatar grid clicks
        const grid = document.getElementById('avatar-grid');
        if (grid) {
            grid.addEventListener('click', e => {
                const item = e.target.closest('.avatar-item');
                if (item) {
                    this.editAvatar(item.dataset.id);
                }
            });
        }

        // Handle new avatar button
        const addBtn = document.getElementById('add-avatar');
        if (addBtn) {
            addBtn.addEventListener('click', () => this.createNewAvatar());
        }

        // Handle file input changes
        this.modal.find('input[type="file"]').on('change', function() {
            const preview = $(this).siblings('.preview').find('img');
            if (this.files && this.files[0]) {
                const reader = new FileReader();
                reader.onload = e => preview.attr('src', e.target.result);
                reader.readAsDataURL(this.files[0]);
            }
        });
    }

    async createNewAvatar() {
        const avatar = {
            id: `avatar_${Date.now()}`,
            states: {}
        };
        this.editAvatar(avatar.id);
    }

    async editAvatar(id) {
        this.currentEditingId = id;
        const avatar = await this.getAvatar(id);
        
        // Set preview images
        $('#idle-preview').attr('src', avatar?.states?.idle || '');
        $('#talking-preview').attr('src', avatar?.states?.talking || '');
        
        this.modal.dialog('open');
    }

    async saveAvatarStates() {
        const idleFile = $('#idle-state')[0].files[0];
        const talkingFile = $('#talking-state')[0].files[0];
        
        try {
            if (idleFile) {
                await this.uploadAvatarState('idle', idleFile);
            }
            if (talkingFile) {
                await this.uploadAvatarState('talking', talkingFile);
            }
            
            this.modal.dialog('close');
            await this.loadAvatars();
        } catch (error) {
            console.error('Failed to save avatar states:', error);
            alert('Failed to save avatar states');
        }
    }

    async uploadAvatar(type, file) {
        console.log(`Uploading ${type} avatar:`, file);
        const formData = new FormData();
        formData.append('avatar', file);
        formData.append('type', type);

        try {
            const response = await fetch('/upload-avatar', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error('Upload failed');
            }

            const result = await response.json();
            console.log('Upload successful:', result);

            // Update grid
            await this.loadCurrentAvatarStates();
        } catch (error) {
            console.error('Upload error:', error);
            alert('Failed to upload avatar');
        }
    }

    setupAvatarSelection() {
        ['idle', 'talking'].forEach(type => {
            const grid = document.getElementById(`${type}-avatar-grid`);
            if (grid) {
                grid.addEventListener('click', async (e) => {
                    const item = e.target.closest('.avatar-item');
                    if (item) {
                        const img = item.querySelector('img');
                        const path = img.src.substring(img.src.indexOf('/avatars/'));
                        await this.updateAvatarState(type, path);
                    }
                });
            }
        });
    }

    async updateAvatarState(type, path) {
        try {
            console.log(`Updating ${type} state to:`, path);
            const response = await fetch('/set-avatar', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ 
                    type, 
                    path 
                })
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Server error:', errorText);
                throw new Error('Failed to update avatar state');
            }

            // Update grid selection
            const grid = document.getElementById(`${type}-avatar-grid`);
            grid.querySelectorAll('.avatar-item').forEach(item => {
                item.classList.toggle('selected', 
                    item.querySelector('img').src.endsWith(path));
            });

            // Update current avatar states
            await this.loadCurrentAvatarStates();
            
            console.log(`Successfully updated ${type} state to:`, path);
        } catch (error) {
            console.error('Error updating avatar state:', error);
            alert('Failed to update avatar state');
        }
    }

    async loadCurrentAvatarStates() {
        try {
            const response = await fetch('/get-avatars');
            const states = await response.json();
            const listResponse = await fetch('/list-avatars');
            const { avatars } = await listResponse.json();
            
            // Show all avatars in both grids
            this.populateAvatarGrid('idle', avatars, states.idle);
            this.populateAvatarGrid('talking', avatars, states.talking);
        } catch (error) {
            console.error('Error loading avatar states:', error);
        }
    }

    populateAvatarGrid(avatarImages, configuredAvatars) {
        const grid = document.getElementById('avatar-grid');
        if (!grid) {
            console.error('Avatar grid element not found');
            return;
        }

        if (!avatarImages || !configuredAvatars) {
            console.error('Missing avatar data:', { avatarImages, configuredAvatars });
            return;
        }

        // Get the current avatar's states
        const currentAvatar = configuredAvatars.find(a => a.id === this.currentAvatarId);
        const currentStates = currentAvatar?.states || {};

        // Create grid items for each image
        grid.innerHTML = avatarImages.map(path => `
            <div class="avatar-item ${path === currentStates.idle || path === currentStates.talking ? 'selected' : ''}"
                 data-path="${path}">
                <div class="avatar-preview">
                    <img src="${path}" alt="Avatar image">
                </div>
                <div class="avatar-info">
                    <div class="avatar-states">
                        ${path === currentStates.idle ? '<span class="state-badge">Idle</span>' : ''}
                        ${path === currentStates.talking ? '<span class="state-badge">Talking</span>' : ''}
                    </div>
                </div>
            </div>
        `).join('');
    }
} 