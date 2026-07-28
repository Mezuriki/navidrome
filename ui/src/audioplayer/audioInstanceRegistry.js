// Process-wide singleton holding the external music player's underlying
// <audio> HTMLMediaElement.
//
// Why a module singleton instead of Redux? The audio element itself must NOT
// live in the Redux store: it is a non-serializable DOM object, and writing it
// to the store on every `timeupdate` event (~4 Hz) would force the whole app
// to re-render. Components that need the live playback position (e.g. the AI
// Assistant floating window's synced-lyric line highlighting) read the
// instance from here and attach their own DOM listeners.
//
// `Player.jsx` is the sole writer: it calls `setAudioInstance(audioInstance)`
// when the external player hands it the element via `getAudioInstance`. The
// element is stable across track changes (only its `src` changes), so a single
// registration is enough.

let instance = null

export const setAudioInstance = (inst) => {
  instance = inst || null
}

export const getAudioInstance = () => instance
