package spotify

import (
	"context"
	"fmt"
	"strings"
	"time"

	spotify "github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"MusicLe/internal/td/model"
)

// FullTrack is a Spotify track (alias of the upstream type).
type FullTrack = spotify.FullTrack

// Client is a Spotify Web API client.
type Client struct {
	inner *spotify.Client
}

// NewClient creates a client using client credentials.
func NewClient(ctx context.Context, clientID, clientSecret string) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("spotify client id/secret gerekli")
	}
	cfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts.spotify.com/api/token",
	}
	return &Client{inner: spotify.New(cfg.Client(ctx))}, nil
}

// NewAuthClient creates a client from a stored refresh token (OAuth user
// authorization). clientID/secret are still required to refresh the token.
func NewAuthClient(ctx context.Context, clientID, clientSecret, refreshToken string) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("spotify client id/secret gerekli")
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("spotify refresh token gerekli")
	}
	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}
	tok := &oauth2.Token{RefreshToken: refreshToken}
	src := oauthCfg.TokenSource(ctx, tok)
	httpClient := oauth2.NewClient(ctx, src)
	return &Client{inner: spotify.New(httpClient)}, nil
}

// ParseID extracts (kind, id) from a Spotify URL, URI or bare ID.
// kind is "track", "playlist", "album", "artist" or "unknown".
//
// Deprecated: use Classify, which also separates "no kind in the input" from
// "not a Spotify reference at all".
func ParseID(input string) (string, string) {
	k, id := Classify(input)
	return k.String(), id
}

// ResolveAny determines what a bare Spotify id refers to by asking the API.
// Tracks, playlists and albums share the same id shape, so a plain 22-character
// id cannot be classified without a lookup.
func (c *Client) ResolveAny(ctx context.Context, input string) (Kind, string, error) {
	if k, id := Classify(input); k != KindUnknown {
		return k, id, nil
	}
	_, id := Classify(input)
	if !LooksLikeID(id) {
		return KindUnknown, "", fmt.Errorf("spotify linki/ID'si taninamadi: %q", input)
	}
	if _, err := c.GetTrack(ctx, id); err == nil {
		return KindTrack, id, nil
	}
	if _, err := c.GetPlaylist(ctx, id); err == nil {
		return KindPlaylist, id, nil
	}
	if _, err := c.GetAlbum(ctx, id); err == nil {
		return KindAlbum, id, nil
	}
	return KindUnknown, id, fmt.Errorf("bu Spotify ID'si sarki, playlist veya album olarak bulunamadi: %s", id)
}

// GetTrack fetches a single track.
func (c *Client) GetTrack(ctx context.Context, id string) (*spotify.FullTrack, error) {
	return c.inner.GetTrack(ctx, spotify.ID(id))
}

// GetPlaylist fetches a playlist with its tracks.
func (c *Client) GetPlaylist(ctx context.Context, id string) (*spotify.FullPlaylist, error) {
	return c.inner.GetPlaylist(ctx, spotify.ID(id))
}

// GetAlbum fetches an album.
func (c *Client) GetAlbum(ctx context.Context, id string) (*spotify.FullAlbum, error) {
	return c.inner.GetAlbum(ctx, spotify.ID(id))
}

// PlaylistTracks returns every track in a playlist, following pagination.
func (c *Client) PlaylistTracks(ctx context.Context, id string) ([]spotify.PlaylistTrack, error) {
	const pageSize = 100
	var all []spotify.PlaylistTrack
	offset := 0
	for {
		page, err := c.inner.GetPlaylistTracks(ctx, spotify.ID(id), spotify.Limit(pageSize), spotify.Offset(offset))
		if err != nil {
			return all, err
		}
		all = append(all, page.Tracks...)
		offset += len(page.Tracks)
		if len(page.Tracks) == 0 || offset >= int(page.Total) {
			break
		}
	}
	return all, nil
}

// AlbumTracks returns every track in an album, following pagination.
func (c *Client) AlbumTracks(ctx context.Context, id string) ([]spotify.SimpleTrack, error) {
	const pageSize = 50
	var all []spotify.SimpleTrack
	offset := 0
	for {
		page, err := c.inner.GetAlbumTracks(ctx, spotify.ID(id), spotify.Limit(pageSize), spotify.Offset(offset))
		if err != nil {
			return all, err
		}
		all = append(all, page.Tracks...)
		offset += len(page.Tracks)
		if len(page.Tracks) == 0 || offset >= int(page.Total) {
			break
		}
	}
	return all, nil
}

// SearchTrack searches Spotify and returns the first matching track.
func (c *Client) SearchTrack(ctx context.Context, query string) (*spotify.FullTrack, error) {
	res, err := c.inner.Search(ctx, query, spotify.SearchTypeTrack)
	if err != nil {
		return nil, err
	}
	if res.Tracks == nil || len(res.Tracks.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks for %q", query)
	}
	return &res.Tracks.Tracks[0], nil
}

// SearchQuery builds a "Artist - Title" query for a track.
func SearchQuery(t *spotify.FullTrack) string {
	artist := ""
	if len(t.Artists) > 0 {
		artist = t.Artists[0].Name
	}
	return strings.TrimSpace(fmt.Sprintf("%s - %s", artist, t.Name))
}

// PlaylistTrackToQuery builds a search query for a playlist entry.
func PlaylistTrackToQuery(t spotify.PlaylistTrack) string {
	artist := ""
	if len(t.Track.Artists) > 0 {
		artist = t.Track.Artists[0].Name
	}
	return strings.TrimSpace(fmt.Sprintf("%s - %s", artist, t.Track.Name))
}

// SimpleTrackSearchQuery builds a search query for an album track.
func SimpleTrackSearchQuery(t spotify.SimpleTrack) string {
	artist := ""
	if len(t.Artists) > 0 {
		artist = t.Artists[0].Name
	}
	return strings.TrimSpace(fmt.Sprintf("%s - %s", artist, t.Name))
}

// TrackInfoFromFullTrack builds TrackInfo from a Spotify track.
func TrackInfoFromFullTrack(t *spotify.FullTrack, playlist string, num int) *model.TrackInfo {
	artist := "Unknown"
	if len(t.Artists) > 0 && t.Artists[0].Name != "" {
		artist = t.Artists[0].Name
	}
	ti := &model.TrackInfo{
		Title:       t.Name,
		Artist:      artist,
		Album:       t.Album.Name,
		DurationSec: int(time.Duration(t.Duration) * time.Millisecond / time.Second),
		StreamURL:   fmt.Sprintf("https://open.spotify.com/track/%s", t.ID),
		Playlist:    playlist,
		TrackNum:    num,
		Source:      "spotify:track:" + string(t.ID),
	}
	if len(t.Album.Images) > 0 {
		ti.Thumbnail = t.Album.Images[0].URL
	}
	return ti
}
