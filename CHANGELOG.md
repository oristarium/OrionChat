# Changelog

All notable changes to OrionChat will be documented in this file.

## [v0.1.1] - 2024-03-18

### Added
- Multiple Avatar Support
  - Added ability to have multiple avatars active simultaneously
  - Each avatar can have multiple TTS voices assigned
  - Dynamic avatar switching during TTS playback
  - Random voice selection per avatar
- Enhanced Avatar Management
  - Avatar activation/deactivation system
  - Individual voice assignment per avatar
  - Avatar state management (idle/talking)
  - Avatar preview in control panel
- Improved TTS System
  - Random avatar selection for each message
  - Per-avatar voice pool
  - Sequential message queue across all avatars
  - Smooth transitions between avatars

### Changed
- Restructured TTS display to support multiple avatars
- Updated control panel for avatar-voice management
- Improved avatar state handling
- Enhanced message queue system for multi-avatar support

### Fixed
- Avatar state synchronization
- TTS voice assignment persistence
- Message queue handling for multiple avatars
- Avatar transition animations