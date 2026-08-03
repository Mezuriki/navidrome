import { v4 as uuidv4 } from 'uuid'
import subsonic from '../subsonic'
import { decisionService } from '../transcode'
import {
  PLAYER_ADD_TRACKS,
  PLAYER_CLEAR_QUEUE,
  PLAYER_CURRENT,
  PLAYER_PLAY_NEXT,
  PLAYER_PLAY_TRACKS,
  PLAYER_SET_TRACK,
  PLAYER_SET_VOLUME,
  PLAYER_SYNC_QUEUE,
  PLAYER_SET_MODE,
  PLAYER_REFRESH_QUEUE,
  PLAYER_SET_LYRIC_LANG,
} from '../actions'
import config from '../config'

const initialState = {
  queue: [],
  current: {},
  clear: false,
  volume: config.defaultUIVolume / 100,
  savedPlayIndex: 0,
  // Which language track to show in the player's lyric panel: 'original' (the
  // main synced lyric) or a language code like 'ru' (a synced translation).
  lyricLang: 'original',
}

const pad = (value) => {
  const str = value.toString()
  if (str.length === 1) {
    return `0${str}`
  } else {
    return str
  }
}

const makeMusicSrc = (trackId) =>
  decisionService.getProfile()
    ? () =>
        decisionService
          .resolveStreamUrl(trackId)
          .catch(() => subsonic.streamUrl(trackId))
    : subsonic.streamUrl(trackId)

// Format a millisecond timestamp as [mm:ss.cc] (centiseconds), matching the
// LRC format the external player's lyric parser expects.
const formatLrcTime = (ms) => {
  let time = Math.floor(ms / 10)
  const cs = time % 100
  time = Math.floor(time / 100)
  const sec = time % 60
  const min = Math.floor(time / 60) % 60
  return `[${pad(min)}:${pad(sec)}.${pad(cs)}]`
}

// Build an LRC text string from the structured lyrics JSON stored in
// media_file.lyrics, selecting the entry to show by language. `lang` is either
// 'original' (the main-kind track) or a language code like 'ru' (a translation
// track). Falls back to original if the requested language isn't present.
const lyricsToLrc = (rawLyrics, lang) => {
  if (!rawLyrics) return ''
  let structured
  try {
    structured = JSON.parse(rawLyrics)
  } catch {
    return ''
  }
  if (!Array.isArray(structured) || structured.length === 0) return ''

  let pick
  if (lang === 'original' || lang == null) {
    pick = structured.find((l) => !l.kind || l.kind === 'main') || structured[0]
  } else {
    // Prefer a translation track matching the requested language; if none,
    // fall back to the main track so the panel is never empty.
    pick =
      structured.find((l) => l.lang === lang) ||
      structured.find((l) => l.kind === 'translation' && l.lang === lang) ||
      structured.find((l) => !l.kind || l.kind === 'main') ||
      structured[0]
  }
  if (!pick || !pick.synced || !Array.isArray(pick.line)) return ''

  return pick.line
    .filter((l) => l.start != null)
    .map((l) => `${formatLrcTime(l.start)} ${l.value}`)
    .join('\n')
}

const mapToAudioLists = (item, lang = 'original') => {
  // If item comes from a playlist, trackId is mediaFileId
  const trackId = item.mediaFileId || item.id

  if (item.isRadio) {
    return {
      trackId,
      uuid: uuidv4(),
      name: item.name,
      song: item,
      musicSrc: item.streamUrl,
      cover: item.cover,
      isRadio: true,
    }
  }

  return {
    trackId,
    uuid: uuidv4(),
    song: item,
    name: item.title,
    lyric: lyricsToLrc(item.lyrics, lang),
    singer: item.artist,
    duration: item.duration,
    musicSrc: makeMusicSrc(trackId),
    cover: subsonic.getCoverArtUrl(
      {
        id: trackId,
        updatedAt: item.updatedAt,
        album: item.album,
      },
      300,
    ),
  }
}

const reduceClearQueue = () => ({ ...initialState, clear: true })

const reducePlayTracks = (state, { data, id }) => {
  let playIndex = 0
  const queue = Object.keys(data).map((key, idx) => {
    if (key === id) {
      playIndex = idx
    }
    return mapToAudioLists(data[key], state.lyricLang)
  })
  return {
    ...state,
    queue,
    playIndex,
    clear: true,
  }
}

const reduceSetTrack = (state, { data }) => {
  return {
    ...state,
    queue: [mapToAudioLists(data, state.lyricLang)],
    playIndex: 0,
    clear: true,
  }
}

const reduceAddTracks = (state, { data }) => {
  const queue = state.queue
  Object.keys(data).forEach((id) => {
    queue.push(mapToAudioLists(data[id], state.lyricLang))
  })
  return { ...state, queue, clear: false }
}

const reducePlayNext = (state, { data }) => {
  const newTracks = Object.keys(data).map((id) =>
    mapToAudioLists(data[id], state.lyricLang),
  )
  const newQueue = []
  const current = state.current || {}
  let foundPos = false
  state.queue.forEach((item) => {
    newQueue.push(item)
    if (item.uuid === current.uuid) {
      foundPos = true
      newQueue.push(...newTracks)
    }
  })
  if (!foundPos) {
    newQueue.push(...newTracks)
  }

  return {
    ...state,
    queue: newQueue,
    clear: true,
  }
}

const reduceSetVolume = (state, { data: { volume } }) => {
  return {
    ...state,
    volume,
  }
}

const reduceSyncQueue = (state, { data: { audioInfo, audioLists } }) => {
  // Keep clear and playIndex alive when there is a pending track switch.
  // A switch is pending when playIndex is set AND either:
  //   - playIndex differs from savedPlayIndex, OR
  //   - clear is true (a new queue was loaded, e.g. after clearQueue + playTracks)
  // The clear check handles the edge case where both playIndex and
  // savedPlayIndex are 0 (close player then play a new album from track 1).
  const hasPendingSwitch =
    state.playIndex != null &&
    (state.clear || state.playIndex !== state.savedPlayIndex)
  return {
    ...state,
    queue: audioLists,
    clear: hasPendingSwitch ? state.clear : false,
    playIndex: hasPendingSwitch ? state.playIndex : undefined,
  }
}

const reduceCurrent = (state, { data }) => {
  const current = data.ended ? {} : data
  const savedPlayIndex = state.queue.findIndex(
    (item) => item.uuid === current.uuid,
  )
  // When a track selection is pending (playIndex is set), keep it alive
  // until the music player confirms it actually switched to the requested
  // track. Without this, a premature onAudioPlay callback for the
  // still-playing old track would overwrite the pending selection.
  const pending = state.playIndex != null && savedPlayIndex !== state.playIndex
  return {
    ...state,
    current,
    playIndex: pending ? state.playIndex : undefined,
    clear: pending ? state.clear : false,
    savedPlayIndex: pending ? state.savedPlayIndex : savedPlayIndex,
    volume: data.volume,
  }
}

const reduceMode = (state, { data: { mode } }) => {
  return {
    ...state,
    mode,
  }
}

export const playerReducer = (previousState = initialState, payload) => {
  const { type } = payload
  switch (type) {
    case PLAYER_CLEAR_QUEUE:
      return reduceClearQueue()
    case PLAYER_PLAY_TRACKS:
      return reducePlayTracks(previousState, payload)
    case PLAYER_SET_TRACK:
      return reduceSetTrack(previousState, payload)
    case PLAYER_ADD_TRACKS:
      return reduceAddTracks(previousState, payload)
    case PLAYER_PLAY_NEXT:
      return reducePlayNext(previousState, payload)
    case PLAYER_SET_VOLUME:
      return reduceSetVolume(previousState, payload)
    case PLAYER_SYNC_QUEUE:
      return reduceSyncQueue(previousState, payload)
    case PLAYER_CURRENT:
      return reduceCurrent(previousState, payload)
    case PLAYER_SET_MODE:
      return reduceMode(previousState, payload)
    case PLAYER_SET_LYRIC_LANG: {
      const lang = payload.lang || 'original'
      // 'both' is an AI-window-only mode; the external player shows original.
      const lrcLang = lang === 'both' ? 'original' : lang
      // Rebuild the lyric text of every queued track for the new language
      // WITHOUT setting `clear` — the external player reloads the current
      // item's lyric from the updated audioLists without restarting playback.
      return {
        ...previousState,
        lyricLang: lang,
        queue: previousState.queue.map((item) =>
          item.isRadio
            ? item
            : { ...item, lyric: lyricsToLrc(item.song?.lyrics, lrcLang) },
        ),
      }
    }
    case PLAYER_REFRESH_QUEUE: {
      const resolvedUrls = payload.data || {}
      return {
        ...previousState,
        queue: previousState.queue.map((item) => ({
          ...item,
          musicSrc: item.isRadio
            ? item.musicSrc
            : resolvedUrls[item.trackId] || subsonic.streamUrl(item.trackId),
        })),
        clear: true,
        autoPlay: false,
        playIndex:
          previousState.savedPlayIndex >= 0 ? previousState.savedPlayIndex : 0,
      }
    }
    default:
      return previousState
  }
}
