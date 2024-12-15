export class AvatarManager {
    constructor() {
        console.log('AvatarManager initialized');
        this.setupEventListeners();
    }

    setupEventListeners() {
        console.log('Setting up avatar upload event listeners');
        document.getElementById('upload-idle').addEventListener('click', () => this.uploadAvatar('idle'));
        document.getElementById('upload-talking').addEventListener('click', () => this.uploadAvatar('talking'));
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
    }
} 