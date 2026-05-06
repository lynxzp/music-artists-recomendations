// === Constants ===
const PERIODS = [
  { key: 'overall', label: 'Overall' },
  { key: '12month', label: '12 Month' },
  { key: '1month', label: '1 Month' }
];
const DISPLAY_LIMIT = 150;

const swiss = {
  bg: '#ffffff', ink: '#0a0a0a', muted: '#7a7a7a',
  line: '#e4e4e4', hairline: '#f0f0f0', panel: '#fafafa',
  warn: '#c94a2b', ok: '#2b7a4a',
  font: '"Helvetica Neue", Helvetica, Arial, sans-serif',
  mono: '"JetBrains Mono", "SF Mono", Menlo, monospace',
};

const GENRE_COLORS = {
  electronic:'#6b7ea8','art rock':'#a87455','indie rock':'#7a8c5f',
  'dream pop':'#b5869c',shoegaze:'#8c7bb2','post-punk':'#5d5d5d',
  'trip hop':'#8c5f5f',folk:'#a8925f','post-rock':'#6ba89c',
  ambient:'#a8a29e','uk bass':'#5f7ba8','post-dubstep':'#5e6d88',
  '4ad':'#b5869c',idm:'#7b8fa2',downtempo:'#9c8c6b',soul:'#b5705f',
  'art pop':'#b58a9c',indie:'#7a8c5f',experimental:'#9b7b9e',instrumental:'#8a8a85',
};

function genreColor(g) {
  return GENRE_COLORS[(g || '').toLowerCase()] || '#888';
}

// === State ===
const App = {
  phase: 'empty',
  seeds: [],
  results: [],
  hidden: [],
  lastfmUser: '',
  error: null,
  progress: { done: 0, total: 0, log: [] },
  selected: null,
  sortKey: 'rank',
  filter: '',
  mobileTab: 'seeds',
  periodData: {},
  artistInfoCache: new Map(),
  autoHiddenArtists: [],
  unexcludedArtists: new Set(),
  lastRenderArgs: null,
};

// === API ===
async function fetchWithRetry(url, maxRetries, statusCallback) {
  maxRetries = maxRetries || 60;
  const delay = 100;
  let lastError;
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      const resp = await fetch(url);
      if (resp.ok) return await resp.json();
      lastError = new Error('HTTP ' + resp.status);
    } catch (err) {
      lastError = err;
    }
    if (attempt < maxRetries) {
      if (statusCallback) statusCallback('Retry ' + (attempt + 1) + '/' + maxRetries + '...');
      await new Promise(r => setTimeout(r, delay));
    }
  }
  throw lastError;
}

async function fetchArtistInfo(name, user) {
  const cacheKey = name + '\0' + (user || '');
  if (App.artistInfoCache.has(cacheKey)) return App.artistInfoCache.get(cacheKey);
  try {
    let url = './api/artist/info?artist=' + encodeURIComponent(name);
    if (user) url += '&user=' + encodeURIComponent(user);
    const data = await fetchWithRetry(url);
    const info = data.data.artist;
    App.artistInfoCache.set(cacheKey, info);
    return info;
  } catch (err) {
    console.error('Error fetching artist info', name, err);
    return null;
  }
}

// === Utils ===
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function getHiddenKey() {
  return 'hiddenArtists_' + (App.lastfmUser || '').toLowerCase();
}

function getHiddenArtists() {
  try { return JSON.parse(localStorage.getItem(getHiddenKey()) || '[]'); }
  catch { return []; }
}

function setHiddenArtists(list) {
  localStorage.setItem(getHiddenKey(), JSON.stringify(list));
}

function saveState() {
  localStorage.setItem('savedUsername', App.lastfmUser);
  localStorage.setItem('savedArtists', JSON.stringify(App.seeds.map(s => ({ name: s.name, weight: s.weight }))));
}

function restoreState() {
  const user = localStorage.getItem('savedUsername');
  if (user) App.lastfmUser = user;
  try {
    const artists = JSON.parse(localStorage.getItem('savedArtists') || '[]');
    if (artists.length > 0) {
      App.seeds = artists.map(a => ({ name: a.name, plays: a.weight || 0, weight: a.weight || 0 }));
      App.phase = 'editing';
    }
  } catch { /* ignore */ }
  App.hidden = getHiddenArtists();
}

function formatTags(tags, limit) {
  limit = limit || 5;
  if (!tags || !tags.tag || tags.tag.length === 0) return '';
  return tags.tag.slice(0, limit).map(t => '<span class="genre-dot">' +
    '<span class="genre-dot-color" style="background:' + genreColor(t.name) + '"></span>' +
    escapeHtml(t.name) + '</span>').join('');
}

function setPhase(phase) {
  App.phase = phase;
  renderApp();
}

// === Renderers ===
function renderApp() {
  const app = document.getElementById('app');
  if (!app) return;
  let html = '';
  html += renderHeader();
  html += '<div class="app-shell">';
  if (App.phase === 'empty') html += renderEmptyState();
  else if (App.phase === 'syncing') html += renderSyncingState();
  else if (App.phase === 'editing' || App.phase === 'gathering' || App.phase === 'results') {
    html += renderControlsRow();
    if (App.phase === 'gathering') html += renderGatheringPanel();
    else html += renderEditingLayout();
  }
  html += '</div>';
  if (App.showHelp) html += renderHelpOverlay();
  app.innerHTML = html;
  attachEventListeners();
}

function renderHeader() {
  const user = App.lastfmUser;
  return '<header class="app-header">' +
    '<div class="app-header-left">' +
      '<div class="app-header-breadcrumb">/ Music · Artist · Recommendations</div>' +
      '<span style="width:4px;height:4px;border-radius:50%;background:' + swiss.muted + '"></span>' +
      '<div class="app-header-version">v0.4.2</div>' +
    '</div>' +
    '<div class="app-header-right">' +
      (user ? '<div class="app-header-pill">' +
        '<span class="pill-dot"></span><span>last.fm</span>' +
        '<span class="pill-user">@' + escapeHtml(user) + '</span></div>' : '') +
      '<button class="btn btn-sm" onclick="App.toggleHelp()" title="Keyboard shortcuts">?</button>' +
    '</div>' +
  '</header>';
}

function renderEmptyState() {
  const error = App.error ? '<div class="inline-error">⚠ ' + escapeHtml(App.error) + '</div>' : '';
  return '<div class="empty-state">' +
    '<h1 class="empty-h1">Recommendations</h1>' +
    '<p class="empty-hint">Enter a last.fm username to bootstrap your seed list, or skip and add artists manually.</p>' +
    '<form class="empty-form" id="empty-form" onsubmit="return false">' +
      '<input type="text" id="empty-username" placeholder="last.fm username" autofocus>' +
      '<button type="submit" class="btn btn-primary">Sync →</button>' +
    '</form>' +
    '<button class="empty-skip" onclick="App.skipSync()">Skip — start from scratch</button>' +
    error +
  '</div>';
}

function renderSyncingState() {
  return '<div class="syncing-state">' +
    '<div class="syncing-label">syncing last.fm</div>' +
    '<div style="display:flex;align-items:center;justify-content:center;gap:12px">' +
      '<span class="dot-pulse"><span></span><span></span><span></span></span>' +
      '<span>fetching @' + escapeHtml(App.lastfmUser) + '&#39;s top artists…</span>' +
    '</div>' +
  '</div>';
}

// Placeholders for phases not yet implemented
function renderControlsRow() { return ''; }
function renderEditingLayout() { return ''; }
function renderGatheringPanel() { return ''; }
function renderHelpOverlay() { return ''; }

// === Actions ===
App.toggleHelp = function() { App.showHelp = !App.showHelp; renderApp(); };

App.skipSync = function() {
  App.lastfmUser = '';
  App.seeds = [];
  App.error = null;
  setPhase('editing');
};

async function doSync(username) {
  App.lastfmUser = username;
  App.error = null;
  App.periodData = {};
  setPhase('syncing');

  for (const p of PERIODS) {
    try {
      const data = await fetchWithRetry(
        './api/user/top-artists?user=' + encodeURIComponent(username) + '&period=' + p.key
      );
      App.periodData[p.key] = (data.data.artists || [])
        .map(a => ({ name: a.name, playcount: parseInt(a.playcount, 10) || 0 }))
        .filter(a => a.playcount >= 5);
    } catch (err) {
      console.error('Error loading ' + p.key, err);
      App.periodData[p.key] = [];
    }
  }

  // Aggregate playcounts
  const totals = new Map();
  const names = new Map();
  for (const period of PERIODS) {
    const artists = App.periodData[period.key] || [];
    for (const a of artists) {
      const key = a.name.toLowerCase();
      totals.set(key, (totals.get(key) || 0) + a.playcount);
      if (!names.has(key)) names.set(key, a.name);
    }
  }
  App.seeds = Array.from(totals.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([key, weight]) => ({ name: names.get(key), plays: weight, weight: weight }));

  if (App.seeds.length === 0) {
    App.error = 'No artists found for user ' + username;
  }
  setPhase('editing');
  saveState();
}

// === Event wiring ===
function attachEventListeners() {
  const form = document.getElementById('empty-form');
  if (form) {
    form.addEventListener('submit', function(e) {
      e.preventDefault();
      const user = document.getElementById('empty-username').value.trim();
      if (user) doSync(user);
    });
  }
}

// === Keyboard ===
document.addEventListener('keydown', function(e) {
  if (App.showHelp && e.key === 'Escape') { App.showHelp = false; renderApp(); return; }
  if (e.target.matches('input,textarea')) {
    if (e.key === 'Escape') { e.target.blur(); e.preventDefault(); }
    return;
  }
  if (e.key === '?') { e.preventDefault(); App.toggleHelp(); }
});

// === Init ===
function init() {
  restoreState();
  renderApp();
}
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
