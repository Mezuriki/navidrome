package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// lrclibBaseURL is the LRCLIB API root used to look up synced lyrics.
const lrclibBaseURL = "https://lrclib.net/api"

// lrclibClient queries the LRCLIB service for synced (timestamped) lyrics.
// LRCLIB returns plain-text LRC bodies with [mm:ss.xx] timing, which is exactly
// what the Navidrome player consumes.
type lrclibClient struct {
	hc      *http.Client
	baseURL string
}

func newLrclibClient() *lrclibClient {
	return &lrclibClient{
		hc:      &http.Client{Timeout: 20 * time.Second},
		baseURL: lrclibBaseURL,
	}
}

// lrclibTrack mirrors the subset of the LRCLIB /api/get response we need.
type lrclibTrack struct {
	TrackName  string `json:"trackName"`
	ArtistName string `json:"artistName"`
	AlbumName  string `json:"albumName"`
	Duration   *float64 `json:"duration"`
	PlainLyric string `json:"plainLyrics"`
	SyncedLyric string `json:"syncedLyrics"`
	Lang       string `json:"lang"`
}

// getSynced queries LRCLIB for a single best-match synced track. Returns
// (nil, nil) when there is no match so callers can fall back to Gemini.
func (c *lrclibClient) getSynced(ctx context.Context, artist, title, album string, durationSec float64) (*lrclibTrack, error) {
	q := url.Values{}
	q.Set("artist_name", artist)
	q.Set("track_name", title)
	if album != "" {
		q.Set("album_name", album)
	}
	if durationSec > 0 {
		q.Set("duration", strconv.Itoa(int(durationSec)))
	}

	endpoint := c.baseURL + "/get?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	// A non-empty User-Agent avoids some naive bot filters.
	req.Header.Set("User-Agent", "navidrome-ai/1.0 (+https://navidrome.org)")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 404 means "no match" on the exact endpoint — fall back to fuzzy search.
	if resp.StatusCode == http.StatusNotFound {
		return c.searchSynced(ctx, artist, title)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib returned status %d: %s", resp.StatusCode, truncateString(string(body), 200))
	}

	var t lrclibTrack
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("failed to parse lrclib response: %w", err)
	}
	if t.SyncedLyric == "" {
		// Only synced lyrics are useful for our flow; plain-only is a miss on
		// the exact endpoint — try the fuzzy search before giving up.
		return c.searchSynced(ctx, artist, title)
	}
	return &t, nil
}

// searchSynced is the fuzzy fallback when the exact /api/get misses (e.g. the
// album/duration don't match LRCLIB's record). It calls /api/search with the
// artist + track name and returns the first hit that has synced lyrics.
func (c *lrclibClient) searchSynced(ctx context.Context, artist, title string) (*lrclibTrack, error) {
	q := url.Values{}
	q.Set("artist_name", artist)
	q.Set("track_name", title)

	endpoint := c.baseURL + "/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "navidrome-ai/1.0 (+https://navidrome.org)")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib search returned status %d: %s", resp.StatusCode, truncateString(string(body), 200))
	}

	var hits []lrclibTrack
	if err := json.Unmarshal(body, &hits); err != nil {
		return nil, fmt.Errorf("failed to parse lrclib search response: %w", err)
	}
	for i := range hits {
		if hits[i].SyncedLyric != "" {
			return &hits[i], nil
		}
	}
	return nil, nil
}
