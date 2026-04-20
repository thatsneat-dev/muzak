// Package model defines the core domain types shared across all packages
// in the muzak application. These types represent the fundamental data
// structures for track metadata, playlists, albums, and playback state,
// decoupled from any specific adapter implementation.
package model

// TrackInfo holds metadata about the currently playing track.
type TrackInfo struct {
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Position float64 `json:"position"`
	Playing  bool
}

// QueueTrack holds metadata about an upcoming track in the queue.
type QueueTrack struct {
	Name     string  `json:"name"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Duration float64 `json:"duration"`
}

// Playlist holds metadata about a user playlist from Music.app.
type Playlist struct {
	Name         string `json:"name"`
	PersistentID string `json:"persistentID"`
	SpecialKind  string `json:"specialKind"`
	TrackCount   int    `json:"trackCount"`
	ParentID     string `json:"parentID"`
}

// IsFolder reports whether this playlist is a folder.
func (p Playlist) IsFolder() bool {
	return p.SpecialKind == "folder"
}

// Album holds metadata about a library album.
type Album struct {
	Name        string `json:"name"`
	AlbumArtist string `json:"albumArtist"`
	TrackCount  int    `json:"trackCount"`
}

// AlbumTrack holds metadata about a track within an album.
type AlbumTrack struct {
	Name         string  `json:"name"`
	PersistentID string  `json:"persistentID"`
	DiscNumber   int     `json:"discNumber"`
	TrackNumber  int     `json:"trackNumber"`
	Duration     float64 `json:"duration"`
}
