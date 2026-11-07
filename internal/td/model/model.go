package model

// AudioFileMetadata describes metadata extracted from a single audio file.
type AudioFileMetadata struct {
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album,omitempty"`
	Format   string `json:"format"`
	Duration int    `json:"duration"` // seconds; only MP3, otherwise 0
}

// Song represents one line in song_list.txt.
type Song struct {
	Filename  string `json:"filename"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	DateAdded string `json:"date_added"`
	Duration  string `json:"duration"` // mm:ss
	FilePath  string `json:"file_path,omitempty"`
	Source    string `json:"source,omitempty"`
}

// Playlist represents a playlist folder and its metadata files.
type Playlist struct {
	FolderName string `json:"folder_name"`
	Name       string `json:"name"`
	Bio        string `json:"bio,omitempty"`
	CreatedAt  string `json:"created_at"`
	ArtPath    string `json:"art_path,omitempty"`
	Songs      []Song `json:"songs"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// TrackInfo carries information gathered during a download.
type TrackInfo struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album,omitempty"`
	DurationSec int    `json:"duration_sec"`
	StreamURL   string `json:"stream_url"`
	Format      string `json:"format"` // webm, m4a, opus, ...
	ContentLen  int64  `json:"content_len"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Playlist    string `json:"playlist,omitempty"`
	TrackNum    int    `json:"track_num,omitempty"`
	Source      string `json:"source,omitempty"` // youtube id / search query
}

// Result is a generic operation result.
type Result struct {
	Status   string             `json:"status"`
	Error    string             `json:"error,omitempty"`
	Message  string             `json:"message,omitempty"`
	Title    string             `json:"title,omitempty"`
	Artist   string             `json:"artist,omitempty"`
	Album    string             `json:"album,omitempty"`
	Duration float64            `json:"duration,omitempty"`
	Filename string             `json:"filename,omitempty"`
	Format   string             `json:"format,omitempty"`
	ArtPath  string             `json:"art_path,omitempty"`
	Songs    []Song             `json:"songs,omitempty"`
	Meta     *AudioFileMetadata `json:"meta,omitempty"`
	Track    *TrackInfo         `json:"track,omitempty"`
}

// DownloadProgress tracks an in-flight download.
type DownloadProgress struct {
	Active  bool    `json:"active"`
	Percent float64 `json:"percent"`
	Speed   string  `json:"speed,omitempty"`
	Message string  `json:"message,omitempty"`
}

// Profile represents a user profile (optional, for storage layout compatibility).
type Profile struct {
	FolderName  string     `json:"folder_name"`
	DisplayName string     `json:"display_name"`
	Bio         string     `json:"bio,omitempty"`
	Language    string     `json:"language,omitempty"`
	Playlists   []Playlist `json:"playlists,omitempty"`
}

// Config is the persisted application configuration.
type Config struct {
	RootDir             string `json:"root_dir"`
	Language            string `json:"language"`
	LastProfile         string `json:"last_profile,omitempty"`
	Theme               string `json:"theme,omitempty"`
	Concurrency         int    `json:"concurrency,omitempty"`
	MP3                 *bool  `json:"mp3,omitempty"`
	SpotifyClientID     string `json:"spotify_client_id,omitempty"`
	SpotifySecret       string `json:"spotify_client_secret,omitempty"`
	SpotifyRefreshToken string `json:"spotify_refresh_token,omitempty"`
	YouTubeAPIKey       string `json:"youtube_api_key,omitempty"`
	LogFile             string `json:"log_file,omitempty"`
}

// MP3Enabled reports whether downloads should be converted to MP3. A nil value
// means "never configured", which defaults to enabled.
func MP3Enabled(c *Config) bool {
	if c == nil || c.MP3 == nil {
		return true
	}
	return *c.MP3
}
