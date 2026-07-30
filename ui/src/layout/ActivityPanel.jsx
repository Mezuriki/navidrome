import React, { useState, useEffect, useCallback } from 'react'
import { useSelector } from 'react-redux'
import { useNotify, useTranslate } from 'react-admin'
import {
  Popover,
  CircularProgress,
  IconButton,
  makeStyles,
  Tooltip,
  Card,
  CardContent,
  CardActions,
  Divider,
  Box,
  Typography,
  Button,
  LinearProgress,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import { BiError, BiMessageError } from 'react-icons/bi'
import { VscSync } from 'react-icons/vsc'
import { GiMagnifyingGlass } from 'react-icons/gi'
import { MdStop } from 'react-icons/md'
import subsonic from '../subsonic'
import { useInitialScanStatus } from './useInitialScanStatus'
import { useInterval } from '../common'
import { useScanElapsedTime } from './useScanElapsedTime'
import { formatDuration, formatShortDuration } from '../utils'
import { httpClient } from '../dataProvider'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  wrapper: {
    position: 'relative',
    color: (props) =>
      props.serverDown
        ? theme.palette.error.main
        : props.hasWarning
          ? theme.palette.warning.main
          : null,
  },
  progress: {
    color: theme.palette.primary.light,
    position: 'absolute',
    top: 10,
    left: 10,
    zIndex: 1,
  },
  button: {
    color: 'inherit',
    zIndex: 2,
  },
  counterStatus: {
    minWidth: '20em',
  },
  error: {
    color: theme.palette.error.main,
  },
  card: {
    maxWidth: 'none',
  },
  cardContent: {
    padding: theme.spacing(2, 3),
  },
}))

const getUptime = (serverStart) =>
  formatDuration((Date.now() - serverStart.startTime) / 1000)

const Uptime = () => {
  const serverStart = useSelector((state) => state.activity.serverStart)
  const [uptime, setUptime] = useState(getUptime(serverStart))
  useInterval(() => {
    setUptime(getUptime(serverStart))
  }, 1000)
  return <span>{uptime}</span>
}

// Mixarr enrich task status (polled from navidrome's own enrich endpoint if
// running, or proxied — for now we poll navidrome's /api/ai/lyrics/status
// across a few known media files to detect active enrichment). This is a
// lightweight proxy: navidrome exposes the running Mixarr task via a shared
// Redis-poll. To keep it simple and avoid coupling, we poll navidrome's
// internal endpoint that mirrors the Mixarr enrich status.
const useMixarrEnrichStatus = () => {
  const [status, setStatus] = useState(null)
  const fetchStatus = useCallback(async () => {
    try {
      const { json } = await httpClient('/api/ai/enrich/status')
      if (json && json.status && json.status !== 'idle') {
        setStatus(json)
      } else {
        setStatus(null)
      }
    } catch {
      setStatus(null)
    }
  }, [])
  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 3000)
    return () => clearInterval(interval)
  }, [fetchStatus])
  return [status, fetchStatus]
}

const MixarrEnrichCard = ({ status, onUpdated }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const [cancelling, setCancelling] = useState(false)

  const handleCancel = async () => {
    setCancelling(true)
    try {
      await httpClient('/api/ai/enrich/cancel', { method: 'POST' })
      notify('Cancelling…', { type: 'info' })
    } catch (e) {
      notify('Failed to cancel', { type: 'error' })
    } finally {
      setCancelling(false)
      onUpdated()
    }
  }

  if (!status) return null
  const pct = status.total > 0 ? Math.round((status.processed / status.total) * 100) : 0

  return (
    <>
      <Divider />
      <CardContent style={{ padding: '16px 24px', minWidth: 300 }}>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
          <Typography variant="subtitle2">
            Mixarr Enrichment
          </Typography>
          <Typography variant="caption" style={{ textTransform: 'capitalize' }}>
            {status.mode || 'lyrics'}
          </Typography>
        </Box>
        {status.currentItem && (
          <Typography variant="body2" color="textSecondary" noWrap style={{ maxWidth: 280 }}>
            {status.currentItem}
          </Typography>
        )}
        {status.currentTrack && (
          <Typography variant="body2" color="textSecondary" noWrap style={{ maxWidth: 280, fontSize: '0.8rem' }}>
            {status.currentTrack}
          </Typography>
        )}
        <Box mt={1} mb={1}>
          <LinearProgress variant="determinate" value={pct} />
        </Box>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="caption" color="textSecondary">
            {status.processed}/{status.total} · ✓{status.enriched}{status.failed > 0 ? ` · ✗${status.failed}` : ''}
          </Typography>
          <Button
            size="small"
            color="secondary"
            disabled={cancelling}
            onClick={handleCancel}
            startIcon={cancelling ? <CircularProgress size={14} /> : <MdStop />}
          >
            {cancelling ? '…' : 'Cancel'}
          </Button>
        </Box>
      </CardContent>
    </>
  )
}

const ActivityPanel = () => {
  const serverStart = useSelector((state) => state.activity.serverStart)
  const up = serverStart.startTime
  const scanStatus = useSelector((state) => state.activity.scanStatus)
  const elapsed = useScanElapsedTime(
    scanStatus.scanning,
    scanStatus.elapsedTime,
  )
  // Determine icon state: error (server down), warning (scan error), or normal
  const serverDown = !up
  const hasWarning = Boolean(scanStatus.error)
  const classes = useStyles({ serverDown, hasWarning })
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  useInitialScanStatus()
  const [enrichStatus, refreshEnrich] = useMixarrEnrichStatus()
  const enrichActive = enrichStatus && (enrichStatus.status === 'running' || enrichStatus.status === 'queued')

  const handleMenuOpen = (event) => {
    setAnchorEl(event.currentTarget)
  }

  const handleMenuClose = () => {
    setAnchorEl(null)
  }
  const triggerScan = (full) => () => subsonic.startScan({ fullScan: full })

  useEffect(() => {
    if (serverStart.version && serverStart.version !== config.version) {
      notify('ra.notification.new_version', 'info', {}, false, 604800000 * 50)
    }
  }, [serverStart, notify])

  const tooltipTitle = scanStatus.error
    ? `${translate('activity.status')}: ${scanStatus.error}`
    : translate('activity.title')

  const lastScanType = (() => {
    switch (scanStatus.scanType) {
      case 'full':
        return translate('activity.fullScan')
      case 'quick':
        return translate('activity.quickScan')
      case 'full-selective':
      case 'quick-selective':
        return translate('activity.selectiveScan')
      default:
        return ''
    }
  })()

  return (
    <div className={classes.wrapper}>
      <Tooltip title={tooltipTitle}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          {serverDown ? (
            <BiError data-testid="activity-error-icon" size={'20'} />
          ) : hasWarning ? (
            <BiMessageError data-testid="activity-warning-icon" size={'20'} />
          ) : (
            <FiActivity data-testid="activity-ok-icon" size={'20'} />
          )}
        </IconButton>
      </Tooltip>
      {scanStatus.scanning && (
        <CircularProgress size={24} className={classes.progress} />
      )}
      <Popover
        id="panel-activity"
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'right',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        open={open}
        onClose={handleMenuClose}
      >
        <Card className={classes.card}>
          <CardContent className={classes.cardContent}>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.serverUptime')}:
              </Box>
              <Box
                component="span"
                flex={1}
                className={!up ? classes.error : null}
              >
                {up ? <Uptime /> : translate('activity.serverDown')}
              </Box>
            </Box>
          </CardContent>
          <Divider />
          <CardContent className={classes.cardContent}>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.totalScanned')}:
              </Box>
              <Box component="span" flex={1}>
                {scanStatus.folderCount || '-'}
              </Box>
            </Box>

            <Box display="flex" className={classes.counterStatus} mt={2}>
              <Box component="span" flex={2}>
                {translate('activity.scanType')}:
              </Box>
              <Box component="span" flex={1}>
                {lastScanType}
              </Box>
            </Box>

            <Box display="flex" className={classes.counterStatus} mt={2}>
              <Box component="span" flex={2}>
                {translate('activity.elapsedTime')}:
              </Box>
              <Box component="span" flex={1}>
                {formatShortDuration(elapsed)}
              </Box>
            </Box>

            {scanStatus.error && (
              <Box
                display="flex"
                flexDirection="column"
                mt={2}
                className={classes.error}
              >
                <Typography variant="subtitle2">
                  {translate('activity.status')}:
                </Typography>
                <Typography variant="body2">{scanStatus.error}</Typography>
              </Box>
            )}
          </CardContent>
          <Divider />
          <CardActions>
            <Tooltip title={translate('activity.quickScan')}>
              <IconButton
                onClick={triggerScan(false)}
                disabled={scanStatus.scanning}
              >
                <VscSync />
              </IconButton>
            </Tooltip>
            <Tooltip title={translate('activity.fullScan')}>
              <IconButton
                onClick={triggerScan(true)}
                disabled={scanStatus.scanning}
              >
                <GiMagnifyingGlass />
              </IconButton>
            </Tooltip>
          </CardActions>
        </Card>
      </Popover>
    </div>
  )
}

export default ActivityPanel
