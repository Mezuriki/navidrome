import React, { useState, useEffect } from 'react'
import {
  Button,
  Card,
  CardContent,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
  Box,
} from '@material-ui/core'
import { MdPsychology, MdSave } from 'react-icons/md'
import { makeStyles } from '@material-ui/core/styles'
import { useNotify, useTranslate } from 'react-admin'
import { httpClient } from '../dataProvider'

const KEY_MASK = '********'

const useStyles = makeStyles((theme) => ({
  root: { marginTop: '2em', width: '100%' },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginBottom: theme.spacing(1),
  },
  field: {
    marginTop: theme.spacing(2),
    width: '100%',
  },
  saveButton: {
    marginTop: theme.spacing(3),
  },
  statusChip: {
    fontSize: '0.8rem',
    marginLeft: theme.spacing(1),
  },
}))

const AIProviderChoices = [
  { id: 'openai', name: 'OpenAI (GPT-4o, GPT-3.5)' },
  { id: 'anthropic', name: 'Anthropic (Claude)' },
  { id: 'ollama', name: 'Ollama (local)' },
  { id: 'localai', name: 'LocalAI' },
  { id: 'openrouter', name: 'OpenRouter' },
]

const DefaultLanguageChoices = [
  { id: 'en', name: 'English' },
  { id: 'ru', name: 'Русский' },
  { id: 'de', name: 'Deutsch' },
  { id: 'fr', name: 'Français' },
  { id: 'es', name: 'Español' },
  { id: 'it', name: 'Italiano' },
  { id: 'pt', name: 'Português' },
  { id: 'ja', name: '日本語' },
  { id: 'zh', name: '中文' },
  { id: 'ko', name: '한국어' },
]

const placeholders = {
  openai: { apiEndpoint: 'https://api.openai.com/v1', model: 'gpt-4o-mini' },
  anthropic: {
    apiEndpoint: 'https://api.anthropic.com/v1',
    model: 'claude-3-5-sonnet-20241022',
  },
  ollama: {
    apiEndpoint: 'http://localhost:11434/v1',
    model: 'codellama:7b-instruct',
  },
  localai: { apiEndpoint: 'http://localhost:8080/v1', model: 'gpt-3.5-turbo' },
  openrouter: {
    apiEndpoint: 'https://openrouter.ai/api/v1',
    model: 'openai/gpt-4o-mini',
  },
}

const AIConfig = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [configured, setConfigured] = useState(false)
  const [form, setForm] = useState({
    provider: '',
    apiKey: '',
    apiEndpoint: '',
    model: '',
    defaultLanguage: 'ru',
  })

  useEffect(() => {
    httpClient('/api/ai/config')
      .then(({ json }) => {
        setConfigured(json.configured === true)
        setForm({
          provider: json.provider || '',
          apiKey: json.apiKey || '',
          apiEndpoint: json.apiEndpoint || '',
          model: json.model || '',
          defaultLanguage: json.defaultLanguage || 'ru',
        })
      })
      .catch(() => notify('ai.config.error', { type: 'warning' }))
      .finally(() => setLoading(false))
  }, [notify])

  const setField = (field) => (e) =>
    setForm((f) => ({ ...f, [field]: e.target.value }))

  const handleProviderChange = (e) => {
    const provider = e.target.value
    const ph = placeholders[provider] || {}
    setForm((f) => ({
      ...f,
      provider,
      apiEndpoint: ph.apiEndpoint || f.apiEndpoint,
      model: ph.model || f.model,
    }))
  }

  const handleSave = (e) => {
    e.preventDefault()
    setSaving(true)
    httpClient('/api/ai/config', {
      method: 'PUT',
      body: JSON.stringify(form),
    })
      .then(() => {
        notify('ai.config.saved', { type: 'success' })
        setConfigured(true)
      })
      .catch((err) =>
        notify(translate('ai.config.error') + ': ' + err.message, {
          type: 'error',
        }),
      )
      .finally(() => setSaving(false))
  }

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={2}>
        <CircularProgress size={24} />
      </Box>
    )
  }

  return (
    <Card className={classes.root}>
      <CardContent>
        <Box className={classes.header}>
          <MdPsychology size={20} />
          <Typography variant="h6">
            {translate('ai.config.title')}
          </Typography>
          {configured && (
            <Typography
              variant="caption"
              className={classes.statusChip}
              color="primary"
            >
              ✓ {translate('ai.config.configured')}
            </Typography>
          )}
        </Box>
        <Typography variant="body2" color="textSecondary">
          {translate('ai.config.description')}
        </Typography>

        <form onSubmit={handleSave}>
          <FormControl className={classes.field} variant="outlined">
            <InputLabel>{translate('ai.config.provider')}</InputLabel>
            <Select value={form.provider} onChange={handleProviderChange} label={translate('ai.config.provider')}>
              {AIProviderChoices.map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <TextField
            className={classes.field}
            label={translate('ai.config.apiKey')}
            type="password"
            variant="outlined"
            value={form.apiKey}
            onChange={setField('apiKey')}
            placeholder={form.provider === 'ollama' ? '(not required for Ollama)' : ''}
            helperText={
              form.apiKey === KEY_MASK
                ? translate('ai.config.apiKeyMasked')
                : translate('ai.config.apiKeyHelp')
            }
          />

          <TextField
            className={classes.field}
            label={translate('ai.config.apiEndpoint')}
            variant="outlined"
            value={form.apiEndpoint}
            onChange={setField('apiEndpoint')}
            helperText={translate('ai.config.apiEndpointHelp')}
          />

          <TextField
            className={classes.field}
            label={translate('ai.config.model')}
            variant="outlined"
            value={form.model}
            onChange={setField('model')}
            helperText={translate('ai.config.modelHelp')}
          />

          <FormControl className={classes.field} variant="outlined">
            <InputLabel>{translate('ai.config.defaultLanguage')}</InputLabel>
            <Select
              value={form.defaultLanguage}
              onChange={setField('defaultLanguage')}
              label={translate('ai.config.defaultLanguage')}
            >
              {DefaultLanguageChoices.map((l) => (
                <MenuItem key={l.id} value={l.id}>
                  {l.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <Button
            type="submit"
            variant="contained"
            color="primary"
            className={classes.saveButton}
            disabled={saving || !form.provider}
            startIcon={
              saving ? <CircularProgress size={16} /> : <MdSave />
            }
          >
            {translate('ra.action.save')}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

export default AIConfig
