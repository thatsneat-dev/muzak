package model

// Song holds metadata about a song from the Apple Music catalog,
// as returned by the iTunes Search API.
type Song struct {
	TrackID      int    `json:"trackId"`
	Name         string `json:"trackName"`
	Artist       string `json:"artistName"`
	Album        string `json:"collectionName"`
	DurationMs   int    `json:"trackTimeMillis"`
	TrackViewURL string `json:"trackViewUrl"`
	ArtworkURL   string `json:"artworkUrl100"`
}
