// Package catalog provides a client for the iTunes Search API to search
// the Apple Music catalog.
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// Song holds metadata about a song from the Apple Music catalog.
type Song struct {
	TrackID      int    `json:"trackId"`
	Name         string `json:"trackName"`
	Artist       string `json:"artistName"`
	Album        string `json:"collectionName"`
	DurationMs   int    `json:"trackTimeMillis"`
	TrackViewURL string `json:"trackViewUrl"`
	ArtworkURL   string `json:"artworkUrl100"`
}

type searchResponse struct {
	ResultCount int    `json:"resultCount"`
	Results     []Song `json:"results"`
}

// Search queries the iTunes Search API for songs matching the given query
// and returns up to 25 results from the US storefront.
func Search(ctx context.Context, query string) ([]Song, error) {
	params := url.Values{
		"media":   {"music"},
		"entity":  {"song"},
		"term":    {query},
		"country": {"us"},
		"limit":   {"25"},
	}

	reqURL := "https://itunes.apple.com/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searching catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searching catalog: unexpected status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	return result.Results, nil
}

var dimPattern = regexp.MustCompile(`/\d+x\d+bb\.`)

// ArtworkURL500 returns the artwork URL scaled to 500x500.
func ArtworkURL500(url100 string) string {
	return dimPattern.ReplaceAllString(url100, "/500x500bb.")
}

// DownloadArtwork fetches artwork from a URL and returns PNG bytes.
// JPEG images are converted to PNG for Kitty graphics protocol compatibility.
func DownloadArtwork(ctx context.Context, artworkURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artworkURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating artwork request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading artwork: unexpected status %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decoding artwork: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding artwork as PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// Ensure JPEG decoder is registered for image.Decode.
var _ = jpeg.Decode
