import React from 'react'
import PropTypes from 'prop-types'
import { useSelector, useDispatch } from 'react-redux'
import { ThemeProvider } from '@material-ui/core/styles'
import { createMuiTheme } from 'react-admin'
import AIWindow from '../dialogs/AIWindow'
import { closeAIDrawer } from '../actions'
import useCurrentTheme from '../themes/useCurrentTheme'

// Normalise a player queue item / audioInfo into the record shape the AI window
// expects (id, title, artist, album, year, genre, lyrics). The player stores
// the full song object under `song`, but it can also be a bare song record.
const toRecord = (item) => {
  if (!item) return undefined
  if (item.song) {
    const s = item.song
    return {
      id: s.mediaFileId || s.id,
      mediaFileId: s.mediaFileId || s.id,
      title: s.title || item.name,
      artist: s.artist || item.singer,
      album: s.album,
      year: s.year,
      genre: s.genre,
      lyrics: s.lyrics,
    }
  }
  return item
}

const AIDrawerContainer = () => {
  const dispatch = useDispatch()
  const aiDrawer = useSelector((state) => state.aiDrawer || { open: false, record: undefined, followPlayer: false })
  const currentTrack = useSelector((state) => state.player?.current)
  // The window is rendered as a sibling of <RALayout> in Layout.jsx, i.e. OUTSIDE
  // react-admin's ThemeProvider. Without its own ThemeProvider, makeStyles() in
  // AIWindow falls back to the default (light) MUI theme and the generated
  // class rules (e.g. for MuiCard) leak globally — turning the dark-theme page
  // cards white whenever the window mounts. Wrapping it here in a ThemeProvider
  // with the active app theme isolates the styles and keeps them dark.
  const theme = useCurrentTheme()

  const handleClose = () => dispatch(closeAIDrawer())

  // When the window was opened from the player toolbar (followPlayer), the
  // displayed record should always reflect the currently-playing track, so
  // switching songs in the player immediately refreshes the window's data.
  let record = aiDrawer.record
  if (aiDrawer.open && aiDrawer.followPlayer) {
    record = toRecord(currentTrack) || aiDrawer.record
  }

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <AIWindow open={aiDrawer.open} onClose={handleClose} record={record} />
    </ThemeProvider>
  )
}

AIDrawerContainer.propTypes = {}

export default AIDrawerContainer
