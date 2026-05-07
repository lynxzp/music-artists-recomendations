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
  mobileTab: 'seeds',
  periodData: {},
  artistInfoCache: new Map(),
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
  if (phase === 'editing') App.mobileTab = 'seeds';
  if (phase === 'results') App.mobileTab = 'results';
  renderApp();
}

// === Renderers ===
function renderApp() {
  const app = document.getElementById('app');
  if (!app) return;
  const scrollTops = {};
  for (const sel of ['.gathering-log', '.seeds-scroll']) {
    const el = document.querySelector(sel);
    if (el) scrollTops[sel] = el.scrollTop;
  }
  let html = '';
  html += renderHeader();
  html += '<div class="app-shell">';
  if (App.phase === 'empty') html += renderEmptyState();
  else if (App.phase === 'syncing') html += renderSyncingState();
  else if (App.phase === 'editing' || App.phase === 'gathering' || App.phase === 'results') {
    html += renderControlsRow();
    html += renderEditingLayout();
  }
  html += '</div>';
  if (App.showHelp) html += renderHelpOverlay();
  app.innerHTML = html;
  attachEventListeners();
  for (const sel of ['.gathering-log', '.seeds-scroll']) {
    const el = document.querySelector(sel);
    if (el && sel in scrollTops) el.scrollTop = scrollTops[sel];
  }
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
function renderControlsRow() {
  const visibleCount = App.results.filter(r => !App.hidden.includes(r.name)).length;
  const subtitle = {
    editing: 'Tune weights, then run.',
    gathering: 'Gathering similar artists…',
    results: visibleCount + ' of 150 candidates · top of ' + (App.seeds.length * 300) + ' lookups'
  }[App.phase] || '';

  return '<div class="stat-row">' +
    '<div style="grid-column:1 / span 5">' +
      '<h1 class="stat-h1">Recommendations</h1>' +
      '<div class="stat-subtitle">' + escapeHtml(subtitle) + '</div>' +
    '</div>' +
    '<div style="grid-column:7 / span 2">' + renderStat('Seeds', App.seeds.length) + '</div>' +
    '<div style="grid-column:9 / span 2">' + renderStat('Candidates', App.phase === 'results' ? visibleCount : '—') + '</div>' +
    '<div style="grid-column:11 / span 2">' + renderStat('Hidden', App.hidden.length) + '</div>' +
  '</div>';
}

function renderStat(label, value) {
  return '<div>' +
    '<div class="stat-block-label">' + escapeHtml(label) + '</div>' +
    '<div class="stat-block-value">' + (value !== undefined ? value : '—') + '</div>' +
  '</div>';
}

function renderResultsPanel() {
  const visible = getVisibleResults();
  const isMobile = window.innerWidth <= 768;

  if (isMobile) {
    return renderResultsCards(visible);
  }
  return renderResultsTable(visible);
}

function getVisibleResults() {
  const hiddenSet = new Set(App.hidden);
  let out = App.results.filter(r => !hiddenSet.has(r.name.toLowerCase()));
  out.sort((a, b) => b.total - a.total);
  return out.slice(0, DISPLAY_LIMIT);
}

function renderResultsTable(visible) {
  const maxTotal = visible.length > 0 ? visible[0].total : 1;
  let rows = '';
  for (let i = 0; i < visible.length; i++) {
    rows += renderRow(visible[i], i + 1, maxTotal);
  }

  return '<div>' +
    '<div class="results-header-bar">' +
      '<div class="section-head"><div class="section-head-label">03 · Results (' + visible.length + ')</div></div>' +
    '</div>' +
    '<table class="results-table">' +
      '<thead><tr>' +
        '<th class="col-rank">#</th>' +
        '<th>Artist</th>' +
        '<th class="col-genres">Genres</th>' +
        '<th class="col-similar">Similar to</th>' +
        '<th class="col-score">Score</th>' +
      '</tr></thead>' +
      '<tbody>' + rows + '</tbody>' +
    '</table>' +
    (visible.length === 0 ? '<div class="no-results">No matches.</div>' : '') +
  '</div>';
}

function renderRow(rec, rank, maxTotal) {
  const genres = (rec.genres || []).map(g =>
    '<span class="genre-dot"><span class="genre-dot-color" style="background:' + genreColor(g) + '"></span>' + escapeHtml(g) + '</span>'
  ).join('');

  const topMatches = Object.entries(rec.matches || {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)
    .filter(([, val]) => val > 0)
    .map(([name]) => escapeHtml(name));

  const similarTo = topMatches.length > 0 ? topMatches.join(', ') : '';
  const barWidth = maxTotal > 0 ? Math.round((rec.total / maxTotal) * 40) : 0;
  const isSelected = App.selected === rec.name;

  let html = '<tr data-artist="' + escapeHtml(rec.name) + '" class="' + (isSelected ? 'selected' : '') + '" onclick="App.selectRow(\'' + escapeHtml(rec.name).replace(/\\/g, '\\\\').replace(/'/g, "\\'") + '\')">' +
    '<td class="col-rank">' + String(rank).padStart(3, '0') + '</td>' +
    '<td class="col-name">' + escapeHtml(rec.name) +
      '<button class="hide-btn" onclick="event.stopPropagation();App.hideArtist(\'' + escapeHtml(rec.name).replace(/\\/g, '\\\\').replace(/'/g, "\\'") + '\')">hide</button></td>' +
    '<td class="col-genres"><div class="genre-dot-list">' + genres + '</div></td>' +
    '<td class="col-similar">' + similarTo + '</td>' +
    '<td class="col-score">' + rec.total.toFixed(1) +
      '<div class="score-bar-bg"><div class="score-bar-fill" style="width:' + barWidth + 'px"></div></div>' +
    '</td>' +
  '</tr>';

  if (isSelected) {
    html += renderExpandedRow(rec, maxTotal);
  }
  return html;
}

function renderExpandedRow(rec, maxTotal) {
  const entries = Object.entries(rec.matches || {})
    .sort((a, b) => b[1] - a[1])
    .filter(([, val]) => val > 0);

  let rows = '';
  for (const [seedName, contrib] of entries) {
    const seed = App.seeds.find(s => s.name === seedName);
    const raw = (rec.rawMatches || {})[seedName] || 0;
    const weight = seed ? seed.weight : 0;
    const barW = maxTotal > 0 ? Math.round((contrib / maxTotal) * 80) : 0;
    rows += '<tr>' +
      '<td>' + escapeHtml(seedName) + '</td>' +
      '<td class="mono">' + (raw * 100).toFixed(0) + '%</td>' +
      '<td class="mono">× ' + weight + '</td>' +
      '<td class="total">' + contrib.toFixed(1) + '</td>' +
      '<td><div class="breakdown-bar-bg"><div class="breakdown-bar-fill" style="width:' + barW + 'px"></div></div></td>' +
    '</tr>';
  }

  return '<tr class="expanded-row"><td colspan="5">' +
    '<div style="padding:8px 0">' +
      '<div class="breakdown-reason">Similarity breakdown</div>' +
      '<table class="breakdown-table">' +
        '<thead><tr><th>Seed</th><th>Sim</th><th>Weight</th><th>Contrib</th><th></th></tr></thead>' +
        '<tbody>' + rows + '</tbody>' +
      '</table>' +
    '</div>' +
  '</td></tr>';
}

function renderResultsCards(visible) {
  const maxTotal = visible.length > 0 ? visible[0].total : 1;
  let cards = '';
  for (let i = 0; i < visible.length; i++) {
    const rec = visible[i];
    const genres = (rec.genres || []).map(g =>
      '<span class="genre-dot"><span class="genre-dot-color" style="background:' + genreColor(g) + '"></span>' + escapeHtml(g) + '</span>'
    ).join('');
    cards += '<div class="result-card">' +
      '<div style="display:flex;justify-content:space-between;align-items:baseline">' +
        '<span style="font-weight:500;font-size:16px">' + escapeHtml(rec.name) + '</span>' +
        '<span style="font-family:' + swiss.mono + ';font-size:11px;color:' + swiss.muted + '">' + rec.total.toFixed(1) + '</span>' +
      '</div>' +
      '<div class="genre-dot-list" style="margin-top:4px">' + genres + '</div>' +
    '</div>';
  }
  return '<div>' + cards + '</div>';
}

function renderEditingLayout() {
  const isMobile = window.innerWidth <= 768;
  const bottomContent = App.phase === 'gathering' ? renderGatheringPanel() :
                        App.phase === 'results' ? renderResultsPanel() :
                        '';
  if (isMobile) {
    return '<div class="tab-bar" role="tablist">' +
      '<button role="tab" aria-selected="' + (App.mobileTab === 'seeds' ? 'true' : 'false') + '" onclick="App.setMobileTab(\'seeds\')">Seeds</button>' +
      '<button role="tab" aria-selected="' + (App.mobileTab === 'results' ? 'true' : 'false') + '" onclick="App.setMobileTab(\'results\')">Results</button>' +
    '</div>' +
    '<div class="tab-pane ' + (App.mobileTab === 'seeds' ? 'is-active' : '') + '">' + renderSeedsPanel() + '</div>' +
    '<div class="tab-pane ' + (App.mobileTab === 'results' ? 'is-active' : '') + '">' + bottomContent + '</div>';
  }
  return '<div class="two-col">' +
    '<div>' + renderSeedsPanel() + '</div>' +
    '<div>' + renderSidePanel() + '</div>' +
  '</div>' +
  (bottomContent ? '<div class="bottom-row">' + bottomContent + '</div>' : '');
}

App.setMobileTab = function(tab) {
  App.mobileTab = tab;
  renderApp();
};

function renderSeedsPanel() {
  let rows = '';
  if (App.seeds.length === 0) {
    rows = '<div class="seeds-empty">No seeds yet — add some below.</div>';
  } else {
    rows = App.seeds.map((sd, i) =>
      '<div class="seed-row">' +
        '<div class="seed-idx">' + String(i + 1).padStart(2, '0') + '</div>' +
        '<div class="seed-name">' + escapeHtml(sd.name) + '</div>' +
        '<div class="seed-plays">' + (sd.plays ? sd.plays.toLocaleString() : '—') + '</div>' +
        '<div class="seed-weight">' +
          '<input type="number" data-seed-idx="' + i + '" value="' + sd.weight + '" min="0">' +
          '<button class="seed-remove" onclick="App.removeSeed(' + i + ')">×</button>' +
        '</div>' +
      '</div>'
    ).join('');
  }

  return '<div>' +
    '<div class="section-head">' +
      '<div class="section-head-label">01 · Seeds</div>' +
      (App.lastfmUser ? '<div style="display:flex;gap:8px;align-items:center;font-size:11px;color:' + swiss.muted + '">' +
        '<span style="font-family:' + swiss.mono + '">last.fm · @' + escapeHtml(App.lastfmUser) + '</span>' +
        '<button class="btn btn-sm" onclick="App.resync()">Resync</button></div>' : '') +
    '</div>' +
    '<div class="seeds-panel">' +
      '<div class="seeds-header">' +
        '<div>#</div><div>Artist</div><div>Plays</div><div>Weight</div>' +
      '</div>' +
      '<div class="seeds-scroll">' + rows + '</div>' +
    '</div>' +
    '<div class="add-seed-form">' +
      '<input type="text" id="add-seed-input" placeholder="Add artist…">' +
      '<button class="btn" onclick="App.addSeedFromInput()">Add</button>' +
    '</div>' +
    '<div class="seeds-actions">' +
      '<button class="btn btn-primary" onclick="App.go()" ' +
        'style="flex:1" ' + (App.seeds.length === 0 ? 'disabled' : '') + '>▶ Find similar</button>' +
    '</div>' +
  '</div>';
}

function renderSidePanel() {
  const hiddenChips = App.hidden.length === 0 ? '' :
    '<div class="hidden-chips">' +
      '<div class="hidden-chips-label">Hidden (' + App.hidden.length + ')</div>' +
      '<div class="hidden-chip-list">' +
        App.hidden.map(n => '<span class="hidden-chip" onclick="App.unhideArtist(\'' + escapeHtml(n).replace(/\\/g, '\\\\').replace(/'/g, "\\'") + '\')">' + escapeHtml(n) + '</span>').join('') +
      '</div>' +
    '</div>';

  return '<div>' +
    '<div class="section-head"><div class="section-head-label">02 · Meta</div></div>' +
    '<div class="side-meta">' +
      renderMeta('last run', App.phase === 'results' ? 'just now' : '—') +
      renderMeta('candidates', App.phase === 'results' ? App.results.length + ' (of ' + (App.seeds.length * 300) + ' raw)' : '—') +
      renderMeta('seed pool', (App.seeds.length * 300) + ' lookups') +
      renderMeta('compute', App.phase === 'results' ? App.computeTime + 's' : '—') +
    '</div>' +
    hiddenChips +
  '</div>';
}

function renderMeta(k, v) {
  return '<div class="meta-row"><span>' + escapeHtml(k) + '</span><span>' + escapeHtml(v) + '</span></div>';
}

function renderGatheringPanel() {
  const pct = (App.progress.done / Math.max(1, App.progress.total)) * 100;
  let log = '';
  for (let i = 0; i < App.seeds.length; i++) {
    const sd = App.seeds[i];
    const entry = App.progress.log.find(x => x.seed === sd.name);
    if (!entry && i > App.progress.done) continue;
    const status = entry ? (entry.failed ? '<span class="log-status warn">⚠</span>' : '<span class="log-status">✓</span>') : '<span class="log-pending">→</span>';
    log += '<div class="gathering-log-row">' +
      status +
      '<span>' + escapeHtml(sd.name) + '</span>' +
      '<span>' + (entry ? (entry.count + ' similar') : 'fetching…') + '</span>' +
      '<span>' + (entry ? ('+' + entry.novel + ' new') : '') + '</span>' +
    '</div>';
  }

  return '<div class="gathering-panel">' +
    '<div class="gathering-header">' +
      '<div class="section-head"><div class="section-head-label">Gathering similar artists</div></div>' +
      '<div class="gathering-counter">' + App.progress.done + '/' + App.progress.total + '</div>' +
    '</div>' +
    '<div class="progress-track"><div class="progress-fill" style="width:' + pct + '%"></div></div>' +
    '<div class="gathering-log">' + log + '</div>' +
  '</div>';
}
function renderHelpOverlay() {
  const shortcuts = [
    ['j / ↓', 'Move down'],
    ['k / ↑', 'Move up'],
    ['e', 'Toggle row expand'],
    ['x', 'Hide selected artist'],
    ['/', 'Focus add-artist input'],
    ['?', 'Toggle this dialog'],
    ['esc', 'Close dialog'],
  ];
  return '<div class="help-overlay" onclick="App.toggleHelp()">' +
    '<div class="help-card" onclick="event.stopPropagation()">' +
      '<div class="help-card-header">' +
        '<h3>Keyboard shortcuts</h3>' +
        '<button class="icon-btn" onclick="App.toggleHelp()">×</button>' +
      '</div>' +
      '<table>' +
        shortcuts.map(([k, v]) =>
          '<tr><td><kbd>' + escapeHtml(k) + '</kbd></td><td style="color:' + swiss.muted + ';padding-left:18px">' + escapeHtml(v) + '</td></tr>'
        ).join('') +
      '</table>' +
    '</div>' +
  '</div>';
}

// === Actions ===
App.toggleHelp = function() { App.showHelp = !App.showHelp; renderApp(); };

App.skipSync = function() {
  App.lastfmUser = '';
  App.seeds = [];
  App.error = null;
  setPhase('editing');
};

App.addSeedFromInput = function() {
  const input = document.getElementById('add-seed-input');
  if (!input) return;
  const name = input.value.trim();
  if (!name) return;
  const key = name.toLowerCase();
  if (App.seeds.some(s => s.name.toLowerCase() === key)) {
    input.value = ''; return; // duplicate
  }
  App.seeds.push({ name: name, plays: 0, weight: 100 });
  input.value = '';
  saveState();
  renderApp();
};

App.removeSeed = function(idx) {
  App.seeds.splice(idx, 1);
  saveState();
  renderApp();
};

App.updateSeedWeight = function(idx, weight) {
  const w = parseInt(weight, 10) || 0;
  if (App.seeds[idx]) {
    App.seeds[idx].weight = w;
    saveState();
  }
};

App.resync = function() {
  if (App.lastfmUser) doSync(App.lastfmUser);
  else setPhase('empty');
};

App.go = async function() {
  if (App.seeds.length === 0) return;
  App.artistInfoCache = new Map();
  App.progress = { done: 0, total: App.seeds.length, log: [] };
  App.computeTime = null;
  const startTime = Date.now();
  setPhase('gathering');

  const allSimilar = new Map();
  const failed = [];

  for (let i = 0; i < App.seeds.length; i++) {
    const seed = App.seeds[i];
    try {
      const data = await fetchWithRetry(
        './api/artist/similar?artist=' + encodeURIComponent(seed.name),
        60,
        (msg) => {
          const log = App.progress.log.find(x => x.seed === seed.name);
          if (log) log.retryMsg = msg;
          renderApp();
        }
      );
      const artists = data.data.artists || [];
      let novel = 0;
      for (const similar of artists) {
        if (!allSimilar.has(similar.name)) {
          allSimilar.set(similar.name, {
            name: similar.name,
            match: similar.match,
            matches: {},
            rawMatches: {},
            total: 0,
            genres: []
          });
          novel++;
        }
        const entry = allSimilar.get(similar.name);
        const weightedMatch = parseFloat(similar.match) * Math.pow(seed.weight, 0.8);
        entry.matches[seed.name] = weightedMatch;
        entry.rawMatches[seed.name] = parseFloat(similar.match) || 0;
        entry.total += weightedMatch;
      }
      App.progress.log.push({ seed: seed.name, count: artists.length, novel: novel, failed: false });
    } catch (err) {
      console.error('Error fetching', seed.name, err);
      failed.push(seed.name);
      App.progress.log.push({ seed: seed.name, count: 0, novel: 0, failed: true });
    }
    App.progress.done = i + 1;
    renderApp();
  }

  App.computeTime = ((Date.now() - startTime) / 1000).toFixed(1);

  App.results = Array.from(allSimilar.values()).sort((a, b) => b.total - a.total).slice(0, DISPLAY_LIMIT);

  populateArtistInfo();

  setPhase('results');
  saveState();
};

async function populateArtistInfo() {
  const batchSize = 10;
  for (let i = 0; i < App.results.length; i += batchSize) {
    const batch = App.results.slice(i, i + batchSize);
    await Promise.all(batch.map(async (rec) => {
      const info = await fetchArtistInfo(rec.name, App.lastfmUser);
      if (info && info.tags && info.tags.tag) {
        rec.genres = info.tags.tag.slice(0, 5).map(t => t.name);
      }
    }));
    if (App.phase === 'results') renderApp();
  }
}

App.selectRow = function(name) {
  App.selected = (App.selected === name) ? null : name;
  renderApp();
};

App.hideArtist = function(name) {
  const hidden = getHiddenArtists();
  const key = name.toLowerCase();
  if (!hidden.includes(key)) {
    hidden.push(key);
    setHiddenArtists(hidden);
  }
  App.hidden = getHiddenArtists();
  renderApp();
};

App.unhideArtist = function(name) {
  const hidden = getHiddenArtists().filter(n => n !== name.toLowerCase());
  setHiddenArtists(hidden);
  App.hidden = getHiddenArtists();
  renderApp();
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
  const addInput = document.getElementById('add-seed-input');
  if (addInput) {
    addInput.addEventListener('keydown', function(e) {
      if (e.key === 'Enter') { e.preventDefault(); App.addSeedFromInput(); }
    });
  }
  document.querySelectorAll('.seed-row input').forEach(function(inp) {
    inp.addEventListener('change', function() {
      const idx = parseInt(inp.dataset.seedIdx, 10);
      App.updateSeedWeight(idx, inp.value);
    });
  });
}

// === Keyboard ===
document.addEventListener('keydown', function(e) {
  if (App.showHelp) {
    if (e.key === 'Escape' || e.key === '?') { App.showHelp = false; renderApp(); }
    return;
  }
  if (e.target.matches('input,textarea')) {
    if (e.key === 'Escape') { e.target.blur(); e.preventDefault(); }
    return;
  }
  if (e.key === '?') { e.preventDefault(); App.toggleHelp(); return; }
  if (App.phase !== 'results') return;

  const visible = getVisibleResults();
  if (e.key === 'j' || e.key === 'ArrowDown') {
    e.preventDefault();
    const idx = visible.findIndex(r => r.name === App.selected);
    const next = idx < 0 ? 0 : Math.min(visible.length - 1, idx + 1);
    App.selected = visible[next] ? visible[next].name : null;
    renderApp();
  }
  if (e.key === 'k' || e.key === 'ArrowUp') {
    e.preventDefault();
    const idx = visible.findIndex(r => r.name === App.selected);
    const next = idx <= 0 ? 0 : idx - 1;
    App.selected = visible[next] ? visible[next].name : null;
    renderApp();
  }
  if (e.key === 'e') {
    e.preventDefault();
    if (App.selected) {
      App.selected = null;
      renderApp();
    }
  }
  if (e.key === '/') {
    e.preventDefault();
    const input = document.getElementById('add-seed-input');
    if (input) input.focus();
  }
  if (e.key === 'x' && App.selected) {
    e.preventDefault();
    App.hideArtist(App.selected);
  }
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
