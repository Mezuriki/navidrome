import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { Paper, Box, Tabs, Tab, Typography, IconButton, Chip, Button, CircularProgress } from '@material-ui/core'
import { MdClose as CloseIcon, MdHelpOutline as MeaningIcon, MdLyrics, MdAutorenew } from 'react-icons/md'
import Draggable from 'react-draggable'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate, useNotify, useGetOne } from 'react-admin'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { httpClient } from '../dataProvider'
import { getAudioInstance } from '../audioplayer/audioInstanceRegistry'
import { setLyricLang } from '../actions'

const WINDOW_WIDTH = 480

const useStyles = makeStyles((theme) => ({
  root: {
    position: 'fixed',
    zIndex: theme.zIndex.modal + 1,
    width: WINDOW_WIDTH,
    maxWidth: '92vw',
    maxHeight: '85vh',
    display: 'flex',
    flexDirection: 'column',
    boxShadow: theme.shadows[8],
    // Let the outer Draggable control positioning (translate), but keep a sane
    // default placement via the initial left/top set inline by the component.
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
  tag: {
    fontSize: '0.72rem',
    height: 22,
  },
  body: {
    flex: 1,
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
    minHeight: 320,
  },
  tabs: {
    minHeight: 40,
  },
  tab: {
    minHeight: 40,
    textTransform: 'none',
    fontSize: '0.9rem',
  },
  panelScroll: {
    flex: 1,
    overflowY: 'auto',
    padding: theme.spacing(2),
  },
  // --- Lyrics tab ---
  lyricsLangSwitch: {
    display: 'flex',
    justifyContent: 'center',
    marginBottom: theme.spacing(1.5),
    gap: theme.spacing(0.5),
  },
  langBtn: {
    padding: theme.spacing(0.5, 1.5),
    fontSize: '0.8rem',
    borderRadius: 999,
    border: `1px solid ${theme.palette.divider}`,
    background: 'transparent',
    color: theme.palette.text.secondary,
    cursor: 'pointer',
    '&:hover': { backgroundColor: theme.palette.action.hover },
  },
  langBtnActive: {
    backgroundColor: theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    borderColor: theme.palette.primary.main,
    '&:hover': { backgroundColor: theme.palette.primary.dark },
  },
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
    transform: 'scale(1.02)',
  },
  lyricEmpty: {
    textAlign: 'center',
    padding: theme.spacing(4, 2),
    color: theme.palette.text.secondary,
  },
  // --- Meaning tab ---
  markdown: {
    '& h1, & h2, & h3, & h4': {
      marginTop: theme.spacing(2),
      marginBottom: theme.spacing(1),
      lineHeight: 1.3,
    },
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
    '& code': {
      backgroundColor: theme.palette.action.hover,
      padding: '2px 5px',
      borderRadius: 4,
      fontSize: '0.85em',
    },
    '& pre': {
      backgroundColor: theme.palette.action.hover,
      padding: theme.spacing(1),
      borderRadius: theme.shape.borderRadius,
      overflowX: 'auto',
    },
    '& strong': { fontWeight: 600 },
    '& em': { fontStyle: 'italic' },
    '& hr': { border: 'none', borderTop: `1px solid ${theme.palette.divider}`, margin: theme.spacing(2, 0) },
  },
  generateBtn: {
    marginTop: theme.spacing(1.5),
  },
  statusBox: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    justifyContent: 'center',
    padding: theme.spacing(3, 2),
    color: theme.palette.text.secondary,
  },
}))

const TabPanel = ({ children, value, index }) =>
  value === index ? <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>{children}</div> : null

TabPanel.propTypes = {
  children: PropTypes.node,
  value: PropTypes.number.isRequired,
  index: PropTypes.number.isRequired,
}

// Pick the lyric track (an entry of the structured LyricList JSON stored in
// media_file.lyrics) for the requested language: 'original' → the main track,
// otherwise the synced translation whose lang matches.
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

// ---- Lyrics tab: full text without timestamps, current line highlighted ----
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

  // Re-read the song record so freshly generated lyrics (written to
  // media_file.lyrics by the backend) show up without a full page reload.
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

  // Active line index = last line whose start <= currentTimeMs. Matches the
  // external player's own lyric parser semantics (line[].start is in ms).
  const activeIndex = useMemo(() => {
    if (lines.length === 0) return -1
    let idx = -1
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].start != null && lines[i].start <= currentTimeMs) idx = i
      else if (lines[i].start == null) {
        // un-synced lines never become "active" on their own
      }
    }
    return idx < 0 ? -1 : idx
  }, [lines, currentTimeMs])

  // Subscribe to the audio element's timeupdate to track the live position.
  // The element is shared from the player via the process-wide registry; we
  // attach our own listener and keep the position in local state.
  useEffect(() => {
    const audio = getAudioInstance()
    if (!audio) return
    const onTime = () => setCurrentTimeMs(Math.floor((audio.currentTime || 0) * 1000))
    onTime()
    audio.addEventListener('timeupdate', onTime)
    return () => audio.removeEventListener('timeupdate', onTime)
  }, [])

  // Auto-scroll the active line into view (without hijacking vertical scroll
  // fights: only when the user isn't interacting).
  useEffect(() => {
    if (activeLineRef.current && scrollContainerRef.current) {
      activeLineRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, [activeIndex])

  // Poll lyrics status while a generation task is queued/running.
  useEffect(() => {
    if (!record || !record.id) return
    if (!lyricsStatus || (lyricsStatus.status !== 'queued' && lyricsStatus.status !== 'running')) return
    const timer = setInterval(() => {
      httpClient(`/api/ai/lyrics/status?mediaFileId=${encodeURIComponent(record.id)}`)
        .then(({ json }) => {
          setLyricsStatus(json)
          // Once done, force a re-read of the record so new lyrics appear.
          if (json && json.status === 'done') {
            // refetch handled by react-admin cache invalidation below
          }
        })
        .catch(() => {})
    }, 2000)
    return () => clearInterval(timer)
  }, [record, lyricsStatus])

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

  // No lyrics at all → empty state with a generate button.
  if (!track || lines.length === 0) {
    return (
      <div className={classes.panelScroll} ref={scrollContainerRef}>
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
            <Button
              className={classes.generateBtn}
              variant="contained"
              color="primary"
              size="small"
              startIcon={<MdLyrics />}
              onClick={handleGenerate}
            >
              {translate('ai.window.generateLyrics')}
            </Button>
          </div>
        )}
      </div>
    )
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className={classes.panelScroll} ref={scrollContainerRef}>
        {ruAvailable && (
          <div className={classes.lyricsLangSwitch}>
            <button
              type="button"
              className={`${classes.langBtn} ${lyricLang === 'original' ? classes.langBtnActive : ''}`}
              onClick={() => dispatch(setLyricLang('original'))}
            >
              {translate('ai.window.original')}
            </button>
            <button
              type="button"
              className={`${classes.langBtn} ${lyricLang === 'ru' ? classes.langBtnActive : ''}`}
              onClick={() => dispatch(setLyricLang('ru'))}
            >
              {translate('ai.window.russian')}
            </button>
          </div>
        )}
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
      </div>
    </div>
  )
}

LyricsTab.propTypes = { record: PropTypes.object }

// ---- Meaning tab: render the .ai.decode.md markdown ----
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

// ---- The draggable floating window ----
const AIWindow = ({ open, onClose, record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [tab, setTab] = useState(0)
  const dragRef = useRef(null)

  // Initial position: centered horizontally, lower-right-ish. react-draggable
  // works in transformed coordinates; defaultPosition sets the starting offset.
  const defaultPos = useMemo(() => ({
    x: Math.max(16, Math.floor((typeof window !== 'undefined' ? window.innerWidth : 1024) / 2 - WINDOW_WIDTH / 2)),
    y: 80,
  }), [])

  useEffect(() => {
    if (open) setTab(0)
  }, [open, record?.id])

  if (!open || !record) return null

  return (
    <Draggable nodeRef={dragRef} handle=".ai-window-header" defaultPosition={defaultPos} bounds="window" cancel="[role=button],button,a">
      <Paper ref={dragRef} className={classes.root} elevation={8}>
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
      </Paper>
    </Draggable>
  )
}

AIWindow.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  record: PropTypes.object,
}

export default AIWindow
