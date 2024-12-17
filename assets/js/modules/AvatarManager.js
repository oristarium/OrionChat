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
            const response = await fetch('/get-avatars');
            const states = await response.json();
            const listResponse = await fetch('/list-avatars');
            const { avatars } = await listResponse.json();
            
            // Show all avatars in both grids
            this.populateAvatarGrid('idle', avatars, states.idle);
            this.populateAvatarGrid('talking', avatars, states.talking);
        } catch (error) {
            console.error('Error loading avatars:', error);
        }
    }

    initializeUI() {
        this.avatarList = document.querySelector('.avatar-list');
        this.addAvatarBtn = document.getElementById('add-avatar');
        this.avatarDetails = document.querySelector('.avatar-details');
    }

    setupEventListeners() {
        this.setupUploadHandlers();
        this.setupAvatarSelection();
        
        // Add new avatar
        this.addAvatarBtn?.addEventListener('click', () => this.showAddAvatarDialog());
    }

    setupUploadHandlers() {
        const uploadButtons = {
            idle: document.getElementById('upload-idle'),
            talking: document.getElementById('upload-talking')
        };

        const fileInputs = {
            idle: document.getElementById('idle-avatar'),
            talking: document.getElementById('talking-avatar')
        };

        Object.entries(uploadButtons).forEach(([type, button]) => {
            if (button) {
                button.addEventListener('click', () => this.handleUpload(type, fileInputs[type]));
            }
        });
    }

    handleUpload(type, fileInput) {
        if (!fileInput.files.length) {
            console.warn('No file selected');
            alert('Please select a file first');
            return;
        }
        this.uploadAvatar(type, fileInput.files[0]);
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

    populateAvatarGrid(type, paths, currentPath) {
        const grid = document.getElementById(`${type}-avatar-grid`);
        if (!grid) return;

        grid.innerHTML = paths.map(path => `
            <div class="avatar-item ${path === currentPath ? 'selected' : ''}">
                <img src="${path}" alt="Avatar option">
            </div>
        `).join('');
    }
} 