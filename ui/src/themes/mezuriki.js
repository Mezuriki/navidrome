import stylesheet from './mezuriki.css.js'

// Mezuriki theme — modern dark, content-first (Spotify-inspired aesthetic).
// Accent: spotify-green on charcoal black. Designed as the new default.

const green = {
  300: '#1ed760',
  400: '#1db954',
  500: '#1db954',
  700: '#169c46',
  900: '#0a6e30',
}

const gray = {
  bg: '#121212', // app background (deepest)
  surface: '#181818', // cards, paper
  surface2: '#1f1f1f', // elevation / nested surfaces
  hover: '#282828', // hover state
  border: '#2a2a2a',
  text2: '#b3b3b3', // secondary text
  text3: '#72727d', // disabled / tertiary text
}

// Album/Playlist action toolbar — large round green play button (Spotify style)
const musicListActions = {
  padding: '1rem 0',
  alignItems: 'center',
  '@global': {
    button: {
      border: '1px solid transparent',
      backgroundColor: 'inherit',
      color: gray.text2,
      '&:hover': {
        border: `1px solid ${gray.text2}`,
        backgroundColor: 'inherit !important',
      },
    },
    'button:first-child:not(:only-child)': {
      '@media screen and (max-width: 720px)': {
        transform: 'scale(1.5)',
        margin: '1rem',
        '&:hover': { transform: 'scale(1.6) !important' },
      },
      transform: 'scale(2)',
      margin: '1.5rem',
      minWidth: 0,
      padding: 5,
      transition: 'transform .3s ease',
      background: green['500'],
      color: '#fff',
      borderRadius: 500,
      border: 0,
      boxShadow: '0 8px 16px rgba(0, 0, 0, 0.4)',
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${green['300']} !important`,
        border: 0,
      },
    },
    'button:only-child': { margin: '1.5rem' },
    'button:first-child>span:first-child': { padding: 0 },
    'button:first-child>span:first-child>span': { display: 'none' },
    'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
      { color: gray.text2 },
  },
}

export default {
  themeName: 'Mezuriki',
  typography: {
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    h6: { fontSize: '1rem', fontWeight: 600 }, // AppBar title
    h5: { fontWeight: 700 },
    button: { textTransform: 'none', fontWeight: 600 },
  },
  palette: {
    primary: {
      light: green['300'],
      main: green['500'],
      dark: green['700'],
      contrastText: '#fff',
    },
    secondary: {
      main: '#fff',
      contrastText: '#000',
    },
    background: {
      default: gray.bg,
      paper: gray.surface,
    },
    text: {
      primary: '#ffffff',
      secondary: gray.text2,
      disabled: gray.text3,
    },
    divider: 'rgba(255,255,255,0.08)',
    action: {
      hover: 'rgba(255,255,255,0.08)',
      selected: 'rgba(255,255,255,0.12)',
    },
    type: 'dark',
  },
  overrides: {
    // ── Material-UI components ──
    MuiFormGroup: { root: { color: green['500'] } },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
        borderRadius: 4,
        margin: '2px 8px',
        '&:hover': { backgroundColor: gray.hover },
      },
    },
    MuiListItem: {
      button: {
        borderRadius: 4,
      },
    },
    MuiDivider: { root: { margin: '.5rem 0', backgroundColor: 'rgba(255,255,255,0.08)' } },
    MuiButton: {
      root: {
        borderRadius: 500,
        textTransform: 'none',
        fontWeight: 600,
      },
      containedPrimary: {
        background: green['500'],
        color: '#fff',
        boxShadow: 'none',
        '&:hover': { background: `${green['300']} !important` },
      },
      textSecondary: {
        border: `1px solid ${gray.text2}`,
        background: 'transparent',
        color: '#fff',
        '&:hover': {
          border: '1px solid #fff !important',
          background: `${gray.hover} !important`,
        },
      },
      label: { paddingRight: '0.7rem', paddingLeft: '0.5rem' },
    },
    MuiPaper: {
      root: { backgroundColor: gray.surface },
      rounded: { borderRadius: 12 },
    },
    MuiCard: {
      root: {
        backgroundColor: gray.surface,
        borderRadius: 12,
        backgroundImage: 'none',
        boxShadow: 'none',
      },
    },
    MuiChip: {
      root: { backgroundColor: gray.hover, color: gray.text2 },
    },
    MuiDrawer: {
      paper: { backgroundColor: '#000000' },
      root: { backgroundColor: '#000000' },
    },
    MuiTableRow: {
      root: {
        padding: '8px 0',
        transition: 'background-color .2s ease',
        '&:hover': { backgroundColor: `${gray.hover} !important` },
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '1px solid rgba(255,255,255,0.06)',
        padding: '10px !important',
        color: `${gray.text2} !important`,
      },
      head: {
        borderBottom: '1px solid rgba(255,255,255,0.1)',
        fontSize: '0.7rem',
        textTransform: 'uppercase',
        letterSpacing: 1.1,
        color: `${gray.text3} !important`,
        fontWeight: 600,
      },
    },
    MuiAppBar: {
      root: {
        backgroundColor: `${gray.bg}cc !important`,
        backdropFilter: 'blur(12px)',
        boxShadow: 'none',
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      },
      positionFixed: { backgroundColor: `${gray.bg}cc !important`, boxShadow: 'none' },
    },
    MuiFilledInput: {
      root: {
        backgroundColor: gray.hover,
        '&:hover': { backgroundColor: gray.surface2 },
        '&.Mui-focused': { backgroundColor: gray.surface2 },
      },
    },
    MuiOutlinedInput: {
      root: {
        '& .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255,255,255,0.1)' },
        '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255,255,255,0.2)' },
      },
    },
    MuiToolbar: {
      regular: { backgroundColor: 'transparent' },
    },
    MuiIconButton: {
      root: {
        color: gray.text2,
        '&:hover': { backgroundColor: gray.hover, color: '#fff' },
      },
    },
    MuiDialog: {
      paper: { backgroundColor: gray.surface },
    },
    MuiTooltip: {
      tooltip: { backgroundColor: gray.surface2, color: '#fff', fontSize: '0.75rem' },
    },

    // ── Navidrome custom (ND*) components ──
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: '#fff',
        fontSize: '0.9rem',
      },
      albumSubtitle: { color: gray.text2 },
      albumContainer: {
        backgroundColor: gray.surface,
        borderRadius: 8,
        padding: '.75rem',
        transition: 'background-color .25s ease',
        '&:hover': { backgroundColor: gray.hover },
      },
      albumPlayButton: {
        backgroundColor: green['500'],
        color: '#fff',
        borderRadius: '50%',
        boxShadow: '0 8px 16px rgba(0, 0, 0, 0.4)',
        padding: '0.35rem',
        transition: 'all .25s ease',
        '&:hover': {
          background: `${green['300']} !important`,
          padding: '0.45rem',
          transform: 'scale(1.05)',
        },
      },
      tileBar: {
        background:
          'linear-gradient(to top, rgba(0,0,0,0.85) 0%, rgba(0,0,0,0.45) 60%, rgba(0,0,0,0) 100%)',
      },
    },
    NDPlaylistDetails: {
      container: {
        background: 'linear-gradient(rgba(40,40,40,0.6), transparent)',
        borderRadius: 0,
        paddingTop: '2.5rem !important',
        boxShadow: 'none',
      },
      title: { fontSize: 'calc(1.5rem + 1.5vw)', fontWeight: 700, color: '#fff' },
      details: { fontSize: '.875rem', color: gray.text2 },
    },
    NDAlbumDetails: {
      root: {
        background: 'linear-gradient(rgba(40,40,40,0.6), transparent)',
        borderRadius: 0,
        boxShadow: 'none',
      },
      cardContents: { alignItems: 'center', paddingTop: '1.5rem' },
      recordName: { fontSize: 'calc(1rem + 1.5vw)', fontWeight: 700, color: '#fff' },
      recordArtist: { fontSize: '.875rem', fontWeight: 700, color: '#fff' },
      recordMeta: { fontSize: '.875rem', color: gray.text2 },
    },
    NDCollapsibleComment: {
      commentBlock: { fontSize: '.875rem', color: gray.text2 },
    },
    NDAlbumShow: { albumActions: musicListActions },
    NDPlaylistShow: { playlistActions: musicListActions },
    NDArtistShow: {
      actions: {
        padding: '2rem 0',
        alignItems: 'center',
        overflow: 'visible',
        minHeight: '120px',
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: 'inherit',
            color: gray.text2,
            margin: '0 0.5rem',
            '&:hover': {
              border: `1px solid ${gray.text2}`,
              backgroundColor: 'inherit !important',
            },
          },
          'button:first-child>span:first-child>span': { display: 'none' },
          'button:first-child': {
            '@media screen and (max-width: 720px)': {
              transform: 'scale(1.5)',
              margin: '1rem',
              '&:hover': { transform: 'scale(1.6) !important' },
            },
            transform: 'scale(2)',
            margin: '1.5rem',
            minWidth: 0,
            padding: 5,
            transition: 'transform .3s ease',
            background: green['500'],
            color: '#fff',
            borderRadius: 500,
            border: 0,
            boxShadow: '0 8px 16px rgba(0, 0, 0, 0.4)',
            '&:hover': {
              transform: 'scale(2.1)',
              backgroundColor: `${green['300']} !important`,
              border: 0,
            },
          },
          'button:first-child>span:first-child': { padding: 0 },
          'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
            { color: gray.text2 },
        },
      },
      actionsContainer: { overflow: 'visible' },
    },
    NDAudioPlayer: {
      audioTitle: { color: '#fff', fontSize: '0.875rem' },
      songTitle: { fontWeight: 400, color: '#fff' },
      songInfo: { fontSize: '0.675rem', color: gray.text2 },
    },
    NDLogin: {
      main: { boxShadow: 'inset 0 0 0 2000px rgba(0, 0, 0, .8)' },
      systemNameLink: { color: green['500'] },
      card: {
        border: `1px solid ${gray.border}`,
        backgroundColor: `${gray.surface}f0`,
        backdropFilter: 'blur(8px)',
      },
      avatar: { marginBottom: 0 },
    },
    NDSubMenu: {
      header: { color: gray.text3, textTransform: 'uppercase', fontSize: '0.7rem', letterSpacing: 1 },
    },
    NDMenu: {
      // active sidebar item: bright white + bold (green icon comes from primary)
      active: { color: '#fff', fontWeight: 700 },
    },
    NDLayout: {
      root: { backgroundColor: gray.bg },
    },
    NDSongDatagrid: {
      headerStyle: { '& th': { color: gray.text3 } },
    },
    NDAlbumTableView: {},
    NDArtistList: {},
    NDSongList: {},

    // ── react-admin (Ra*) components ──
    RaLayout: {
      content: {
        padding: '0 !important',
        background: `linear-gradient(${gray.surface2}, ${gray.bg})`,
      },
    },
    RaList: { content: { backgroundColor: 'inherit' } },
    RaListToolbar: { toolbar: { padding: '0 .55rem !important' } },
    RaSearchInput: {
      input: {
        paddingLeft: '.9rem',
        border: 0,
        '& .MuiInputBase-root': {
          backgroundColor: `${gray.surface} !important`,
          borderRadius: '20px !important',
          color: '#fff',
          border: `1px solid ${gray.border}`,
          '& fieldset': { borderColor: gray.border },
          '&:hover fieldset': { borderColor: 'rgba(255,255,255,0.2)' },
          '&.Mui-focused fieldset': { borderColor: green['500'] },
          '& svg': { color: `${gray.text2} !important` },
        },
      },
    },
    RaFilter: {
      form: {
        '& .MuiOutlinedInput-input:-webkit-autofill': {
          '-webkit-box-shadow': `0 0 0 100px ${gray.surface2} inset`,
          '-webkit-text-fill-color': '#fff',
        },
      },
    },
    RaFilterButton: { root: { marginRight: '1rem' } },
    RaButton: { button: { margin: '0 5px 0 5px' } },
    RaPaginationActions: {
      currentPageButton: { border: `1px solid ${gray.text2}` },
      button: {
        backgroundColor: 'inherit',
        minWidth: 48,
        margin: '0 4px',
        border: `1px solid ${gray.border}`,
        color: gray.text2,
        '@global': { '> .MuiButton-label': { padding: 0 } },
      },
    },
    RaSidebar: { root: { height: 'initial', backgroundColor: '#000' } },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
