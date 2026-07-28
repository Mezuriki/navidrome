import React from 'react'
import PropTypes from 'prop-types'
import { useSelector, useDispatch } from 'react-redux'
import AIWindow from '../dialogs/AIWindow'
import { closeAIDrawer } from '../actions'

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

  const handleClose = () => dispatch(closeAIDrawer())

  // When the window was opened from the player toolbar (followPlayer), the
  // displayed record should always reflect the currently-playing track, so
  // switching songs in the player immediately refreshes the window's data
  // (metadata, lyrics, meaning) without reopening it.
  let record = aiDrawer.record
  if (aiDrawer.open && aiDrawer.followPlayer) {
    record = toRecord(currentTrack) || aiDrawer.record
  }

  return <AIWindow open={aiDrawer.open} onClose={handleClose} record={record} />
}

AIDrawerContainer.propTypes = {}

export default AIDrawerContainer
