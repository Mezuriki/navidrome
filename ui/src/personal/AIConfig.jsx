import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import {
  SimpleForm,
  TextInput,
  SelectInput,
  useDataProvider,
  useNotify,
  useTranslate,
  SaveContextProvider,
} from 'react-admin'
import { Typography, Box, Card, CardContent } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { MdPsychology as AIIcon } from 'react-icons/md'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginBottom: '16px',
  },
  infoText: {
    color: 'rgba(0, 0, 0, 0.54)',
    fontSize: '0.875rem',
    marginTop: '8px',
    marginBottom: '16px',
  },
  section: {
    marginBottom: '20px',
  },
})

const AIProviderChoices = [
  { id: 'openai', name: 'OpenAI (GPT-4, GPT-3.5)' },
  { id: 'anthropic', name: 'Anthropic (Claude)' },
  { id: 'ollama', name: 'Ollama (Local)' },
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

const AIConfig = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [loading, setLoading] = useState(true)
  const [config, setConfig] = useState({
    provider: '',
    apiKey: '',
    apiEndpoint: '',
    model: '',
    defaultLanguage: 'en',
  })

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const result = await dataProvider.getAIConfig()
        setConfig(result.data || {})
      } catch (error) {
        console.error('Failed to load AI config:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchConfig()
  }, [dataProvider])

  const handleSubmit = async (values) => {
    try {
      await dataProvider.saveAIConfig({ data: values })
      notify('ai.config.saved', { type: 'success' })
    } catch (error) {
      notify(`ai.config.error: ${error.message}`, { type: 'error' })
    }
  }

  const getPlaceholderForProvider = (field) => {
    const placeholders = {
      openai: {
        apiEndpoint: 'https://api.openai.com/v1',
        model: 'gpt-4o-mini',
      },
      anthropic: {
        apiEndpoint: 'https://api.anthropic.com/v1',
        model: 'claude-3-5-sonnet-20241022',
      },
      ollama: {
        apiEndpoint: 'http://localhost:11434/api',
        model: 'llama3',
      },
      localai: {
        apiEndpoint: 'http://localhost:8080/v1',
        model: 'ggml-gpt4all-j',
      },
      openrouter: {
        apiEndpoint: 'https://openrouter.ai/api/v1',
        model: 'anthropic/claude-3.5-sonnet',
      },
    }
    return placeholders[config.provider]?.[field] || ''
  }

  if (loading) return null

  return (
    <Card className={classes.root}>
      <CardContent>
        <Box className={classes.header}>
          <AIIcon color="primary" />
          <Typography variant="h6">
            {translate('ai.config.title')}
          </Typography>
        </Box>

        <Typography variant="body2" className={classes.infoText}>
          {translate('ai.config.description')}
        </Typography>

        <SaveContextProvider value={{ save: handleSubmit }}>
          <SimpleForm toolbar={null} variant="outlined" onSubmit={handleSubmit}>
            <SelectInput
              source="provider"
              label={translate('ai.config.provider')}
              choices={AIProviderChoices}
              defaultValue={config.provider}
              fullWidth
            />

            <TextInput
              source="apiKey"
              label={translate('ai.config.apiKey')}
              defaultValue={config.apiKey}
              type="password"
              fullWidth
              helperText={translate('ai.config.apiKeyHelp')}
            />

            <TextInput
              source="apiEndpoint"
              label={translate('ai.config.apiEndpoint')}
              defaultValue={config.apiEndpoint || getPlaceholderForProvider('apiEndpoint')}
              fullWidth
              helperText={translate('ai.config.apiEndpointHelp')}
            />

            <TextInput
              source="model"
              label={translate('ai.config.model')}
              defaultValue={config.model || getPlaceholderForProvider('model')}
              fullWidth
              helperText={translate('ai.config.modelHelp')}
            />

            <SelectInput
              source="defaultLanguage"
              label={translate('ai.config.defaultLanguage')}
              choices={DefaultLanguageChoices}
              defaultValue={config.defaultLanguage || 'en'}
              fullWidth
            />
          </SimpleForm>
        </SaveContextProvider>
      </CardContent>
    </Card>
  )
}

export default AIConfig
