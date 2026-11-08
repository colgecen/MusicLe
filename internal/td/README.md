# internal/td — TerminalDownloader çekirdeği

Bu paketler, TerminalDownloader CLI'ından alınmış **saf Go** indirme/mödülleme
kodudur. Amaç: MusicLe'nin yt-dlp + ffmpeg dış bağımlılığını kaldırmak.

| Paket | Görev |
|---|---|
| `youtube` | InnerTube istemcisi, arama, akış seçimi, indirme, cookie/PO-token hatalarında Türkçeleştirme |
| `spotify` | Spotify URL/URI sınıflandırma + API istemcisi (opsiyonel, credsiz de çalışır) |
| `download` | Akış → dosya orkestrasyonu, MP3 dönüşümü, dosya adı üretimi |
| `encode` | WebM/Opus ve MP4/AAC → MP3 (saf Go: pion/opus, go-aac, shine) |
| `mp4audio` | MP4 box ayrıştırma + ADTS sarma (AAC-LC) |
| `audio` | ID3/metadata okuma-yazma, kapak gömme |
| `model` | Ortak tipler (TrackInfo, Library) |
| `util` | Dosya adı temizleme, süre biçimlendirme |
| `logging` | Küçük aktivite logu (varsayılan kapalı; MusicLe `SetFile` çağırmaz) |

Bilinçli olarak alınmayanlar: `storage`, `backup`, `cmd` — MusicLe'nin kendi
kütüphane/kayıt (`state.songlist`, `bridge.registerSong`) ve yedekleme katmanı
zaten var. TerminalDownloader'daki `SaveToPlaylist*` / `SetPlaylistArt`
fonksiyonları bu yüzden söküldü; kayıt işi `bridge.registerSong` üzerinden yapılır.

Testler ağa çıkmaz (`httptest` + `internal/td/testdata` sabitleri).
