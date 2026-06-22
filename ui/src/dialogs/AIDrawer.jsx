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
import { MdPsychology } from 'react-icons/md'
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

// formatDecode turns the structured AI decode response into readable text.
const formatDecode = (json) => {
  if (!json) return ''
  const parts = []
  if (json.meaning) parts.push(json.meaning)
  if (json.mood) parts.push(`\n🎵 ${json.mood}`)
  if (json.themes && json.themes.length) {
    parts.push('\n🏷️ ' + json.themes.join(', '))
  }
  if (json.interpretation) parts.push('\n\n' + json.interpretation)
  return parts.join('\n').trim() || json.description || ''
}

// formatAnalyze turns the structured AI analyze response into readable text.
const formatAnalyze = (json) => {
  if (!json) return ''
  const lines = []
  if (json.genre) lines.push(`🎶 Genre: ${json.genre}`)
  if (json.mood && json.mood.length)
    lines.push(`🎭 Mood: ${json.mood.join(', ')}`)
  if (json.style && json.style.length)
    lines.push(`🎨 Style: ${json.style.join(', ')}`)
  if (json.themes && json.themes.length)
    lines.push(`🏷️ Themes: ${json.themes.join(', ')}`)
  if (json.similarArtists && json.similarArtists.length)
    lines.push(`👥 Similar: ${json.similarArtists.join(', ')}`)
  if (json.description) lines.push(`\n${json.description}`)
  return lines.join('\n').trim() || JSON.stringify(json, null, 2)
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

  // Reset results when record changes
  useEffect(() => {
    setTranslationResult('')
    setDecodeResult('')
    setAnalyzeResult('')
  }, [record])

  const handleTranslate = async () => {
    if (!record?.lyrics) {
      notify('ai.warning.noLyrics', { type: 'warning' })
      return
    }

    setLoading(true)
    try {
      const { json } = await httpClient('/api/ai/translate', {
        method: 'POST',
        body: JSON.stringify({
          text: record.lyrics,
          toLang: translateLanguage,
        }),
      })
      setTranslationResult(json.translation)
      notify('ai.success.translate', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.translate') + ': ' + error.message, {
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
        }),
      })
      setDecodeResult(formatDecode(json))
      notify('ai.success.decode', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.decode') + ': ' + error.message, {
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
        }),
      })
      setAnalyzeResult(formatAnalyze(json))
      notify('ai.success.analyze', { type: 'success' })
    } catch (error) {
      notify(translate('ai.error.analyze') + ': ' + error.message, {
        type: 'error',
      })
    } finally {
      setLoading(false)
    }
  }

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
                {translate('ai.translate.noLyrics')}
              </Typography>
            </Box>
          )}

          <Box className={classes.actionButtons}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleTranslate}
              disabled={!record.lyrics || loading}
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
