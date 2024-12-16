export class AvatarManager {
    constructor() {
        console.log('AvatarManager initialized');
        this.currentAvatars = { idle: null, talking: null };
        this.initializeModal();
        this.setupEventListeners();
        this.loadAvatars();
    }

    initializeModal() {
        // Initialize tabs first
        $("#avatar-tabs").tabs();

        // Then initialize dialog
        $("#avatar-settings-modal").dialog({
            autoOpen: false,
            width: 500,
            modal: true
        });
    }

    async loadAvatars() {
        try {
            // Get current avatar settings
            const response = await fetch('/get-avatars');
            const current = await response.json();
            this.currentAvatars = current;

            // Get list of all avatars
            const listResponse = await fetch('/list-avatars');
            const { avatars } = await listResponse.json();

            // Populate grids
            this.populateAvatarGrid('idle', avatars);
            this.populateAvatarGrid('talking', avatars);
        } catch (error) {
            console.error('Error loading avatars:', error);
        }
    }

    populateAvatarGrid(type, avatars) {
        const grid = document.getElementById(`${type}-avatar-grid`);
        grid.innerHTML = '';

        avatars.forEach(path => {
            const item = document.createElement('div');
            item.className = 'avatar-item';
            if (path === this.currentAvatars[type]) {
                item.classList.add('selected');
            }

            const img = document.createElement('img');
            img.src = path;
            img.alt = `Avatar option`;

            item.appendChild(img);
            grid.appendChild(item);
        });
    }

    async setAvatar(type, path) {
        try {
            console.log(`Setting ${type} avatar to:`, path);
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
                throw new Error('Failed to set avatar');
            }

            // Update selected state in grid
            const grid = document.getElementById(`${type}-avatar-grid`);
            grid.querySelectorAll('.avatar-item').forEach(item => {
                item.classList.toggle('selected', 
                    item.querySelector('img').src.endsWith(path));
            });

            this.currentAvatars[type] = path;
            console.log(`Successfully set ${type} avatar to:`, path);
        } catch (error) {
            console.error('Error setting avatar:', error);
            alert('Failed to set avatar');
        }
    }

    setupEventListeners() {
        console.log('Setting up avatar upload event listeners');
        this.setupUploadHandlers();
        this.setupAvatarSelection();
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

    setupAvatarSelection() {
        ['idle', 'talking'].forEach(type => {
            const grid = document.getElementById(`${type}-avatar-grid`);
            if (grid) {
                grid.addEventListener('click', (e) => {
                    const item = e.target.closest('.avatar-item');
                    if (item) {
                        const img = item.querySelector('img');
                        const path = img.src.substring(img.src.indexOf('/avatars/'));
                        this.setAvatar(type, path);
                    }
                });
            }
        });
    }

    handleUpload(type, fileInput) {
        if (!fileInput.files.length) {
            console.warn('No file selected');
            alert('Please select a file first');
            return;
        }
        this.uploadAvatar(type);
    }

    async uploadAvatar(type) {
        console.log(`Starting upload for ${type} avatar`);
        const fileInput = document.getElementById(`${type}-avatar`);
        const file = fileInput.files[0];
        
        if (!file) {
            console.warn('No file selected');
            alert('Please select a file first');
            return;
        }

        console.log(`File selected:`, {
            name: file.name,
            type: file.type,
            size: file.size
        });

        const formData = new FormData();
        formData.append('avatar', file);
        formData.append('type', type);

        try {
            console.log('Sending upload request to server...');
            const response = await fetch('/upload-avatar', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                console.error('Upload failed with status:', response.status);
                throw new Error('Upload failed');
            }

            const result = await response.json();
            console.log(`Upload successful:`, result);
            fileInput.value = ''; // Clear the input
        } catch (error) {
            console.error('Upload error details:', error);
            alert('Failed to upload avatar');
        }

        // After successful upload, reload the avatars
        await this.loadAvatars();
    }
} 