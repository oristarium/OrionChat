Baca ini dalam bahasa lain: [English](README.md) | [Indonesia](README.id.md)

<div align="center">
<a href="https://discord.gg/JgjExyntw4"><img src="https://img.shields.io/discord/733027681184251937.svg?style=flat&label=Join%20Our%20Discord&color=7289DA" alt="Join Community Badge"/></a>
<a href="https://twitter.com/oristarium"><img src="https://img.shields.io/twitter/follow/oristarium.svg?style=social" /></a>
<a href="https://github.com/oristarium/orionchat/stargazers"><img src="https://img.shields.io/github/stars/oristarium/orionchat" alt="Stars Badge"/></a>
<a href="https://github.com/oristarium/orionchat/graphs/contributors"><img alt="GitHub contributors" src="https://img.shields.io/github/contributors/oristarium/orionchat?color=2b9348"></a>
<a href="https://github.com/oristarium/orionchat/blob/master/LICENSE"><img src="https://img.shields.io/github/license/oristarium/orionchat?color=2b9348" alt="License Badge"/></a>
<br>
<img src="assets/icon-big.png" alt="OrionChat logo" width="200" height="200"/>
<h1 align="center"> OrionChat </h1>
<i>Alat integrasi OBS yang powerful untuk mengelola chat live stream dengan TTS dan animasi avatar</i>
</div>

## ✨ Fitur

- 🎮 **Dukungan Multi-Platform** - Bekerja dengan YouTube, TikTok, dan Twitch
- 🗣️ **Text-to-Speech** - Dukungan TTS multi-bahasa dengan animasi avatar yang dapat disesuaikan
- 💬 **Penyedia TTS Beragam** - Mendukung suara dari Google Translate dan TikTok
- 🗣️ **Pilihan Suara yang Kaya** - Lebih dari 50 suara TikTok termasuk suara karakter dan suara menyanyi
- 🎮 **Integrasi OBS yang Mudah** - Pengaturan browser source yang sederhana untuk tampilan chat dan avatar TTS
- 🗣 **Sistem Multi-Avatar** - Mendukung beberapa avatar aktif dengan kumpulan suara individual
- 🎨 **Manajemen Avatar Lanjutan** - Avatar yang dapat disesuaikan dengan pengaturan suara dan status individual
- 🎯 **Tampilan Chat Real-time** - Tampilkan pesan yang disorot di stream
- 🔧 **Panel Kontrol** - Antarmuka yang mudah digunakan untuk mengelola semua fitur


## 💖 Dukung Kami

<a href="https://oristarium.com" target="_blank"><img alt="Oristarium Logo" src="https://ucarecdn.com/87bb45de-4a95-40d7-83c6-73866de942d5/-/crop/5518x2493/1408,2949/-/preview/1000x1000/" width="200"/></a>

Kami berkontribusi untuk komunitas VTuber melalui perangkat open-source, sumber daya teknis, dan berbagi pengetahuan.  
Dukungan Anda membantu kami terus berkarya untuk komunitas!
<br><br>
<a href="https://trakteer.id/oristarium">
  <img src="https://cdn.trakteer.id/images/embed/trbtn-red-1.png" height="40" alt="Support us on Trakteer" />
</a>

## 🚀 Panduan Instalasi & Pengaturan

### Instalasi Windows
1. Unduh rilis Windows terbaru (file .zip) dari [Halaman Rilis](https://github.com/oristarium/orionchat/releases/latest)
2. Ekstrak file .zip yang sudah diunduh:
   - Klik kanan pada file .zip
   - Pilih "Extract All..."
   - Pilih lokasi (misalnya Desktop atau Documents)
   - Klik "Extract"
3. Buka folder hasil ekstrak dan klik dua kali `orionchat.exe`
4. Aplikasi akan membuka jendela status yang menunjukkan server berjalan di port 7777
   
   ![OrionChat Status Window](https://ucarecdn.com/841b9ca8-2d7d-43c6-b440-e8e7e1bc5628/orionchatstatusapp.png)

5. Gunakan tombol copy yang tersedia untuk mendapatkan URL yang diperlukan untuk pengaturan streaming Anda
6. Biarkan OrionChat tetap berjalan selama streaming

⚠️ Catatan: Saat ini, OrionChat hanya tersedia untuk Windows. Dukungan Mac dan Linux akan segera hadir.

### 🎮 Pengaturan Panel Kontrol di OBS
1. Di OBS Studio, buka "View" → "Docks" → "Custom Browser Docks..."
2. Di jendela popup:
   - Masukkan "OrionChat Control" untuk Nama Dock
   - Masukkan `http://localhost:7777` untuk URL
3. Klik "Apply" lalu "Close"
4. Panel Kontrol akan muncul sebagai jendela yang bisa dipindahkan di OBS
5. Anda sekarang bisa mengaturnya bersama panel OBS lainnya

### 📺 Menambahkan Source di OBS Studio
1. Buka OBS Studio
2. Klik kanan di panel "Sources"
3. Pilih "Add" → "Browser"
4. Buat baru dan beri nama (contoh: "OrionChat Display")
5. Tambahkan source ini satu per satu:
   - Avatar Individual: `http://localhost:7777/avatar/{avatar_id}` (Lebar: 1080, Tinggi: 1080)
   Catatan: Anda dapat menambahkan beberapa source avatar individual, masing-masing dengan ID sendiri

### 🎭 Menggunakan dengan VTube Studio
1. Buka VTube Studio
2. Klik tombol "+" di pojok kanan bawah untuk menambahkan item baru
3. Pilih "Web Item"
4. Tambahkan URL ini satu per satu:
   - Untuk Avatar Individual: `http://localhost:7777/avatar/{avatar_id}`
     - Ukuran yang disarankan: 1080x1080 per avatar
5. Atur posisi dan ukuran web item sesuai kebutuhan di scene Anda
6. Gunakan dock Panel Kontrol OBS untuk mengatur pengaturan

Catatan: Untuk informasi lebih detail tentang Web Item di VTube Studio, silakan lihat [Wiki resmi VTube Studio](https://github.com/DenchiSoft/VTubeStudio/wiki/Web-Items/).

### 💡 Tips untuk Content Creator
- Pastikan OrionChat berjalan sebelum memulai OBS atau VTube Studio
- Uji pengaturan Anda sebelum mulai live
- Panel Kontrol dapat diakses dari browser manapun di `http://localhost:7777`
- Sesuaikan pengaturan avatar dan tampilan chat melalui Panel Kontrol
- Untuk hasil terbaik, biarkan aplikasi tetap berjalan di background selama streaming

### 🔧 Pemecahan Masalah
- Jika Anda tidak dapat melihat chat atau avatar, pastikan OrionChat sedang berjalan
- Coba refresh browser source di OBS atau web item di VTube Studio
- Periksa apakah antivirus Anda memblokir aplikasi
- Pastikan Anda menggunakan URL yang benar
- Jika tidak ada yang berhasil, restart OrionChat dan software streaming Anda

## 🛠️ Konfigurasi

### Sistem Avatar
- **Avatar Individual yang Terpisah**
  - Setiap avatar berjalan di browser source-nya sendiri
  - Avatar berkoordinasi melalui sistem antrian pusat
  - Tidak ada tumpang tindih suara antar avatar
  - Pemilihan avatar otomatis dengan distribusi berbobot
  - Sinkronisasi status real-time di semua avatar

- **Pengaturan Suara**
  - Tetapkan beberapa suara untuk setiap avatar
  - Pemilihan suara acak dari kumpulan suara avatar
  - Pengaturan suara individual per avatar
  - Pratinjau suara di panel kontrol

- **Manajemen Status**
  - Status diam dan berbicara per avatar
  - Transisi status otomatis
  - Animasi tersinkronisasi
  - Tidak ada tumpang tindih suara antar avatar
  - Sistem antrian pesan yang cerdas
  - Pembersihan sumber daya audio otomatis

### Halaman Indeks Avatar
- Jelajahi semua avatar yang tersedia
- Pratinjau status diam dan berbicara
- Lihat suara yang ditetapkan untuk setiap avatar
- Tautan langsung ke halaman avatar individual
- Pembaruan real-time saat avatar berubah

### Penyedia TTS
- **Google Translate TTS**
  - Suara berbasis bahasa yang sederhana
  - Mendukung berbagai bahasa
  
- **TikTok TTS**
  - 50+ suara karakter unik
  - Beberapa gaya suara per bahasa
  - Suara karakter khusus (Disney, Star Wars, dll.)
  - Suara menyanyi dengan berbagai gaya
  - Pratinjau sampel audio untuk setiap suara
  - Pemilihan suara acak dari favorit
  - [Lihat daftar lengkap suara TikTok](assets/data/tiktok_voice_ids.csv)

### Kategori Suara
Suara TikTok meliputi:
- 🎭 Suara Karakter (Disney, Star Wars, dll.)
- 🎵 Suara Menyanyi
- 🌍 Beragam aksen (UK, US, AU)
- 👥 Opsi Pria/Wanita/Netral
- 🎬 Suara Narator/Pendongeng
- 🎪 Suara Lucu/Unik

### Bahasa yang Didukung
- 🇮🇩 Indonesia
- 🇺🇸 Inggris (varian US/UK/AU)
- 🇰🇷 Korea
- 🇯🇵 Jepang
- 🇫🇷 Prancis
- 🇩🇪 Jerman
- 🇪🇸 Spanyol
- 🇵🇹 Portugis
- Dan lainnya...

## 💖 Dukungan

Suka proyek ini? Silakan dukung kami:

<a href="https://trakteer.id/oristarium">
  <img src="https://cdn.trakteer.id/images/embed/trbtn-red-1.png" height="40" alt="Dukung kami di Trakteer" />
</a>

## 📜 Lisensi

Proyek ini dilisensikan di bawah Lisensi MIT - lihat file [LICENSE](LICENSE) untuk detail.