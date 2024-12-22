export class AvatarManager {
    constructor() {
        this.avatars = [];
        this.showToast = null;
        this.avatarImages = [];
        this.voices = [];
        this.currentAudio = null;
    }

    async init() {
        try {
            await Promise.all([
                this.fetchAvatars(),
                this.fetchAvatarImages(),
                this.loadVoices()
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

    async deleteAvatarImage(path) {
        try {
            const response = await fetch('/api/avatar-images/delete', {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ path })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error);
            }

            await this.fetchAvatarImages(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Image deleted successfully', 'success');
            }
        } catch (error) {
            console.error('Error deleting avatar image:', error);
            if (this.showToast) {
                this.showToast(error.message, 'error');
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

    async loadVoices() {
        try {
            const response = await fetch('/data/voices.csv');
            const csvText = await response.text();
            
            // Parse CSV
            const lines = csvText.split('\n').filter(line => line.trim());
            const headers = lines[0].split(',').map(h => h.trim());
            
            this.voices = lines.slice(1).map(line => {
                const values = line.split(',').map(v => v.trim());
                const voice = headers.reduce((obj, header, index) => {
                    // Don't include empty values
                    if (values[index] && values[index] !== '') {
                        obj[header] = values[index];
                    }
                    return obj;
                }, {});
                
                // Only include if it has required fields
                if (voice.provider && voice.voice_id) {
                    return voice;
                }
                return null;
            }).filter(Boolean); // Remove null entries

            if (this.onVoicesChange) {
                this.onVoicesChange(this.voices);
            }
        } catch (error) {
            console.error('Error loading voices:', error);
            if (this.showToast) {
                this.showToast('Failed to load voices', 'error');
            }
        }
    }

    async getAvatarVoices(avatarId) {
        try {
            const response = await fetch(`/api/avatars/${avatarId}/voices`);
            if (!response.ok) {
                throw new Error('Failed to fetch avatar voices');
            }
            const data = await response.json();
            return data.voices;
        } catch (error) {
            console.error('Error fetching avatar voices:', error);
            if (this.showToast) {
                this.showToast('Failed to fetch avatar voices', 'error');
            }
            throw error;
        }
    }

    async updateAvatarVoices(avatarId, voices) {
        try {
            const response = await fetch(`/api/avatars/${avatarId}/voices`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ voices })
            });

            if (!response.ok) {
                throw new Error('Failed to update avatar voices');
            }

            await this.fetchAvatars(); // Refresh the list
            
            if (this.showToast) {
                this.showToast('Voices updated successfully', 'success');
            }
        } catch (error) {
            console.error('Error updating avatar voices:', error);
            if (this.showToast) {
                this.showToast('Failed to update voices', 'error');
            }
            throw error;
        }
    }

    playVoiceSample(voice) {
        if (this.currentAudio) {
            this.currentAudio.pause();
            this.currentAudio = null;
        }

        if (voice.sample_voice) {
            this.currentAudio = new Audio(voice.sample_voice);
            this.currentAudio.play().catch(error => {
                console.error('Error playing voice sample:', error);
                if (this.showToast) {
                    this.showToast('Failed to play voice sample', 'error');
                }
            });
        }
    }

    stopVoiceSample() {
        if (this.currentAudio) {
            this.currentAudio.pause();
            this.currentAudio = null;
        }
    }
} 