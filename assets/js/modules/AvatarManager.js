export class AvatarManager {
    constructor() {
        console.log('AvatarManager initialized');
        this.avatars = [];
        this.currentAvatarId = null;
        this.loadAvatars();
        this.setupEventListeners();
        this.createUploadModal();
    }

    createUploadModal() {
        const modalHtml = `
            <div id="avatar-upload-modal" title="Upload Avatar Image" style="display:none;">
                <form id="avatar-upload-form">
                    <input type="file" id="avatar-image-input" accept="image/png,image/gif,image/jpeg" />
                    <div class="upload-preview">
                        <img id="upload-preview-img" style="display:none;" />
                    </div>
                </form>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', modalHtml);

        this.uploadModal = $('#avatar-upload-modal').dialog({
            autoOpen: false,
            modal: true,
            width: 400,
            buttons: {
                Upload: () => this.handleImageUpload(),
                Cancel: () => this.uploadModal.dialog('close')
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
        return `
            <tr class="avatar-row" data-avatar-id="${avatar.id}">
                <td>
                    <input type="checkbox" 
                           class="avatar-active-toggle" 
                           data-avatar-id="${avatar.id}" 
                           ${avatar.is_active ? 'checked' : ''}>
                </td>
                <td>${avatar.name}</td>
                <td>
                    <div class="avatar-preview" data-state-type="idle">
                        <img src="${avatar.states.idle}" alt="Idle state" class="preview-img">
                    </div>
                </td>
                <td>
                    <div class="avatar-preview" data-state-type="talking">
                        <img src="${avatar.states.talking}" alt="Talking state" class="preview-img">
                    </div>
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
            const response = await fetch(`/api/avatars/${avatarId}/set`, {
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

    async handleImageUpload() {
        const fileInput = document.getElementById('avatar-image-input');
        const file = fileInput.files[0];
        
        if (!file || !this.currentUpload) {
            return;
        }

        const { avatarId, stateType } = this.currentUpload;
        
        try {
            // First upload the file
            const formData = new FormData();
            formData.append('avatar', file);
            formData.append('type', stateType);

            const uploadResponse = await fetch('/api/avatar-images/upload', {
                method: 'POST',
                body: formData
            });

            if (!uploadResponse.ok) {
                throw new Error('Failed to upload image');
            }

            const { path } = await uploadResponse.json();

            // Then update the avatar state
            const updateResponse = await fetch(`/api/avatars/${avatarId}/set`, {
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

            if (!updateResponse.ok) {
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
            console.error('Error updating avatar image:', error);
            alert('Failed to update avatar image. Please try again.');
        }
    }
} 