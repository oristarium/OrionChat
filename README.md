Read this in other languages: [English](README.md) | [Indonesia](README.id.md)

<div align="center">
<a href="https://discord.gg/JgjExyntw4"><img src="https://img.shields.io/discord/733027681184251937.svg?style=flat&label=Join%20Our%20Discord&color=7289DA" alt="Join Community Badge"/></a>
<a href="https://twitter.com/oristarium"><img src="https://img.shields.io/twitter/follow/oristarium.svg?style=social" /></a>
<a href="https://github.com/oristarium/orionchat/stargazers"><img src="https://img.shields.io/github/stars/oristarium/orionchat" alt="Stars Badge"/></a>
<a href="https://github.com/oristarium/orionchat/graphs/contributors"><img alt="GitHub contributors" src="https://img.shields.io/github/contributors/oristarium/orionchat?color=2b9348"></a>
<a href="https://github.com/oristarium/orionchat/blob/master/LICENSE"><img src="https://img.shields.io/github/license/oristarium/orionchat?color=2b9348" alt="License Badge"/></a>
<br>
<img src="assets/icon-big.png" alt="OrionChat logo" width="200" height="200"/>
<h1 align="center"> OrionChat </h1>
<i>A powerful OBS integration tool for managing live stream chats with TTS and avatar animations</i>
</div>

## ✨ Features

- 💬 **Multi-Platform Support** - Works with YouTube, TikTok, and Twitch
- 🗣️ **Text-to-Speech** - Multi-language TTS support with customizable avatar animations
- 💬 **Multiple TTS Providers** - Support for both Google Translate and TikTok voices
- 🗣️ **Rich Voice Selection** - Over 50 TikTok voices including character voices and singing voices
- 🎮 **Easy OBS Integration** - Simple browser source setup for chat display and TTS avatar
- 💬 **Multi-Avatar System** - Support for multiple active avatars with individual voice pools
- 🎨 **Advanced Avatar Management** - Customizable avatars with individual voice assignments and state management
- 🎯 **Real-time Chat Display** - Show highlighted messages on stream
- 🔧 **Control Panel** - User-friendly interface for managing all features


## 💖 Support Us

<a href="https://oristarium.com" target="_blank"><img alt="Oristarium Logo" src="https://ucarecdn.com/87bb45de-4a95-40d7-83c6-73866de942d5/-/crop/5518x2493/1408,2949/-/preview/1000x1000/" width="200"/></a>

We contribute to the VTuber community through open-source tools, technical resources, and knowledge sharing.  
Your support helps us continue building for the community!
<br><br>
<a href="https://trakteer.id/oristarium">
  <img src="https://cdn.trakteer.id/images/embed/trbtn-red-1.png" height="40" alt="Support us on Trakteer" />
</a>

## 🚀 Installation & Setup Guide

### Windows Installation
1. Download the latest Windows release (.zip file) from our [Releases Page](https://github.com/oristarium/orionchat/releases/latest)
2. Extract the downloaded .zip file
3. Open the extracted folder and double-click `orionchat.exe`
4. The app will open a status window showing the server is running on port 7777
   
   ![OrionChat Status Window](https://ucarecdn.com/841b9ca8-2d7d-43c6-b440-e8e7e1bc5628/orionchatstatusapp.png)

5. Use the provided copy buttons to get the necessary URLs for your streaming setup
6. Keep OrionChat running while streaming

⚠️ Note: Currently, OrionChat is only available for Windows. Mac and Linux support coming soon.

### 📺 Control Panel Setup in OBS
1. In OBS Studio, go to "View" → "Docks" → "Custom Browser Docks..."
2. In the popup window:
   - Enter "OrionChat Control" for the Dock Name
   - Enter `http://localhost:7777` for the URL
3. Click "Apply" then "Close"
4. The Control Panel will appear as a dockable window in OBS
5. You can now arrange it alongside your other OBS panels

### 📺 Adding Sources in OBS Studio
1. Open OBS Studio
2. Right-click in the "Sources" panel
3. Select "Add" → "Browser"
4. Create new and name it (example: "OrionChat Display")
5. Add these sources one by one:
   - Individual Avatars: `http://localhost:7777/avatar/{avatar_id}` (Width: 1080, Height: 1080)
   Note: You can add multiple individual avatar sources, each with its own ID

### 🎭 Using with VTube Studio
1. Open VTube Studio
2. Click the "+" button in the bottom right to add a new item
3. Select "Web Item"
4. Add these URLs one at a time:
   - For Individual Avatars: `http://localhost:7777/avatar/{avatar_id}`
     - Recommended size: 1080x1080 per avatar
5. Position and resize the web items as needed in your scene
6. Use the OBS Control Panel dock to manage settings

Note: For more detailed information about Web Items in VTube Studio, please refer to the [official VTube Studio Wiki](https://github.com/DenchiSoft/VTubeStudio/wiki/Web-Items/).

### 💡 Tips for Content Creators
- Make sure OrionChat is running before starting OBS or VTube Studio
- Test your setup before going live
- The Control Panel can be accessed from any browser at `http://localhost:7777`
- Customize your avatar and chat display settings through the Control Panel
- For best results, keep the application running in the background while streaming

### 🔧 Troubleshooting
- If you can't see the chat or avatar, make sure OrionChat is running
- Try refreshing your browser sources in OBS or web widgets in VTube Studio
- Check if your antivirus is blocking the application
- Make sure you're using the correct URLs
- If nothing works, restart OrionChat and your streaming software

## 🛠️ Configuration

### Avatar System
- **Multiple Independent Avatars**
  - Each avatar runs in its own browser source
  - Avatars coordinate through a central queue system
  - No overlapping speech between avatars
  - Automatic avatar selection with weighted distribution
  - Real-time state synchronization across all avatars

- **Voice Assignment**
  - Assign multiple voices to each avatar
  - Random voice selection from avatar's voice pool
  - Individual voice settings per avatar
  - Preview voices in control panel

- **State Management**
  - Idle and talking states per avatar
  - Automatic state transitions
  - Synchronized animations
  - No speech overlap between avatars
  - Smart queuing system for messages
  - Automatic cleanup of audio resources

### Avatar Index Page
- Browse all available avatars
- Preview idle and talking states
- View assigned voices for each avatar
- Direct links to individual avatar pages
- Real-time updates when avatars change

### TTS Providers
- **Google Translate TTS**
  - Simple language-based voices
  - Supports multiple languages
  
- **TikTok TTS**
  - 50+ unique character voices
  - Multiple voice styles per language
  - Special character voices (Disney, Star Wars, etc.)
  - Singing voices with different styles
  - Preview audio samples for each voice
  - Random voice selection from favorites
  - [View complete list of TikTok voices](assets/data/tiktok_voice_ids.csv)

### Voice Categories
TikTok voices include:
- 🎭 Character Voices (Disney, Star Wars, etc.)
- 🎵 Singing Voices
- 🌍 Multiple accents (UK, US, AU)
- 👥 Male/Female/Neutral options
- 🎬 Narrator/Storyteller voices
- 🎪 Fun/Novelty voices

### Supported Languages
- 🇮🇩 Indonesian
- 🇺🇸 English (US/UK/AU variants)
- 🇰🇷 Korean
- 🇯🇵 Japanese
- 🇫🇷 French
- 🇩🇪 German
- 🇪🇸 Spanish
- 🇵🇹 Portuguese
- And more...

### Platform Support
- YouTube (Channel ID or Username or Live ID)
- TikTok (Username)
- Twitch (Username)

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


