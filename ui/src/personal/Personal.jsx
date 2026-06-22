import { SimpleForm, Title, useTranslate } from 'react-admin'
import { Card } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import { ListenBrainzScrobbleToggle } from './ListenBrainzScrobbleToggle'
import config from '../config'
import { ReplayGainToggle } from './ReplayGainToggle'
import AIConfig from './AIConfig'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <Card className={classes.root}>
        <SimpleForm toolbar={null} variant={'outlined'}>
          <SelectTheme />
          <SelectLanguage />
          <SelectDefaultView />
          {config.enableReplayGain && <ReplayGainToggle />}
          <NotificationsToggle />
          {config.lastFMEnabled && <LastfmScrobbleToggle />}
          {config.listenBrainzEnabled && <ListenBrainzScrobbleToggle />}
        </SimpleForm>
      </Card>
      <AIConfig />
    </>
  )
}

export default Personal
