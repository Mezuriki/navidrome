import React from 'react'
import PropTypes from 'prop-types'
import { useSelector, useDispatch } from 'react-redux'
import AIDrawer from '../dialogs/AIDrawer'
import { closeAIDrawer } from '../actions'

const AIDrawerContainer = () => {
  const dispatch = useDispatch()
  const { open, record } = useSelector((state) => state.aiDrawer || { open: false, record: undefined })

  const handleClose = () => {
    dispatch(closeAIDrawer())
  }

  return <AIDrawer open={open} onClose={handleClose} record={record} />
}

AIDrawerContainer.propTypes = {}

export default AIDrawerContainer
