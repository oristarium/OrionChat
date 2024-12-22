export class AvatarManager {
    constructor() {
        this.avatars = [];
        this.showToast = null;
        this.avatarImages = [];
    }

    async init() {
        try {
            await Promise.all([
                this.fetchAvatars(),
                this.fetchAvatarImages()
            ]);
        } catch (error) {
            console.error('Failed to initialize AvatarManager:', error);
            if (this.showToast) {
                this.showToast('Failed to load avatars', 'error');
            }
        }
    }

    async fetchAvatars() {
        try {
            const response = await fetch('/api/avatars');
            if (!response.ok) {
                throw new Error('Failed to fetch avatars');
            }
            const data = await response.json();
            this.avatars = data.avatars;
            if (this.onAvatarsChange) {
                this.onAvatarsChange(this.avatars);
            }
        } catch (error) {
            console.error('Error fetching avatars:', error);
            if (this.showToast) {
                this.showToast('Failed to fetch avatars', 'error');
            }
            throw error;
        }
    }

    async fetchAvatarImages() {
        try {
            const response = await fetch('/api/avatar-images');
            if (!response.ok) {
                throw new Error('Failed to fetch avatar images');
            }
            const data = await response.json();
            this.avatarImages = data['avatar-images'];
            if (this.onAvatarImagesChange) {
                this.onAvatarImagesChange(this.avatarImages);
            }
        } catch (error) {
            console.error('Error fetching avatar images:', error);
            if (this.showToast) {
                this.showToast('Failed to fetch avatar images', 'error');
            }
            throw error;
        }
    }

    async uploadAvatarImage(file, type) {
        try {
            const formData = new FormData();
            formData.append('avatar', file);
            formData.append('type', type);

            const response = await fetch('/api/avatar-images/upload', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error('Failed to upload avatar image');
            }

            const data = await response.json();
            await this.fetchAvatarImages(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Image uploaded successfully', 'success');
            }
            
            return data.path;
        } catch (error) {
            console.error('Error uploading avatar image:', error);
            if (this.showToast) {
                this.showToast('Failed to upload image', 'error');
            }
            throw error;
        }
    }

    async updateAvatarState(avatarId, states) {
        try {
            const response = await fetch(`/api/avatars/${avatarId}/set`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ states })
            });

            if (!response.ok) {
                throw new Error('Failed to update avatar state');
            }

            await this.fetchAvatars(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Avatar updated successfully', 'success');
            }
        } catch (error) {
            console.error('Error updating avatar state:', error);
            if (this.showToast) {
                this.showToast('Failed to update avatar', 'error');
            }
            throw error;
        }
    }

    async createAvatar(avatarData) {
        try {
            const response = await fetch('/api/avatars/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(avatarData)
            });

            if (!response.ok) {
                throw new Error('Failed to create avatar');
            }

            const newAvatar = await response.json();
            await this.fetchAvatars(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Avatar created successfully', 'success');
            }
            
            return newAvatar;
        } catch (error) {
            console.error('Error creating avatar:', error);
            if (this.showToast) {
                this.showToast('Failed to create avatar', 'error');
            }
            throw error;
        }
    }

    async deleteAvatar(avatarId) {
        try {
            const response = await fetch(`/api/avatars/${avatarId}/delete`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                if (response.status === 400) {
                    throw new Error('Cannot delete default avatar');
                } else if (response.status === 404) {
                    throw new Error('Avatar not found');
                }
                throw new Error('Failed to delete avatar');
            }

            await this.fetchAvatars(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Avatar deleted successfully', 'success');
            }
        } catch (error) {
            console.error('Error deleting avatar:', error);
            if (this.showToast) {
                this.showToast(error.message, 'error');
            }
            throw error;
        }
    }
} 