package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/ioutils"
)

func fromEmbedded(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if mf.Lyrics != "" {
		log.Trace(ctx, "embedded lyrics found in file", "title", mf.Title)
		return mf.StructuredLyrics()
	}

	log.Trace(ctx, "no embedded lyrics for file", "path", mf.Title)

	return nil, nil
}

func fromExternalFile(ctx context.Context, mf *model.MediaFile, suffix string) (model.LyricList, error) {
	ext := path.Ext(mf.Path)
	sidecarRelPath := mf.Path[0:len(mf.Path)-len(ext)] + suffix

	store, err := storage.For(mf.LibraryPath)
	if err != nil {
		return nil, fmt.Errorf("getting storage for library: %w", err)
	}
	fsys, err := store.FS()
	if err != nil {
		return nil, fmt.Errorf("opening library filesystem: %w", err)
	}

	f, err := fsys.Open(sidecarRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Trace(ctx, "no lyrics found at path", "path", sidecarRelPath)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	contents, err := io.ReadAll(ioutils.UTF8Reader(f))
	if err != nil {
		return nil, err
	}

	list, err := model.ParseLyrics(suffix, "xxx", contents)
	if err != nil {
		log.Error(ctx, "error parsing external lyric file", "path", sidecarRelPath, err)
		return nil, err
	}

	if len(list) == 0 {
		log.Trace(ctx, "empty lyrics from external file", "path", sidecarRelPath)
		return nil, nil
	}

	// Load any language-suffixed sidecars next to the track, e.g. "Song.ru.lrc"
	// or "Song.en.lrc". Each becomes an additional Lyrics entry so the player
	// can show original + translation side by side. The main file stays kind=main;
	// language-suffixed files are marked kind=translation.
	for _, lang := range sidecarLanguages {
		langRelPath := mf.Path[0:len(mf.Path)-len(ext)] + "." + lang + suffix
		lf, lerr := fsys.Open(langRelPath)
		if errors.Is(lerr, fs.ErrNotExist) {
			continue
		} else if lerr != nil {
			continue
		}
		lcontents, cerr := io.ReadAll(ioutils.UTF8Reader(lf))
		lf.Close()
		if cerr != nil {
			continue
		}
		langList, perr := model.ParseLyrics(suffix, lang, lcontents)
		if perr != nil {
			log.Warn(ctx, "error parsing language sidecar lyric file", "path", langRelPath, err)
			continue
		}
		for i := range langList {
			// Tag as a translation unless the file explicitly declares itself
			// otherwise. Lang is taken from the suffix (and refined by any
			// [lang:] tag inside the file during parsing).
			langList[i].Kind = model.LyricKindTranslation
			if langList[i].Lang == "" {
				langList[i].Lang = lang
			}
		}
		list = append(list, langList...)
		log.Trace(ctx, "retrieved translation lyrics from sidecar", "path", langRelPath, "lang", lang)
	}

	log.Trace(ctx, "retrieved lyrics from external file", "path", sidecarRelPath)
	return list, nil
}

// sidecarLanguages lists the language suffixes probed next to each media file,
// e.g. "Track.ru.lrc", "Track.en.lrc". Extend as needed.
var sidecarLanguages = []string{"ru", "en", "de", "fr", "es", "it", "pt", "ja", "zh", "ko"}

// fromPlugin attempts to load lyrics from a plugin with the given name.
func (l *lyricsService) fromPlugin(ctx context.Context, mf *model.MediaFile, pluginName string) (model.LyricList, error) {
	if l.pluginLoader == nil {
		log.Debug(ctx, "Invalid lyric source", "source", pluginName)
		return nil, nil
	}

	provider, ok := l.pluginLoader.LoadLyricsProvider(pluginName)
	if !ok {
		log.Warn(ctx, "Lyrics plugin not found", "plugin", pluginName)
		return nil, nil
	}

	lyricsList, err := provider.GetLyrics(ctx, mf)
	if err != nil {
		return nil, err
	}

	if len(lyricsList) > 0 {
		log.Trace(ctx, "Retrieved lyrics from plugin", "plugin", pluginName, "count", len(lyricsList))
	}
	return lyricsList, nil
}
