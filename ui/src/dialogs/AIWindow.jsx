import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { Paper, Box, Tabs, Tab, Typography, IconButton, Chip, Button, CircularProgress } from '@material-ui/core'
import { MdClose as CloseIcon, MdHelpOutline as MeaningIcon, MdLyrics, MdAutorenew } from 'react-icons/md'
import { Rnd } from 'react-rnd'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate, useNotify, useGetOne } from 'react-admin'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { httpClient } from '../dataProvider'
import { setLyricLang } from '../actions'

// Height reserved at the bottom for the external music-player panel so the
// window never slides under it and the Lyrics/Meaning tabs stay fully visible.
const PLAYER_BAR_RESERVE = 96
const MIN_W = 340
const MIN_H = 320

const useStyles = makeStyles((theme) => ({
  rnd: {
    // react-rnd (via react-draggable) positions its root with an INLINE
    // `style="position: absolute"` + a CSS transform. Absolute means it scrolls
    // WITH the page and the window disappears when the user scrolls. We MUST use
    // !important to override that inline style and force `position: fixed`, so
    // the transform becomes viewport-relative and the window floats over the
    // content, always staying on screen regardless of scroll position.
    position: 'fixed !important',
    // High z-index so the window floats above all app chrome (sidebars, the
    // player bar at z-index 99) WITHOUT a backdrop — background stays interactive.
    zIndex: theme.zIndex.modal + 50,
  },
  root: {
    position: 'relative',
    width: '100%',
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    // Strong elevation + a visible border so the window is clearly separated
    // from the page content behind it. In light themes the page cards are also
    // white, so without this the window would visually merge with the cards
    // and look like the background "turned white" when the window opened.
    boxShadow: theme.shadows[24],
    border: `1px solid ${theme.palette.divider}`,
    backgroundColor: theme.palette.background.paper,
    overflow: 'hidden',
    borderRadius: theme.shape.borderRadius,
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: theme.spacing(1, 1, 1, 2),
    cursor: 'move',
    backgroundColor: theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    userSelect: 'none',
    flexShrink: 0,
  },
  headerTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    fontSize: '0.95rem',
    fontWeight: 600,
  },
  metadata: {
    padding: theme.spacing(1.5, 2),
    borderBottom: `1px solid ${theme.palette.divider}`,
    flexShrink: 0,
  },
  titleLine: {
    fontWeight: 600,
    fontSize: '1.05rem',
    lineHeight: 1.3,
  },
  artistLine: {
    color: theme.palette.text.secondary,
    fontSize: '0.9rem',
    marginTop: 2,
  },
  albumLine: {
    color: theme.palette.text.secondary,
    fontSize: '0.82rem',
    marginTop: 2,
  },
  tagsRow: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
    marginTop: theme.spacing(1),
  },
  tag: { fontSize: '0.72rem', height: 22 },
  body: {
    flex: 1,
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
    minHeight: 0, // allow flex child to shrink so the panel scrolls, not the window
  },
  tabs: { minHeight: 40, flexShrink: 0 },
  tab: { minHeight: 40, textTransform: 'none', fontSize: '0.9rem' },
  panelScroll: {
    flex: 1,
    overflowY: 'auto',
    padding: theme.spacing(2),
    // Keep room for the vertical language switch pinned inside the panel,
    // clear of the scrollbar (which sits at the far right edge).
    paddingRight: theme.spacing(7),
  },
  // Minimalist vertical language switch pinned inside the panel, to the LEFT of
  // the scrollbar so it never overlaps it.
  langRail: {
    position: 'absolute',
    top: '50%',
    right: theme.spacing(3.5),
    transform: 'translateY(-50%)',
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(0.5),
    zIndex: 2,
  },
  langBtn: {
    width: 32,
    height: 32,
    minWidth: 0,
    padding: 0,
    borderRadius: 999,
    border: `1px solid ${theme.palette.divider}`,
    background: theme.palette.background.paper,
    color: theme.palette.text.secondary,
    fontSize: '1.1rem',
    cursor: 'pointer',
    lineHeight: 1,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    '&:hover': { backgroundColor: theme.palette.action.hover },
  },
  langBtnActive: {
    borderColor: theme.palette.primary.main,
    boxShadow: `0 0 0 2px ${theme.palette.primary.main}`,
    borderColor: theme.palette.primary.main,
  },
  // Manual lyric sync controls (±0.5s), pinned to the bottom-right of the panel,
  // below the language rail, so it never overlaps the scrollbar.
  syncRail: {
    position: 'absolute',
    bottom: theme.spacing(2),
    right: theme.spacing(2),
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: theme.spacing(0.5),
    zIndex: 2,
  },
  syncBtn: {
    width: 34,
    height: 24,
    minWidth: 0,
    padding: 0,
    borderRadius: 999,
    border: `1px solid ${theme.palette.divider}`,
    background: theme.palette.background.paper,
    color: theme.palette.text.primary,
    fontSize: '0.62rem',
    fontWeight: 600,
    cursor: 'pointer',
    lineHeight: 1,
    '&:hover': { backgroundColor: theme.palette.action.hover, borderColor: theme.palette.primary.main },
  },
  syncValue: {
    fontSize: '0.6rem',
    color: theme.palette.text.secondary,
    fontWeight: 600,
    fontVariantNumeric: 'tabular-nums',
    lineHeight: 1.1,
    textAlign: 'center',
  },
  syncReset: {
    width: 18,
    height: 18,
    minWidth: 0,
    padding: 0,
    borderRadius: 999,
    border: 'none',
    background: 'transparent',
    color: theme.palette.text.disabled,
    fontSize: '0.7rem',
    cursor: 'pointer',
    lineHeight: 1,
    '&:hover': { color: theme.palette.error.main },
  },
  // --- Lyrics tab ---
  lyricLine: {
    padding: theme.spacing(0.6, 1),
    margin: theme.spacing(0.2, 0),
    borderRadius: theme.shape.borderRadius,
    fontSize: '0.95rem',
    lineHeight: 1.5,
    color: theme.palette.text.secondary,
    transition: 'background-color 0.2s, color 0.2s',
    textAlign: 'center',
  },
  lyricLineActive: {
    backgroundColor: theme.palette.action.selected,
    color: theme.palette.text.primary,
    fontWeight: 600,
  },
  lyricEmpty: {
    textAlign: 'center',
    padding: theme.spacing(4, 2),
    color: theme.palette.text.secondary,
  },
  // --- Meaning tab ---
  markdown: {
    '& h1, & h2, & h3, & h4': { marginTop: theme.spacing(2), marginBottom: theme.spacing(1), lineHeight: 1.3 },
    '& h1': { fontSize: '1.3rem' },
    '& h2': { fontSize: '1.15rem' },
    '& h3': { fontSize: '1.05rem' },
    '& p': { margin: theme.spacing(1, 0), lineHeight: 1.6 },
    '& ul, & ol': { paddingLeft: theme.spacing(3), margin: theme.spacing(1, 0) },
    '& li': { margin: theme.spacing(0.3, 0), lineHeight: 1.5 },
    '& blockquote': {
      borderLeft: `3px solid ${theme.palette.primary.main}`,
      paddingLeft: theme.spacing(1.5),
      margin: theme.spacing(1, 0),
      color: theme.palette.text.secondary,
    },
    '& code': { backgroundColor: theme.palette.action.hover, padding: '2px 5px', borderRadius: 4, fontSize: '0.85em' },
    '& pre': { backgroundColor: theme.palette.action.hover, padding: theme.spacing(1), borderRadius: theme.shape.borderRadius, overflowX: 'auto' },
    '& strong': { fontWeight: 600 },
    '& em': { fontStyle: 'italic' },
    '& hr': { border: 'none', borderTop: `1px solid ${theme.palette.divider}`, margin: theme.spacing(2, 0) },
  },
  generateBtn: { marginTop: theme.spacing(1.5) },
  statusBox: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    justifyContent: 'center',
    padding: theme.spacing(3, 2),
    color: theme.palette.text.secondary,
  },
  resizeHandle: {
    position: 'absolute',
    right: 2,
    bottom: 2,
    width: 16,
    height: 16,
    cursor: 'nwse-resize',
    color: theme.palette.text.disabled,
    opacity: 0.5,
    '&:hover': { opacity: 1 },
  },
}))

const TabPanel = ({ children, value, index }) =>
  value === index ? <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>{children}</div> : null

TabPanel.propTypes = {
  children: PropTypes.node,
  value: PropTypes.number.isRequired,
  index: PropTypes.number.isRequired,
}

const pickLyricTrack = (list, lang) => {
  if (!Array.isArray(list) || list.length === 0) return null
  if (lang === 'original' || !lang) {
    return list.find((l) => l && (!l.kind || l.kind === 'main') && l.synced) || list.find((l) => l && l.synced) || null
  }
  return (
    list.find((l) => l && l.kind === 'translation' && l.lang === lang && l.synced) ||
    list.find((l) => l && l.lang === lang && l.synced) ||
    null
  )
}

const hasTranslation = (list, lang) =>
  Array.isArray(list) && list.some((l) => l && l.kind === 'translation' && l.lang === lang && l.synced)

// ---- Lyrics tab ----
const LyricsTab = ({ record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dispatch = useDispatch()
  const lyricLang = useSelector((state) => state.player?.lyricLang || 'original')
  const notify = useNotify()
  const [currentTimeMs, setCurrentTimeMs] = useState(0)
  const [lyricsStatus, setLyricsStatus] = useState(null)
  const [fetching, setFetching] = useState(false)
  const activeLineRef = useRef(null)
  const scrollContainerRef = useRef(null)

  const { data: fresh } = useGetOne('song', record?.id, { enabled: !!record?.id })
  const lyricsSource = fresh?.lyrics || record?.lyrics
  const structured = useMemo(() => {
    if (!lyricsSource) return null
    try {
      const parsed = JSON.parse(lyricsSource)
      return Array.isArray(parsed) ? parsed : null
    } catch {
      return null
    }
  }, [lyricsSource])

  const track = useMemo(() => pickLyricTrack(structured, lyricLang), [structured, lyricLang])
  const lines = useMemo(() => {
    if (!track || !Array.isArray(track.line)) return []
    return track.line.filter((l) => l && l.value != null && l.value !== '')
  }, [track])
  const ruAvailable = useMemo(() => hasTranslation(structured, 'ru'), [structured])

  // Per-track lyric timing offset in ms, persisted in localStorage so the user's
  // manual sync adjustment survives reopening the window. A positive offset means
  // the lyric line is shown EARLIER (lyrics lag behind playback → shift forward).
  const offsetKey = record?.id ? `ai_lyric_offset_${record.id}` : null
  const [lyricOffsetMs, setLyricOffsetMs] = useState(() => {
    if (!offsetKey) return 0
    const raw = typeof localStorage !== 'undefined' ? localStorage.getItem(offsetKey) : null
    const n = raw ? parseInt(raw, 10) : 0
    return Number.isFinite(n) ? n : 0
  })
  const adjustOffset = useCallback(
    (deltaMs) => {
      setLyricOffsetMs((prev) => {
        const next = prev + deltaMs
        if (offsetKey) localStorage.setItem(offsetKey, String(next))
        return next
      })
    },
    [offsetKey],
  )

  const activeIndex = useMemo(() => {
    if (lines.length === 0) return -1
    // Apply the manual offset: subtract it from the line start when comparing,
    // so a +500ms offset makes each line active 500ms earlier.
    const adjusted = currentTimeMs - lyricOffsetMs
    let idx = -1
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].start != null && lines[i].start <= adjusted) idx = i
    }
    return idx
  }, [lines, currentTimeMs, lyricOffsetMs])

  // Subscribe to the <audio> element's timeupdate for live position. We read
  // the element directly from the DOM (the external player renders exactly one)
  // instead of a module singleton, which bundler code-splitting can split into
  // separate copies and break the read.
  useEffect(() => {
    const audio = document.querySelector('audio')
    if (!audio) return
    const onTime = () => setCurrentTimeMs(Math.floor((audio.currentTime || 0) * 1000))
    onTime()
    audio.addEventListener('timeupdate', onTime)
    return () => audio.removeEventListener('timeupdate', onTime)
  }, [record?.id])

  useEffect(() => {
    if (activeLineRef.current && scrollContainerRef.current) {
      activeLineRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, [activeIndex])

  useEffect(() => {
    if (!record || !record.id) return
    if (!lyricsStatus || (lyricsStatus.status !== 'queued' && lyricsStatus.status !== 'running')) return
    const timer = setInterval(() => {
      httpClient(`/api/ai/lyrics/status?mediaFileId=${encodeURIComponent(record.id)}`)
        .then(({ json }) => setLyricsStatus(json))
        .catch(() => {})
    }, 2000)
    return () => clearInterval(timer)
  }, [record, lyricsStatus])

  // Reset transient state when the track changes so stale content never lingers.
  useEffect(() => {
    setLyricsStatus(null)
    setFetching(false)
    setCurrentTimeMs(0)
  }, [record?.id])

  const handleGenerate = async () => {
    if (!record || !record.id) return
    setFetching(true)
    try {
      await httpClient('/api/ai/lyrics/fetch', {
        method: 'POST',
        body: JSON.stringify({ mediaFileId: record.id }),
      })
      setLyricsStatus({ status: 'queued', step: 'lyrics' })
      notify(translate('ai.lyrics.queued'), { type: 'info' })
    } catch (error) {
      notify(translate('ai.lyrics.error') + ': ' + (error?.message || ''), { type: 'error' })
    } finally {
      setFetching(false)
    }
  }

  const busy = fetching || (lyricsStatus && (lyricsStatus.status === 'queued' || lyricsStatus.status === 'running'))

  if (!track || lines.length === 0) {
    return (
      <div className={classes.panelScroll} ref={scrollContainerRef} style={{ position: 'relative' }}>
        {busy ? (
          <div className={classes.statusBox}>
            <CircularProgress size={18} />
            <Typography variant="body2">
              {lyricsStatus?.step === 'translation'
                ? translate('ai.lyrics.statusTranslation')
                : translate('ai.lyrics.statusLyrics')}
              …
            </Typography>
          </div>
        ) : (
          <div className={classes.lyricEmpty}>
            <Typography variant="body2" paragraph>
              {translate('ai.window.noLyrics')}
            </Typography>
            <Button className={classes.generateBtn} variant="contained" color="primary" size="small" startIcon={<MdLyrics />} onClick={handleGenerate}>
              {translate('ai.window.generateLyrics')}
            </Button>
          </div>
        )}
      </div>
    )
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', position: 'relative' }}>
      <div className={classes.panelScroll} ref={scrollContainerRef}>
        {lines.map((l, i) => (
          <div
            key={i}
            ref={i === activeIndex ? activeLineRef : null}
            className={`${classes.lyricLine} ${i === activeIndex ? classes.lyricLineActive : ''}`}
          >
            {l.value}
          </div>
        ))}
        {lyricLang === 'ru' && !ruAvailable && (
          <Typography variant="body2" className={classes.lyricEmpty}>
            {translate('ai.window.noRu')}
          </Typography>
        )}
        <div className={classes.langRail}>
          <button
            type="button"
            title={translate('ai.window.russian')}
            className={`${classes.langBtn} ${lyricLang === 'ru' ? classes.langBtnActive : ''}`}
            onClick={() => dispatch(setLyricLang('ru'))}
          >
            🇷🇺
          </button>
          <button
            type="button"
            title={translate('ai.window.original')}
            className={`${classes.langBtn} ${lyricLang === 'original' ? classes.langBtnActive : ''}`}
            onClick={() => dispatch(setLyricLang('original'))}
          >
            🇺🇳
          </button>
        </div>
        {lines.some((l) => l.start != null) && (
          <div className={classes.syncRail}>
            <button
              type="button"
              title={translate('ai.window.syncBack')}
              className={classes.syncBtn}
              onClick={() => adjustOffset(-500)}
            >
              −0.5с
            </button>
            <span className={classes.syncValue} title={translate('ai.window.syncHint')}>
              {lyricOffsetMs > 0 ? `+${(lyricOffsetMs / 1000).toFixed(1)}с` : `${(lyricOffsetMs / 1000).toFixed(1)}с`}
            </span>
            <button
              type="button"
              title={translate('ai.window.syncForward')}
              className={classes.syncBtn}
              onClick={() => adjustOffset(500)}
            >
              +0.5с
            </button>
            {lyricOffsetMs !== 0 && (
              <button
                type="button"
                title={translate('ai.window.syncReset')}
                className={classes.syncReset}
                onClick={() => adjustOffset(-lyricOffsetMs)}
              >
                ✕
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

LyricsTab.propTypes = { record: PropTypes.object }

// ---- Meaning tab ----
const MeaningTab = ({ record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const [text, setText] = useState('')
  const [found, setFound] = useState(false)
  const [loading, setLoading] = useState(true)
  const [generating, setGenerating] = useState(false)

  const loadStored = useCallback(() => {
    if (!record || !record.id) return
    setLoading(true)
    httpClient(`/api/ai/decode?mediaFileId=${encodeURIComponent(record.id)}`)
      .then(({ json }) => {
        if (json && json.found && json.text) {
          setText(json.text)
          setFound(true)
        } else {
          setText('')
          setFound(false)
        }
      })
      .catch(() => {
        setText('')
        setFound(false)
      })
      .finally(() => setLoading(false))
  }, [record])

  useEffect(() => {
    setText('')
    setFound(false)
    loadStored()
  }, [loadStored])

  const html = useMemo(() => {
    if (!text) return ''
    try {
      return DOMPurify.sanitize(marked.parse(text, { breaks: true, gfm: true }))
    } catch {
      return ''
    }
  }, [text])

  const handleGenerate = async () => {
    if (!record || !record.id) return
    setGenerating(true)
    try {
      const { json } = await httpClient('/api/ai/decode', {
        method: 'POST',
        body: JSON.stringify({
          title: record.title,
          artist: record.artist,
          album: record.album,
          lyrics: record.lyrics || '',
          mediaFileId: record.id,
        }),
      })
      if (json && json.text) {
        setText(json.text)
        setFound(true)
        notify(translate('ai.success.decode'), { type: 'success' })
      }
    } catch (error) {
      notify(translate('ai.error.decode') + ': ' + (error?.message || ''), { type: 'error' })
    } finally {
      setGenerating(false)
    }
  }

  if (loading) {
    return (
      <div className={classes.statusBox}>
        <CircularProgress size={18} />
      </div>
    )
  }

  if (!found || !text) {
    return (
      <div className={classes.panelScroll}>
        <div className={classes.lyricEmpty}>
          <Typography variant="body2" paragraph>
            {translate('ai.window.noMeaning')}
          </Typography>
          <Button
            className={classes.generateBtn}
            variant="contained"
            color="primary"
            size="small"
            startIcon={generating ? <CircularProgress size={14} color="inherit" /> : <MdAutorenew />}
            onClick={handleGenerate}
            disabled={generating}
          >
            {translate('ai.window.generateMeaning')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className={classes.panelScroll}>
      <div className={classes.markdown} dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  )
}

MeaningTab.propTypes = { record: PropTypes.object }

// ---- The resizable, draggable floating window ----
const AIWindow = ({ open, onClose, record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [tab, setTab] = useState(0)

  const initial = useMemo(() => {
    const vw = typeof window !== 'undefined' ? window.innerWidth : 1280
    const vh = typeof window !== 'undefined' ? window.innerHeight : 800
    const w = Math.min(480, vw - 32)
    const h = Math.min(vh - PLAYER_BAR_RESERVE - 24, 560)
    return {
      x: Math.max(16, Math.floor(vw / 2 - w / 2)),
      y: 24,
      width: w,
      height: h,
    }
  }, [])

  const [size, setSize] = useState({ width: initial.width, height: initial.height })
  const [pos, setPos] = useState({ x: initial.x, y: initial.y })
  // Track whether the window was open last render. When it transitions from
  // closed→open, we clamp the position SYNCHRONOUSLY (before Rnd renders) so
  // the window never flashes off-screen. useEffect runs too late for this.
  const wasOpenRef = useRef(false)
  if (open && !wasOpenRef.current) {
    // Window just opened — clamp position to current viewport immediately.
    const vw = typeof window !== 'undefined' ? window.innerWidth : 1280
    const vh = typeof window !== 'undefined' ? window.innerHeight : 800
    const w = Math.min(size.width, vw - 32)
    const maxX = Math.max(16, vw - w - 16)
    const newY = Math.max(16, Math.min(vh - size.height - PLAYER_BAR_RESERVE - 8, 24))
    // Set synchronously so Rnd renders with the correct position on first paint.
    pos.x = Math.min(pos.x, maxX)
    pos.y = newY
  }
  wasOpenRef.current = open

  useEffect(() => {
    if (open) setTab(0)
  }, [open, record?.id])

  // Keep the window fully inside the viewport whenever the browser window is
  // resized or zoomed, so it never ends up off-screen (the user reported the
  // window disappearing from view). Re-clamp the stored position/size.
  useEffect(() => {
    if (!open) return
    const onResize = () => {
      const vw = window.innerWidth
      const vh = window.innerHeight
      setSize((s) => ({
        width: Math.min(s.width, Math.max(MIN_W, vw - 16)),
        height: Math.min(s.height, Math.max(MIN_H, vh - PLAYER_BAR_RESERVE - 8)),
      }))
      setPos((p) => {
        const w = Math.min(size.width, vw - 16)
        const h = Math.min(size.height, vh - PLAYER_BAR_RESERVE - 8)
        return {
          x: Math.max(0, Math.min(p.x, Math.max(0, vw - w - 8))),
          y: Math.max(0, Math.min(p.y, Math.max(0, vh - h - PLAYER_BAR_RESERVE))),
        }
      })
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [open, size.width, size.height])

  // Clamp the dropped position so the window stays fully on-screen (never let
  // the header be dragged off the top, and never let the bottom slide under /
  // past the player bar). This is what makes the window "always visible".
  const handleDragStop = (_, d) => {
    const vw = window.innerWidth
    const vh = window.innerHeight
    const maxX = Math.max(0, vw - size.width - 8)
    // Keep the whole window above the player bar and the header grabbable.
    const maxY = Math.max(0, vh - size.height - PLAYER_BAR_RESERVE)
    setPos({
      x: Math.max(0, Math.min(d.x, maxX)),
      y: Math.max(0, Math.min(d.y, maxY)),
    })
  }

  if (!open || !record) return null

  const vh = typeof window !== 'undefined' ? window.innerHeight : 800
  const vw = typeof window !== 'undefined' ? window.innerWidth : 1280

  return (
    <Rnd
      className={classes.rnd}
      size={{ width: size.width, height: size.height }}
      position={{ x: pos.x, y: pos.y }}
      onDrag={(_, d) => {
        // Live clamp DURING drag so the window can never leave the viewport.
        const vw = window.innerWidth
        const vh = window.innerHeight
        const maxX = Math.max(0, vw - size.width - 4)
        const maxY = Math.max(0, vh - size.height - PLAYER_BAR_RESERVE)
        if (d.x < 0 || d.y < 0 || d.x > maxX || d.y > maxY) {
          setPos({
            x: Math.max(0, Math.min(d.x, maxX)),
            y: Math.max(0, Math.min(d.y, maxY)),
          })
        }
      }}
      onDragStop={handleDragStop}
      onResizeStop={(_, __, ref) => {
        const newW = ref.offsetWidth
        const newH = ref.offsetHeight
        setSize({ width: newW, height: newH })
        // If enlarging pushed the window off the right/bottom edge, pull it back in.
        setPos((p) => ({
          x: Math.min(p.x, Math.max(0, vw - newW - 8)),
          y: Math.min(p.y, Math.max(0, vh - newH - PLAYER_BAR_RESERVE)),
        }))
      }}
      bounds={null}
      dragHandleClassName="ai-window-header"
      minWidth={MIN_W}
      minHeight={MIN_H}
      maxWidth={vw - 16}
      maxHeight={vh - PLAYER_BAR_RESERVE - 8}
      enableResizing={{ bottomRight: true, right: true, bottom: true }}
    >
      <Paper className={classes.root} elevation={8}>
        <Box className={`${classes.header} ai-window-header`}>
          <span className={classes.headerTitle}>
            <MdLyrics fontSize="small" />
            {translate('ai.window.title')}
          </span>
          <IconButton size="small" onClick={onClose} style={{ color: 'inherit' }} aria-label="close">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Box>

        <Box className={classes.metadata}>
          <Typography className={classes.titleLine}>{record.title || '—'}</Typography>
          {record.artist && <Typography className={classes.artistLine}>{record.artist}</Typography>}
          {(record.album || record.year) && (
            <Typography className={classes.albumLine}>
              {[record.album, record.year].filter(Boolean).join(' · ')}
            </Typography>
          )}
          {record.genre && (
            <Box className={classes.tagsRow}>
              <Chip size="small" label={record.genre} className={classes.tag} />
            </Box>
          )}
        </Box>

        <Tabs value={tab} onChange={(e, v) => setTab(v)} className={classes.tabs} variant="fullWidth">
          <Tab className={classes.tab} icon={<MdLyrics fontSize="small" />} label={translate('ai.window.lyricsTab')} />
          <Tab className={classes.tab} icon={<MeaningIcon fontSize="small" />} label={translate('ai.window.meaningTab')} />
        </Tabs>

        <Box className={classes.body}>
          <TabPanel value={tab} index={0}>
            <LyricsTab record={record} />
          </TabPanel>
          <TabPanel value={tab} index={1}>
            <MeaningTab record={record} />
          </TabPanel>
        </Box>

        {/* resize affordance */}
        <svg className={classes.resizeHandle} viewBox="0 0 16 16" aria-hidden="true">
          <path d="M16 16 L16 8 L8 16 Z" fill="currentColor" />
        </svg>
      </Paper>
    </Rnd>
  )
}

AIWindow.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  record: PropTypes.object,
}

export default AIWindow
