import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import IconButton from '@material-ui/core/IconButton'
import Select from '@material-ui/core/Select'
import MenuItem from '@material-ui/core/MenuItem'
import { useMediaQuery } from '@material-ui/core'
import { RiSaveLine } from 'react-icons/ri'
import { MdPsychology, MdTranslate } from 'react-icons/md'
import { LoveButton, useToggleLove } from '../common'
import {
  openSaveQueueDialog,
  openAIDrawer,
  setLyricLang,
} from '../actions'
import { keyMap } from '../hotkeys'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles((theme) => ({
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    flexGrow: 1,
    justifyContent: 'flex-end',
    gap: '0.5rem',
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  mobileListItem: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    listStyle: 'none',
    padding: theme.spacing(0.5),
    margin: 0,
    height: 24,
  },
  button: {
    width: '2.5rem',
    height: '2.5rem',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 0,
  },
  mobileButton: {
    width: 24,
    height: 24,
    padding: 0,
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '18px',
  },
  mobileIcon: {
    fontSize: '18px',
    display: 'flex',
    alignItems: 'center',
  },
  langSelect: {
    minWidth: '2rem',
    fontSize: '0.8rem',
    '&:before': { display: 'none' },
    '& .MuiSelect-select': { paddingRight: '18px !important', paddingTop: 0, paddingBottom: 0 },
  },
}))

const PlayerToolbar = ({ id, isRadio }) => {
  const dispatch = useDispatch()
  const lyricLang = useSelector((state) => state.player?.lyricLang || 'original')
  const { data, loading } = useGetOne('song', id, { enabled: !!id && !isRadio })
  const [toggleLove, toggling] = useToggleLove('song', data)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const classes = useStyles()

  // Discover which lyric languages are available for the current track by
  // parsing item.lyrics. Always offers 'original'; adds an entry per synced
  // translation track (e.g. 'ru').
  const lyricLanguageOptions = React.useMemo(() => {
    const opts = [{ value: 'original', label: 'Original' }]
    if (data?.lyrics) {
      try {
        const structured = JSON.parse(data.lyrics)
        if (Array.isArray(structured)) {
          structured
            .filter((l) => l && l.synced && l.kind === 'translation' && l.lang)
            .forEach((l) => {
              if (!opts.find((o) => o.value === l.lang)) {
                opts.push({ value: l.lang, label: l.lang.toUpperCase() })
              }
            })
        }
      } catch {
        // ignore parse errors — keep just 'original'
      }
    }
    return opts
  }, [data])

  const handleLangChange = useCallback(
    (e) => {
      dispatch(setLyricLang(e.target.value))
      e.stopPropagation()
    },
    [dispatch],
  )

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  const handleSaveQueue = useCallback(
    (e) => {
      dispatch(openSaveQueueDialog())
      e.stopPropagation()
    },
    [dispatch],
  )

  // Open the AI Assistant drawer (lyrics / translation / meaning) for the
  // currently playing track. The drawer is mounted globally in Layout and
  // driven by Redux state (state.aiDrawer).
  const handleOpenAI = useCallback(
    (e) => {
      if (data) {
        dispatch(openAIDrawer(data))
      }
      e.stopPropagation()
    },
    [dispatch, data],
  )

  const buttonClass = isDesktop ? classes.button : classes.mobileButton
  const listItemClass = isDesktop ? classes.toolbar : classes.mobileListItem

  const saveQueueButton = (
    <IconButton
      size={isDesktop ? 'small' : undefined}
      onClick={handleSaveQueue}
      disabled={isRadio}
      data-testid="save-queue-button"
      className={buttonClass}
    >
      <RiSaveLine className={!isDesktop ? classes.mobileIcon : undefined} />
    </IconButton>
  )

  const loveButton = (
    <LoveButton
      record={data}
      resource={'song'}
      size={isDesktop ? undefined : 'inherit'}
      disabled={loading || toggling || !id || isRadio}
      className={buttonClass}
    />
  )

  const aiButton = (
    <IconButton
      size={isDesktop ? 'small' : undefined}
      onClick={handleOpenAI}
      disabled={loading || !id || isRadio}
      aria-label="AI Assistant"
      title="AI Assistant"
      className={buttonClass}
    >
      <MdPsychology className={!isDesktop ? classes.mobileIcon : undefined} />
    </IconButton>
  )

  // Lyric language switch. Hidden when there is no translation available, so it
  // only appears for tracks that actually have a synced translation sidecar.
  const showLangSelect =
    !isRadio && lyricLanguageOptions.length > 1 && data?.lyrics
  const langSelect = showLangSelect ? (
    <Select
      value={lyricLang}
      onChange={handleLangChange}
      disableUnderline
      title="Lyric language"
      className={classes.langSelect}
      IconComponent={(props) => <MdTranslate {...props} />}
      renderValue={(value) =>
        (lyricLanguageOptions.find((o) => o.value === value) || {}).label ||
        value
      }
    >
      {lyricLanguageOptions.map((o) => (
        <MenuItem key={o.value} value={o.value}>
          {o.label}
        </MenuItem>
      ))}
    </Select>
  ) : null

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      {isDesktop ? (
        <li className={`${listItemClass} item`}>
          {saveQueueButton}
          {aiButton}
          {langSelect}
          {loveButton}
        </li>
      ) : (
        <>
          <li className={`${listItemClass} item`}>{saveQueueButton}</li>
          <li className={`${listItemClass} item`}>{aiButton}</li>
          {langSelect && <li className={`${listItemClass} item`}>{langSelect}</li>}
          <li className={`${listItemClass} item`}>{loveButton}</li>
        </>
      )}
    </>
  )
}

export default PlayerToolbar
