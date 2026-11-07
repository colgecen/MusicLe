// Package engine is MusicLeCLI's self-contained download engine: a pure-Go
// pipeline that turns YouTube or Spotify links into MP3 files with title,
// artist and cover art embedded — no external binaries required.
//
// Pipeline:
//
//  1. internal/td/youtube negotiates an InnerTube client and resolves the
//     playable audio stream (or scores a search for free-text input).
//  2. internal/td/download transfers it with resume support and re-encodes
//     WebM/Opus or MP4/AAC to MP3 with internal/td/encode (pure Go).
//  3. Title/artist/cover are written as ID3v2 (APIC) during encoding, so the
//     file that lands in the library is already player-ready.
//
// Spotify links are resolved through Spotify's public embed page (no API,
// no token) into an "Artist - Title" query that the same pipeline downloads.
package engine
