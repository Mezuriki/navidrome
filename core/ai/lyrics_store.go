package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/model"
)

// lyricsStore writes sidecar .lrc files next to a media file. The music library
// is mounted read/write in the user's deployment; writes go through plain os
// calls on the absolute path (same approach as core/image_upload.go), because
// core/storage exposes only a read-only fs.FS.
type lyricsStore struct{}

func newLyricsStore() *lyricsStore { return &lyricsStore{} }

// sidecarBase returns the media file path with its extension stripped, e.g.
// "/music/Artist/Album/01-Song.flac" -> "/music/Artist/Album/01-Song".
func sidecarBase(mf *model.MediaFile) string {
	p := mf.AbsolutePath()
	ext := filepath.Ext(p)
	return p[0 : len(p)-len(ext)]
}

// writeSidecar writes the given body to <base><suffix>, where suffix is the
// COMPLETE suffix including extension (e.g. ".lrc", ".ru.lrc", ".ai.decode.txt").
// The parent directory is created if needed (it always exists for a real media
// file, but the guard keeps the call safe).
func (s *lyricsStore) writeSidecar(mf *model.MediaFile, suffix, body string) error {
	path := sidecarBase(mf) + suffix
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// hasSidecar reports whether a sidecar file with the given suffix exists next
// to the track. suffix includes the leading dot, e.g. ".lrc" or ".ru.lrc".
func (s *lyricsStore) hasSidecar(mf *model.MediaFile, suffix string) bool {
	_, err := os.Stat(sidecarBase(mf) + suffix)
	return err == nil
}

// readSidecar reads a sidecar file with the given suffix. Returns
// (body, true, nil) on success and ("", false, nil) when the file does not
// exist. The suffix includes the leading dot, e.g. ".ai.decode.txt".
func (s *lyricsStore) readSidecar(mf *model.MediaFile, suffix string) (string, bool) {
	body, err := os.ReadFile(sidecarBase(mf) + suffix)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(body)), true
}

// ensureLangHeader guarantees a [lang:xx] tag at the very top of the LRC body.
// The Navidrome LRC parser reads it to set Lyrics.Lang (model/lyrics_lrc.go),
// which lets the loader emit a properly typed translation track.
func ensureLangHeader(body, lang string) string {
	body = strings.TrimSpace(body)
	if lang == "" {
		return body
	}
	// If a lang tag is already present, leave it untouched.
	for _, line := range strings.SplitN(body, "\n", 5) {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "[lang:") {
			return body
		}
		// Stop scanning headers once we reach a timestamp line.
		if strings.HasPrefix(l, "[0") || strings.HasPrefix(l, "[1") {
			break
		}
	}
	return fmt.Sprintf("[lang:%s]\n%s", lang, body)
}
