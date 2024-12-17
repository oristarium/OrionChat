export class AvatarManager {
    constructor() {
        console.log('AvatarManager initialized');
        this.avatars = [];
        this.currentAvatarId = null;
        this.loadAvatars();
        this.setupEventListeners();
        this.createUploadModal();
        this.setupAddAvatarButton();
    }

    createUploadModal() {
        const modalHtml = `
            <div id="avatar-upload-modal" title="Change Avatar Image" style="display:none;">
                <div id="avatar-image-tabs">
                    <ul>
                        <li><a href="#upload-tab">Upload New</a></li>
                        <li><a href="#existing-tab">Choose Existing</a></li>
                    </ul>
                    <div id="upload-tab">
                        <form id="avatar-upload-form">
                            <input type="file" id="avatar-image-input" accept="image/png,image/gif,image/jpeg" />
                            <div class="upload-preview">
                                <img id="upload-preview-img" style="display:none;" />
                            </div>
                        </form>
                    </div>
                    <div id="existing-tab">
                        <div class="existing-images-grid" id="existing-images-grid">
                            <!-- Images will be populated here -->
                        </div>
                    </div>
                </div>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', modalHtml);

        // Initialize jQuery UI dialog and tabs
        this.uploadModal = $('#avatar-upload-modal').dialog({
            autoOpen: false,
            modal: true,
            width: 500,
            height: 500,
            buttons: {
                Apply: () => this.handleImageSelection(),
                Cancel: () => this.uploadModal.dialog('close')
            }
        });

        $('#avatar-image-tabs').tabs({
            activate: (event, ui) => {
                if (ui.newPanel.is('#existing-tab')) {
                    this.loadExistingImages();
                }
            }
        });
    }

    setupEventListeners() {
        // Listen for changes on avatar checkboxes
        document.addEventListener('change', async (e) => {
            if (e.target.classList.contains('avatar-active-toggle')) {
                const avatarId = e.target.dataset.avatarId;
                const isActive = e.target.checked;
                await this.updateAvatarActive(avatarId, isActive);
            }
        });

        // Add click handlers for avatar images
        document.addEventListener('click', (e) => {
            const previewImg = e.target.closest('.preview-img');
            if (!previewImg) return;

            const row = previewImg.closest('.avatar-row');
            if (!row) return;

            const avatarId = row.dataset.avatarId;
            const stateType = previewImg.closest('[data-state-type]')?.dataset.stateType;
            
            if (avatarId && stateType) {
                this.openUploadModal(avatarId, stateType);
            }
        });

        // Add preview for selected file
        document.getElementById('avatar-image-input')?.addEventListener('change', (e) => {
            const file = e.target.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = (e) => {
                    const previewImg = document.getElementById('upload-preview-img');
                    previewImg.src = e.target.result;
                    previewImg.style.display = 'block';
                };
                reader.readAsDataURL(file);
            }
        });

        // Add click handler for delete buttons
        document.addEventListener('click', async (e) => {
            const deleteBtn = e.target.closest('.delete-avatar-btn');
            if (!deleteBtn) return;

            const avatarId = deleteBtn.dataset.avatarId;
            if (confirm('Are you sure you want to delete this avatar?')) {
                await this.deleteAvatar(avatarId);
            }
        });
    }

    async loadAvatars() {
        try {
            const response = await fetch('/api/avatars');
            const data = await response.json();
            
            this.avatars = data.avatars;
            this.currentAvatarId = data.current_id;
            
            this.renderAvatarList();
        } catch (error) {
            console.error('Error loading avatars:', error);
        }
    }

    renderAvatarList() {
        const tbody = document.getElementById('avatar-list');
        if (!tbody) return;

        tbody.innerHTML = this.avatars.map(avatar => this.createAvatarRow(avatar)).join('');
    }

    createAvatarRow(avatar) {
        if (!avatar || !avatar.id) {
            console.error('Invalid avatar data:', avatar);
            return '';
        }

        return `
            <tr class="avatar-row" data-avatar-id="${avatar.id}">
                <td>
                    <input type="checkbox" 
                           class="avatar-active-toggle" 
                           data-avatar-id="${avatar.id}" 
                           ${avatar.is_active ? 'checked' : ''}>
                </td>
                <td>
                    <div class="avatar-preview" data-state-type="idle">
                        <img src="${avatar.states?.idle || ''}" alt="Idle state" class="preview-img">
                    </div>
                </td>
                <td>
                    <div class="avatar-preview" data-state-type="talking">
                        <img src="${avatar.states?.talking || ''}" alt="Talking state" class="preview-img">
                    </div>
                </td>
                <td>
                    <button class="delete-avatar-btn" data-avatar-id="${avatar.id}" ${avatar.is_default ? 'disabled' : ''}>
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <path d="M2 4h12M5.333 4V2.667a1.333 1.333 0 011.334-1.334h2.666a1.333 1.333 0 011.334 1.334V4m2 0v9.333a1.333 1.333 0 01-1.334 1.334H4.667a1.333 1.333 0 01-1.334-1.334V4h9.334z" 
                                  stroke="currentColor" 
                                  stroke-width="1.5" 
                                  stroke-linecap="round" 
                                  stroke-linejoin="round"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `;
    }

    async updateAvatarActive(avatarId, isActive) {
        const checkbox = document.querySelector(`.avatar-active-toggle[data-avatar-id="${avatarId}"]`);
        if (checkbox) {
            // Disable checkbox during update
            checkbox.disabled = true;
            
            // Add loading state to row
            const row = checkbox.closest('.avatar-row');
            if (row) row.classList.add('loading');
        }
    
        try {
            // Fix: Use the correct URL format with avatarId in the path
            const response = await fetch(`/api/avatars/${avatarId}/set`, {  // Changed from /api/avatars/set
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    is_active: isActive
                })
            });
    
            if (!response.ok) {
                throw new Error('Failed to update avatar active state');
            }
    
            // Update local state
            const avatar = this.avatars.find(a => a.id === avatarId);
            if (avatar) {
                avatar.is_active = isActive;
            }
        } catch (error) {
            console.error('Error updating avatar active state:', error);
            // Revert the checkbox state on error
            if (checkbox) {
                checkbox.checked = !isActive;
            }
        } finally {
            // Re-enable checkbox and remove loading state
            if (checkbox) {
                checkbox.disabled = false;
                const row = checkbox.closest('.avatar-row');
                if (row) row.classList.remove('loading');
            }
        }
    }

    openUploadModal(avatarId, stateType) {
        // Reset form and store current context
        document.getElementById('avatar-upload-form').reset();
        document.getElementById('upload-preview-img').style.display = 'none';
        
        this.currentUpload = { avatarId, stateType };
        
        // Open modal
        this.uploadModal.dialog('open');
    }

    async loadExistingImages() {
        const grid = document.getElementById('existing-images-grid');
        grid.innerHTML = '<div class="loading-indicator">Loading images...</div>';

        try {
            const response = await fetch('/api/avatar-images');
            const data = await response.json();

            grid.innerHTML = data['avatar-images'].map(path => `
                <div class="existing-image-item ${this.isCurrentImage(path) ? 'selected' : ''}" data-path="${path}">
                    <img src="${path}" alt="Avatar image" />
                </div>
            `).join('');

            // Add click handlers for selection
            grid.querySelectorAll('.existing-image-item').forEach(item => {
                item.addEventListener('click', () => {
                    grid.querySelectorAll('.existing-image-item').forEach(i => i.classList.remove('selected'));
                    item.classList.add('selected');
                });
            });
        } catch (error) {
            console.error('Error loading existing images:', error);
            grid.innerHTML = '<div class="error-message">Failed to load images</div>';
        }
    }

    isCurrentImage(path) {
        if (!this.currentUpload) return false;
        const avatar = this.avatars.find(a => a.id === this.currentUpload.avatarId);
        return avatar && avatar.states[this.currentUpload.stateType] === path;
    }

    async handleImageSelection() {
        const activeTab = $('#avatar-image-tabs').tabs('option', 'active');
        
        if (activeTab === 0) {
            // Upload tab
            await this.handleImageUpload();
        } else {
            // Existing images tab
            const selectedImage = document.querySelector('.existing-image-item.selected');
            if (selectedImage && this.currentUpload) {
                const path = selectedImage.dataset.path;
                await this.updateAvatarState(this.currentUpload.avatarId, this.currentUpload.stateType, path);
            }
        }
    }

    async updateAvatarState(avatarId, stateType, path) {
        try {
            const response = await fetch(`/api/avatars/${avatarId}/set`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    states: {
                        [stateType]: path
                    }
                })
            });

            if (!response.ok) {
                throw new Error('Failed to update avatar state');
            }

            // Update local state and UI
            const avatar = this.avatars.find(a => a.id === avatarId);
            if (avatar) {
                avatar.states[stateType] = path;
                this.renderAvatarList();
            }

            // Close modal
            this.uploadModal.dialog('close');

        } catch (error) {
            console.error('Error updating avatar state:', error);
            alert('Failed to update avatar state. Please try again.');
        }
    }

    // Modify handleImageUpload to use updateAvatarState
    async handleImageUpload() {
        const fileInput = document.getElementById('avatar-image-input');
        const file = fileInput.files[0];
        
        if (!file || !this.currentUpload) {
            return;
        }

        try {
            // First upload the file
            const formData = new FormData();
            formData.append('avatar', file);
            formData.append('type', this.currentUpload.stateType);

            const uploadResponse = await fetch('/api/avatar-images/upload', {
                method: 'POST',
                body: formData
            });

            if (!uploadResponse.ok) {
                throw new Error('Failed to upload image');
            }

            const { path } = await uploadResponse.json();

            // Update the avatar state with the new path
            await this.updateAvatarState(
                this.currentUpload.avatarId, 
                this.currentUpload.stateType, 
                path
            );

        } catch (error) {
            console.error('Error uploading image:', error);
            alert('Failed to upload image. Please try again.');
        }
    }

    setupAddAvatarButton() {
        // Add button to the avatar management header
        const avatarManagement = document.querySelector('.avatar-management');
        const headerHtml = `
            <div class="avatar-header">
                <h3>Avatar Management</h3>
                <button id="add-avatar-btn" class="btn btn-primary btn-sm">
                    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M6 1V11M1 6H11" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                    </svg>
                    New Avatar
                </button>
            </div>
        `;
        avatarManagement.insertAdjacentHTML('afterbegin', headerHtml);

        // Add click handler
        document.getElementById('add-avatar-btn')?.addEventListener('click', () => {
            this.createNewAvatar();
        });
    }

    async createNewAvatar() {
        const button = document.getElementById('add-avatar-btn');
        if (!button) return;

        try {
            // Disable button and show loading state
            button.disabled = true;
            button.innerHTML = `
                <svg class="spinner" width="12" height="12" viewBox="0 0 24 24">
                    <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="4"/>
                </svg>
                Creating...
            `;

            // Make API request
            const response = await fetch('/api/avatars/create', {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error('Failed to create avatar');
            }

            // Get created avatar data
            const newAvatar = await response.json();

            // Add to local state
            this.avatars.push(newAvatar);
            
            // Update UI
            this.renderAvatarList();

            // Scroll to new avatar
            const newRow = document.querySelector(`[data-avatar-id="${newAvatar.id}"]`);
            if (newRow) {
                newRow.scrollIntoView({ behavior: 'smooth', block: 'center' });
                newRow.classList.add('highlight');
                setTimeout(() => newRow.classList.remove('highlight'), 2000);
            }

        } catch (error) {
            console.error('Error creating avatar:', error);
            alert('Failed to create new avatar. Please try again.');
        } finally {
            // Reset button state
            button.disabled = false;
            button.innerHTML = `
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M6 1V11M1 6H11" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                </svg>
                New Avatar
            `;
        }
    }

    async deleteAvatar(avatarId) {
        const row = document.querySelector(`tr[data-avatar-id="${avatarId}"]`);
        if (!row) return;

        try {
            // Add loading state
            row.classList.add('loading');
            const deleteBtn = row.querySelector('.delete-avatar-btn');
            if (deleteBtn) {
                deleteBtn.disabled = true;
            }

            const response = await fetch(`/api/avatars/${avatarId}/delete`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.message || 'Failed to delete avatar');
            }

            // Remove from local state
            this.avatars = this.avatars.filter(a => a.id !== avatarId);
            
            // Animate row removal
            row.style.transition = 'opacity 0.3s, transform 0.3s';
            row.style.opacity = '0';
            row.style.transform = 'translateX(-10px)';
            
            // Remove row after animation
            setTimeout(() => {
                this.renderAvatarList();
            }, 300);

        } catch (error) {
            console.error('Error deleting avatar:', error);
            alert(error.message || 'Failed to delete avatar. Please try again.');
            
            // Remove loading state on error
            row.classList.remove('loading');
            const deleteBtn = row.querySelector('.delete-avatar-btn');
            if (deleteBtn) {
                deleteBtn.disabled = false;
            }
        }
    }
} 