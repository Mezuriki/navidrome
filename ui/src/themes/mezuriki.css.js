// Mezuriki theme — external music-player CSS overrides.
// Targets the navidrome-music-player (react-jinke-music-player fork) classes:
// accent green slider/scrollbar, glassmorphism bottom panel.

const stylesheet = `
/* Accent green for interactive player elements */
.react-jinke-music-player-main svg:active,
.react-jinke-music-player-main svg:hover {
  color: #1ed760;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle,
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
  background-color: #1db954;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
  box-shadow: 0 0 2px #1db954;
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
  background-color: #1db954;
}

/* Currently-playing item in the player + queue panel */
.react-jinke-music-player-main .audio-item.playing svg {
  color: #1db954;
}
.react-jinke-music-player-main .audio-item.playing .player-singer {
  color: #1db954 !important;
}
.audio-lists-panel-content .audio-item.playing,
.audio-lists-panel-content .audio-item.playing svg {
  color: #1db954;
}
.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg,
.audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
  color: #1db954;
}

/* Glassmorphism bottom panel: translucent dark surface + strong blur so the
   album artwork / page content shows through. */
.react-jinke-music-player-main .music-player-panel {
  background-color: rgba(18, 18, 18, 0.72) !important;
  backdrop-filter: blur(24px) saturate(1.4);
  -webkit-backdrop-filter: blur(24px) saturate(1.4);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

/* Keep the mobile/full panel readable on the dark surface */
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-rail {
  background-color: rgba(255, 255, 255, 0.12);
}
.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar .rc-slider {
  background-color: transparent;
}

/* Queue panel surface — frosted glass */
.audio-lists-panel {
  background-color: rgba(18, 18, 18, 0.82) !important;
  backdrop-filter: blur(24px) saturate(1.4);
  -webkit-backdrop-filter: blur(24px) saturate(1.4);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

/* Lyric panel — frosted glass */
.react-jinke-music-player-main .music-player-lyric {
  background-color: rgba(18, 18, 18, 0.6) !important;
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 8px;
  padding: 6px 12px;
}

/* Lyric toggle button + play/pause hover accent */
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:after {
  background-color: #1db954;
}
`

export default stylesheet
