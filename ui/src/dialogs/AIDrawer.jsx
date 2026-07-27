import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import {
  Drawer,
  Typography,
  Box,
  Tabs,
  Tab,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CircularProgress,
  Divider,
  Chip,
  Paper,
  IconButton,
} from '@material-ui/core'
import {
  Close as CloseIcon,
  Translate as TranslateIcon,
  Autorenew as DecodeIcon,
  Info as AnalyzeIcon,
  Send as SendIcon,
} from '@material-ui/icons'
import { MdPsychology, MdLyrics } from 'react-icons/md'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate, useNotify } from 'react-admin'
import { httpClient } from '../dataProvider'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  drawer: {
    width: 450,
    flexShrink: 0,
  },
  drawerPaper: {
    width: 450,
    padding: theme.spacing(2),
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing(2),
  },
  title: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    fontWeight: 600,
  },
  content: {
    height: 'calc(100vh - 180px)',
    overflowY: 'auto',
  },
  tabPanel: {
    padding: theme.spacing(2, 0),
  },
  formControl: {
    margin: theme.spacing(1, 0),
    minWidth: '100%',
  },
  resultBox: {
    marginTop: theme.spacing(2),
    padding: theme.spacing(2),
    backgroundColor: theme.palette.background.default,
    borderRadius: theme.shape.borderRadius,
    minHeight: 100,
  },
  resultText: {
    whiteSpace: 'pre-wrap',
    lineHeight: 1.6,
  },
  loadingBox: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    padding: theme.spacing(4),
  },
  actionButtons: {
    display: 'flex',
    gap: theme.spacing(1),
    marginTop: theme.spacing(2),
  },
  chip: {
    margin: theme.spacing(0.5),
  },
  tagsContainer: {
    display: 'flex',
    flexWrap: 'wrap',
    marginTop: theme.spacing(1),
  },
  metadataBox: {
    padding: theme.spacing(1.5),
    marginBottom: theme.spacing(2),
    backgroundColor: theme.palette.action.hover,
    borderRadius: theme.shape.borderRadius,
  },
  metadataText: {
    fontSize: '0.875rem',
    color: theme.palette.text.secondary,
  },
}))

const TabPanel = ({ children, value, index }) => {
  return value === index ? <div>{children}</div> : null
}

// formatDecode turns the AI decode response into readable text.
const formatDecode = (json) => {
  if (!json) return ''
  return (json.text || '').trim() || '(no response from model)'
}

// formatAnalyze turns the AI analyze response into readable text.
const formatAnalyze = (json) => {
  if (!json) return ''
  return (json.text || '').trim() || '(no response from model)'
}

// aiErrorMessage extracts a human-readable message from an httpClient error.
// The backend returns JSON {error, retryable} for AI failures; react-admin's
// fetchJson puts the raw body in error.message, so we try to parse it. When the
// error is retryable (quota / rate limit), we surface a clear "try later" hint.
const aiErrorMessage = (error, translate) => {
  let msg = error?.message || ''
  let retryable = false
  try {
    const parsed = JSON.parse(msg)
    if (parsed && typeof parsed === 'object') {
      msg = parsed.error || msg
      retryable = !!parsed.retryable
    }
  } catch {
    // not JSON — fall back to the raw message
  }
  if (retryable) {
    return translate('ai.error.quota')
  }
  return msg
}

const AIDrawer = ({ open, onClose, record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const [tabValue, setTabValue] = useState(0)
  const [translateLanguage, setTranslateLanguage] = useState('ru')
  const [translationResult, setTranslationResult] = useState('')
  const [decodeResult, setDecodeResult] = useState('')
  const [analyzeResult, setAnalyzeResult] = useState('')
  const [loading, setLoading] = useState(false)
  const [lyricsStatus, setLyricsStatus] = useState(null)
  const [lyricsFetching, setLyricsFetching] = useState(false)

  // When the record changes, load any previously persisted AI results from
  // sidecar files so the drawer is not empty on open.
  useEffect(() => {
    setTranslationResult('')
    setDecodeResult('')
    setAnalyzeResult('')
    setLyricsStatus(null)
    if (!record || !record.id) return
    const id = encodeURIComponent(record.id)
    // Translate (ru by default)
    httpClient(`/api/ai/translate?mediaFileId=${id}&lang=ru`)
      .then(({ json }) => {
        if (json && json.found && json.text) setTranslationResult(json.text)
      })
      .catch(() => {})
    // Decode
    httpClient(`/api/ai/decode?mediaFileId=${id}`)
      .then(({ json }) => {
        if (json && json.found && json.text) setDecodeResult(json.text)
      })
      .catch(() => {})
    // Analyze
    httpClient(`/api/ai/analyze?mediaFileId=${id}`)
      .then(({ json }) => {
        if (json && json.found && json.text) setAnalyzeResult(json.text)
      })
      .catch(() => {})
  }, [record])

  // Poll the lyrics fetch status while a task is queued or running.
  useEffect(() => {
    if (!record || !record.id) return
    if (
      !lyricsStatus ||
      (lyricsStatus.status !== 'queued' && lyricsStatus.status !== 'running')
    ) {
      return
    }
    const timer = setInterval(() => {
      httpClient(
        `/api/ai/lyrics/status?mediaFileId=${encodeURIComponent(record.id)}`,
      )
        .then(({ json }) => setLyricsStatus(json))
        .catch(() => {})
    }, 2000)
    return () => clearInterval(timer)
  }, [record, lyricsStatus])

  // Load the existing lyrics status when the Lyrics tab is first opened.
  useEffect(() => {
    if (tabValue !== 3 || !record || !record.id || lyricsStatus) return
    httpClient(
      `/api/ai/lyrics/status?mediaFileId=${encodeURIComponent(record.id)}`,
    )
      .then(({ json }) => setLyricsStatus(json))
      .catch(() => {})
  }, [tabValue, record, lyricsStatus])

  const handleTranslate = async () => {
    setLoading(true)
    try {
      const { json } = await httpClient('/api/ai/translate', {
        method: 'POST',
        body: JSON.stringify({
          title: record.title,
          artist: record.artist,
          lyrics: record.lyrics || '',
          toLang: translateLanguage,
          mediaFileId: record.id,
        }),
      })
      setTranslationResult(json.translation)
      notify('ai.success.translate', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.translate') + ': ' + aiErrorMessage(error, translate), {
        type: 'error',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleDecode = async () => {
    setLoading(true)
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
      setDecodeResult(formatDecode(json))
      notify('ai.success.decode', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.decode') + ': ' + aiErrorMessage(error, translate), {
        type: 'error',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleAnalyze = async () => {
    setLoading(true)
    try {
      const { json } = await httpClient('/api/ai/analyze', {
        method: 'POST',
        body: JSON.stringify({
          title: record.title,
          artist: record.artist,
          album: record.album,
          year: record.year,
          genre: record.genre,
          lyrics: record.lyrics || '',
          mediaFileId: record.id,
        }),
      })
      setAnalyzeResult(formatAnalyze(json))
      notify('ai.success.analyze', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.analyze') + ': ' + aiErrorMessage(error, translate), {
        type: 'error',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleFetchLyrics = async () => {
    if (!record || !record.id) return
    setLyricsFetching(true)
    try {
      await httpClient('/api/ai/lyrics/fetch', {
        method: 'POST',
        body: JSON.stringify({ mediaFileId: record.id }),
      })
      // Seed the status so the polling effect kicks in.
      setLyricsStatus({ status: 'queued', step: 'lyrics' })
    } catch (error) {
      notify(translate('ai.lyrics.error') + ': ' + error.message, {
        type: 'error',
      })
    } finally {
      setLyricsFetching(false)
    }
  }

  // Fetch the existing status once when the Lyrics tab is opened for a record.
  // (handled by the useEffect above on tabValue change)


  if (!record) return null

  const languages = [
    { code: 'ru', name: 'Русский' },
    { code: 'en', name: 'English' },
    { code: 'de', name: 'Deutsch' },
    { code: 'fr', name: 'Français' },
    { code: 'es', name: 'Español' },
    { code: 'it', name: 'Italiano' },
    { code: 'pt', name: 'Português' },
    { code: 'ja', name: '日本語' },
    { code: 'zh', name: '中文' },
    { code: 'ko', name: '한국어' },
  ]

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      classes={{ paper: classes.drawerPaper }}
    >
      <Box className={classes.header}>
        <Typography variant="h6" className={classes.title}>
          <MdPsychology fontSize="small" />
          AI Assistant
        </Typography>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </Box>

      <Box className={classes.metadataBox}>
        <Typography variant="body2" className={classes.metadataText}>
          <strong>{record.title}</strong> • {record.artist}
        </Typography>
        <Typography variant="body2" className={classes.metadataText}>
          {record.album} ({record.year})
        </Typography>
        {record.genre && (
          <Chip
            size="small"
            label={record.genre}
            className={classes.chip}
          />
        )}
      </Box>

      <Tabs
        value={tabValue}
        onChange={(e, v) => setTabValue(v)}
        variant="fullWidth"
      >
        <Tab label={translate('ai.translate.title')} icon={<TranslateIcon />} />
        <Tab label={translate('ai.decode.title')} icon={<DecodeIcon />} />
        <Tab label={translate('ai.analyze.title')} icon={<AnalyzeIcon />} />
        <Tab label={translate('ai.lyrics.title')} icon={<MdLyrics />} />
      </Tabs>

      <Box className={classes.content}>
        <TabPanel value={tabValue} index={0}>
          <FormControl className={classes.formControl}>
            <InputLabel>{translate('ai.translate.targetLanguage')}</InputLabel>
            <Select
              value={translateLanguage}
              onChange={(e) => setTranslateLanguage(e.target.value)}
            >
              {languages.map((lang) => (
                <MenuItem key={lang.code} value={lang.code}>
                  {lang.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          {!record.lyrics && (
            <Box className={classes.resultBox}>
              <Typography variant="body2" color="textSecondary">
                {translate('ai.translate.recallHint')}
              </Typography>
            </Box>
          )}

          <Box className={classes.actionButtons}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleTranslate}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={16} /> : <SendIcon />}
              fullWidth
            >
              {translate('ai.translate.action')}
            </Button>
          </Box>

          {translationResult && (
            <Paper className={classes.resultBox}>
              <Typography variant="body1" className={classes.resultText}>
                {translationResult}
              </Typography>
            </Paper>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('ai.decode.description')}
          </Typography>

          <Box className={classes.actionButtons}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleDecode}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={16} /> : <DecodeIcon />}
              fullWidth
            >
              {translate('ai.decode.action')}
            </Button>
          </Box>

          {decodeResult && (
            <Paper className={classes.resultBox}>
              <Typography variant="body1" className={classes.resultText}>
                {decodeResult}
              </Typography>
            </Paper>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('ai.analyze.description')}
          </Typography>

          <Box className={classes.actionButtons}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleAnalyze}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={16} /> : <AnalyzeIcon />}
              fullWidth
            >
              {translate('ai.analyze.action')}
            </Button>
          </Box>

          {analyzeResult && (
            <Paper className={classes.resultBox}>
              <Typography variant="body1" className={classes.resultText}>
                {analyzeResult}
              </Typography>
            </Paper>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={3}>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('ai.lyrics.description')}
          </Typography>

          {lyricsStatus && lyricsStatus.status === 'done' && (
            <Box className={classes.resultBox}>
              <Typography variant="body2" color="primary">
                ✓ {translate('ai.lyrics.done')}
              </Typography>
            </Box>
          )}

          {lyricsStatus && lyricsStatus.status === 'error' && (
            <Box className={classes.resultBox}>
              <Typography variant="body2" color="error">
                ✗ {translate('ai.lyrics.error')}: {lyricsStatus.error}
              </Typography>
            </Box>
          )}

          {lyricsStatus &&
            (lyricsStatus.status === 'queued' ||
              lyricsStatus.status === 'running') && (
              <Box className={classes.resultBox}>
                <CircularProgress size={16} />
                <Typography variant="body2" color="textSecondary">
                  {lyricsStatus.step === 'translation'
                    ? translate('ai.lyrics.statusTranslation')
                    : translate('ai.lyrics.statusLyrics')}
                  …
                </Typography>
              </Box>
            )}

          <Box className={classes.actionButtons}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleFetchLyrics}
              disabled={
                lyricsFetching ||
                (lyricsStatus &&
                  (lyricsStatus.status === 'queued' ||
                    lyricsStatus.status === 'running'))
              }
              startIcon={
                lyricsFetching ? <CircularProgress size={16} /> : <MdLyrics />
              }
              fullWidth
            >
              {translate('ai.lyrics.action')}
            </Button>
          </Box>
        </TabPanel>
      </Box>
    </Drawer>
  )
}

AIDrawer.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  record: PropTypes.object,
}

export default AIDrawer
