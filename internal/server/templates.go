package server

func indexHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="Deep music recommendations based on your Last.fm profile. Analyzes your top artists and finds similar ones with weighted scoring.">
    <meta property="og:title" content="Music Artist Recommendations">
    <meta property="og:type" content="website">
    <meta property="og:url" content="https://sergua.com/music/artists-recomendations/">
    <meta property="og:description" content="Deep music recommendations based on your Last.fm profile. Analyzes your top artists and finds similar ones with weighted scoring.">
    <meta property="og:image" content="https://sergua.com/static/music-artists-recomendations.png">
    <meta property="og:site_name" content="SergUA">
    <link rel="canonical" href="https://sergua.com/music/artists-recomendations/">
    <link rel="icon" href="/static/favicon.ico" sizes="any">
    <link rel="apple-touch-icon" href="/static/apple-touch-icon.png">
    <title>Music Artist Recommendations - SergUA</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            margin: 0;
            padding: 24px;
            display: flex;
            gap: 24px;
            height: 100vh;
            background: #f8f9fa;
            color: #2d3748;
            line-height: 1.5;
        }
        .left-panel {
            width: 550px;
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            gap: 14px;
            background: #fff;
            padding: 24px;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.05);
        }
        .right-panel {
            flex: 1;
            background: #fff;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.05);
            padding: 20px;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        .table-scroll {
            overflow: auto;
            flex: 1;
        }
        .period-table-container {
            flex: 1;
            min-height: 0;
            overflow: auto;
        }
        .period-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 12px;
        }
        .period-table th {
            background: #f1f5f9;
            padding: 8px 10px;
            text-align: left;
            font-weight: 600;
            color: #475569;
            border-bottom: 2px solid #e2e8f0;
        }
        .period-table td {
            padding: 4px 8px;
            border-bottom: 1px solid #f1f5f9;
            vertical-align: top;
        }
        .artist-cell {
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 8px;
        }
        .artist-name {
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 60px;
        }
        .period-col { width: 100px; }
        .period-col .artist-name { max-width: 50px; font-size: 11px; }
        .total-col { min-width: 200px; }
        .editable-artist {
            display: flex;
            align-items: center;
            gap: 4px;
        }
        .artist-input {
            width: 100px;
            padding: 4px 6px;
            font-size: 12px;
            border: 1px solid #e2e8f0;
            border-radius: 4px;
        }
        .weight-input {
            width: 55px;
            padding: 4px 6px;
            font-size: 12px;
            border: 1px solid #e2e8f0;
            border-radius: 4px;
            text-align: right;
            -moz-appearance: textfield;
        }
        .weight-input::-webkit-outer-spin-button,
        .weight-input::-webkit-inner-spin-button {
            -webkit-appearance: none;
            margin: 0;
        }
        .remove-btn {
            padding: 2px 6px;
            cursor: pointer;
            background: #fee2e2;
            border: 1px solid #fecaca;
            border-radius: 4px;
            color: #dc2626;
            font-size: 12px;
            line-height: 1;
        }
        .remove-btn:hover {
            background: #fecaca;
        }
        .add-row-btn {
            width: 100%;
            margin-top: 8px;
            padding: 6px 12px;
            background: #f0fdf4;
            border: 1px dashed #86efac;
            color: #16a34a;
            font-size: 12px;
        }
        .add-row-btn:hover {
            background: #dcfce7;
            border-color: #4ade80;
        }
        .playcount {
            color: #94a3b8;
            font-size: 11px;
            font-variant-numeric: tabular-nums;
        }
        .multi-period {
            background: #fef3c7;
        }
        .multi-period .artist-name {
            font-weight: 500;
            color: #92400e;
        }
        input, button {
            padding: 10px 12px;
            font-size: 14px;
            border-radius: 8px;
            border: 1px solid #e2e8f0;
            transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
        }
        input:focus {
            outline: none;
            border-color: #6366f1;
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
        }
        button {
            cursor: pointer;
            background: #f8fafc;
            font-weight: 500;
        }
        button:hover {
            background: #f1f5f9;
            border-color: #cbd5e1;
        }
        #goBtn {
            background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
            color: white;
            border: none;
            padding: 14px 16px;
            font-size: 15px;
            font-weight: 600;
            margin-top: 8px;
            box-shadow: 0 2px 4px rgba(99, 102, 241, 0.3);
        }
        #goBtn:hover {
            background: linear-gradient(135deg, #4f46e5 0%, #4338ca 100%);
            box-shadow: 0 4px 8px rgba(99, 102, 241, 0.4);
        }
        #goBtn:disabled {
            background: #cbd5e1;
            box-shadow: none;
            cursor: not-allowed;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            font-size: 13px;
        }
        th, td {
            border: 1px solid #e5e7eb;
            padding: 10px 14px;
            text-align: left;
        }
        th {
            background: #f8fafc;
            position: sticky;
            top: -1px;
            z-index: 1;
            font-weight: 600;
            color: #475569;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        tbody tr:hover {
            background: #f8fafc;
        }
        td.match {
            text-align: right;
            font-variant-numeric: tabular-nums;
            color: #64748b;
        }
        .total {
            font-weight: 600;
            background: #f0f9ff;
            color: #0369a1;
        }
        tbody tr:hover .total {
            background: #e0f2fe;
        }
        #status {
            color: #64748b;
            font-size: 13px;
            min-height: 20px;
        }
        .username-section {
            display: flex;
            gap: 8px;
        }
        .username-section input { flex: 1; }
        .username-section button {
            padding: 10px 16px;
            background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
            color: white;
            border: none;
            font-weight: 600;
            box-shadow: 0 2px 4px rgba(99, 102, 241, 0.3);
        }
        .username-section button:hover {
            background: linear-gradient(135deg, #4f46e5 0%, #4338ca 100%);
            box-shadow: 0 4px 8px rgba(99, 102, 241, 0.4);
        }
        .section {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }
        .section.grow {
            flex: 1;
            min-height: 0;
        }
        .section-label {
            font-size: 13px;
            font-weight: 500;
            color: #475569;
        }
        .section-hint {
            font-size: 12px;
            color: #94a3b8;
            font-style: italic;
        }
        .right-panel h3 {
            margin: 0 0 16px 0;
            font-size: 18px;
            font-weight: 600;
            color: #1a202c;
        }

        @media (max-width: 768px) {
            body {
                flex-direction: column;
                height: auto;
                min-height: 100vh;
                padding: 12px;
                gap: 16px;
            }

            .left-panel {
                width: 100%;
                max-height: none;
            }

            .right-panel {
                width: 100%;
                min-height: 400px;
            }

            .period-table .period-col {
                width: 80px;
            }

            .total-col {
                min-width: 150px;
            }
        }

        .period-col.hidden, .match-col.hidden, .total-col-header.hidden { display: none; }
        .explain-btn {
            padding: 6px 12px;
            font-size: 12px;
            background: #f1f5f9;
            border: 1px solid #e2e8f0;
            color: #64748b;
            border-radius: 6px;
            cursor: pointer;
        }
        .explain-btn:hover {
            background: #e2e8f0;
            color: #475569;
        }
        .explain-text {
            font-size: 12px;
            color: #64748b;
            background: #f8fafc;
            padding: 8px 12px;
            border-radius: 6px;
            margin-top: 8px;
            display: none;
        }
        .explain-text.visible { display: block; }
        .period-table.collapsed .total-col { width: 100%; }
        .period-table.collapsed .artist-input { width: 200px; }
        .results-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }
        .results-header h3 { margin: 0; }
        .similar-to {
            font-size: 11px;
            color: #94a3b8;
            margin-top: 2px;
        }
        .artist-tags { font-size: 10px; color: #64748b; margin-top: 2px; }
        .tag { background: #f1f5f9; padding: 1px 5px; border-radius: 3px; margin-right: 3px; white-space: nowrap; }
    </style>
</head>
<body>
    <div class="left-panel">
        <div class="section">
            <label class="section-label">Get top artists from Last.fm user:</label>
            <div class="username-section">
                <input type="text" id="username" placeholder="Last.fm username">
                <button onclick="loadUserArtists()">Load</button>
            </div>
        </div>
        <div class="section grow">
            <label class="section-label">Your top artists by time period:</label>
            <div class="period-table-container">
                <table class="period-table collapsed" id="periodTable">
                    <thead>
                        <tr>
                            <th class="period-col hidden">Overall</th>
                            <th class="period-col hidden">12 Month</th>
                            <th class="period-col hidden">1 Month</th>
                            <th class="total-col">Total (editable)</th>
                        </tr>
                    </thead>
                    <tbody id="periodTableBody"></tbody>
                </table>
                <button class="add-row-btn" onclick="addArtistRow()">+ Add Artist</button>
                <button class="explain-btn" id="periodExplainBtn" onclick="togglePeriodExplain()">Explain numbers</button>
                <div class="explain-text" id="periodExplainText">Numbers show your play counts from Last.fm. Total combines plays across all time periods. Will be used as weight to find similar artists for recommendations.</div>
            </div>
        </div>
        <button id="goBtn" onclick="go()">Find Similar Artists</button>
        <div id="status"></div>
    </div>
    <div class="right-panel">
        <div class="results-header">
            <h3>Recommended Artists</h3>
            <button class="explain-btn" id="resultsExplainBtn" onclick="toggleResultsExplain()">Explain how calculated</button>
        </div>
        <div class="explain-text" id="resultsExplainText">Match scores show how similar each artist is to your favorites, weighted by your play counts.</div>
        <div class="table-scroll">
            <table id="resultsTable"></table>
        </div>
    </div>

    <script>
        const PERIODS = [
            { key: 'overall', label: 'Overall' },
            { key: '12month', label: '12 Month' },
            { key: '1month', label: '1 Month' }
        ];
        let periodData = {};  // { period: [{ name, playcount }, ...] }
        let aggregatedArtists = [];  // [{ name, weight }, ...] sorted by weight desc
        let periodExpanded = false;
        let resultsExpanded = false;
        const artistInfoCache = new Map();

        async function fetchWithRetry(url, maxRetries = 60, statusCallback = null) {
            const delay = 100;
            let lastError;

            for (let attempt = 0; attempt <= maxRetries; attempt++) {
                try {
                    const resp = await fetch(url);
                    if (resp.ok) {
                        return await resp.json();
                    }
                    lastError = new Error('HTTP ' + resp.status);
                } catch (err) {
                    lastError = err;
                }

                if (attempt < maxRetries) {
                    if (statusCallback) {
                        statusCallback('Retry ' + (attempt + 1) + '/' + maxRetries + '...');
                    }
                    await new Promise(r => setTimeout(r, delay));
                }
            }
            throw lastError;
        }

        async function fetchArtistInfo(name) {
            if (artistInfoCache.has(name)) {
                return artistInfoCache.get(name);
            }
            try {
                const data = await fetchWithRetry('./api/artist/info?artist=' + encodeURIComponent(name));
                const info = data.data.artist;
                artistInfoCache.set(name, info);
                return info;
            } catch (err) {
                console.error('Error fetching artist info', name, err);
                return null;
            }
        }

        function formatTags(tags, limit = 5) {
            if (!tags || !tags.tag || tags.tag.length === 0) return '';
            return tags.tag.slice(0, limit).map(t => '<span class="tag">' + escapeHtml(t.name) + '</span>').join('');
        }

        async function populateArtistInfo() {
            const rows = document.querySelectorAll('#resultsTable tbody tr[data-artist]');
            const batchSize = 10;
            for (let i = 0; i < rows.length; i += batchSize) {
                const batch = Array.from(rows).slice(i, i + batchSize);
                await Promise.all(batch.map(async (row) => {
                    const artistName = row.getAttribute('data-artist');
                    const info = await fetchArtistInfo(artistName);
                    if (!info) return;

                    const tagsEl = row.querySelector('.artist-tags');
                    if (tagsEl) {
                        tagsEl.innerHTML = formatTags(info.tags);
                    }
                }));
            }
        }

        function togglePeriodExplain() {
            periodExpanded = !periodExpanded;
            const table = document.getElementById('periodTable');
            table.classList.toggle('collapsed', !periodExpanded);
            const cols = document.querySelectorAll('#periodTable .period-col');
            cols.forEach(col => col.classList.toggle('hidden', !periodExpanded));
            document.getElementById('periodExplainBtn').textContent = periodExpanded ? 'Hide details' : 'Explain numbers';
            document.getElementById('periodExplainText').classList.toggle('visible', periodExpanded);
        }

        function toggleResultsExplain() {
            resultsExpanded = !resultsExpanded;
            const cols = document.querySelectorAll('#resultsTable .match-col');
            cols.forEach(col => col.classList.toggle('hidden', !resultsExpanded));
            document.getElementById('resultsExplainBtn').textContent = resultsExpanded ? 'Hide details' : 'Explain numbers';
            document.getElementById('resultsExplainText').classList.toggle('visible', resultsExpanded);
        }

        async function loadUserArtists() {
            const username = document.getElementById('username').value.trim();
            if (!username) {
                alert('Please enter a username');
                return;
            }

            const status = document.getElementById('status');
            periodData = {};

            for (const p of PERIODS) {
                status.textContent = 'Loading ' + p.label + ' artists...';
                try {
                    const data = await fetchWithRetry(
                        './api/user/top-artists?user=' + encodeURIComponent(username) + '&period=' + p.key
                    );
                    periodData[p.key] = (data.data.artists || [])
                        .map(a => ({ name: a.name, playcount: parseInt(a.playcount, 10) || 0 }))
                        .filter(a => a.playcount >= 5);
                } catch (err) {
                    console.error('Error loading ' + p.key, err);
                    periodData[p.key] = [];
                }
            }

            renderPeriodTable();
            status.textContent = 'Loaded artists for ' + username;
        }

        function renderPeriodTable() {
            const tbody = document.getElementById('periodTableBody');
            tbody.innerHTML = '';

            // Calculate total weight for each artist across all periods
            const artistTotals = new Map();
            const artistNames = new Map(); // key -> proper cased name
            const artistPeriodCount = new Map();
            for (const period of PERIODS) {
                const artists = periodData[period.key] || [];
                for (const a of artists) {
                    const key = a.name.toLowerCase();
                    artistTotals.set(key, (artistTotals.get(key) || 0) + a.playcount);
                    artistPeriodCount.set(key, (artistPeriodCount.get(key) || 0) + 1);
                    if (!artistNames.has(key)) artistNames.set(key, a.name);
                }
            }

            // Build sorted aggregated list and store globally
            aggregatedArtists = Array.from(artistTotals.entries())
                .sort((a, b) => b[1] - a[1])
                .map(([key, weight]) => ({ name: artistNames.get(key), weight }));

            // Find the max number of rows needed
            const maxRows = Math.max(
                ...PERIODS.map(p => (periodData[p.key] || []).length),
                aggregatedArtists.length
            );

            for (let i = 0; i < maxRows; i++) {
                const tr = document.createElement('tr');
                for (const period of PERIODS) {
                    const artists = periodData[period.key] || [];
                    const td = document.createElement('td');
                    td.className = 'period-col' + (periodExpanded ? '' : ' hidden');
                    if (artists[i]) {
                        const a = artists[i];
                        const isMulti = artistPeriodCount.get(a.name.toLowerCase()) > 1;
                        if (isMulti) td.className += ' multi-period';
                        td.innerHTML = '<div class="artist-cell">' +
                            '<span class="artist-name" title="' + escapeHtml(a.name) + '">' + escapeHtml(a.name) + '</span>' +
                            '<span class="playcount">' + a.playcount + '</span>' +
                            '</div>';
                    }
                    tr.appendChild(td);
                }
                // Total column - editable inputs
                const td = document.createElement('td');
                td.className = 'total-col';
                if (aggregatedArtists[i]) {
                    const a = aggregatedArtists[i];
                    td.innerHTML = '<div class="editable-artist">' +
                        '<input type="text" class="artist-input" value="' + escapeHtml(a.name) + '">' +
                        '<input type="number" class="weight-input" value="' + a.weight + '">' +
                        '<button class="remove-btn" onclick="removeArtistRow(this)">\u00d7</button>' +
                        '</div>';
                }
                tr.appendChild(td);
                tbody.appendChild(tr);
            }
        }

        function addArtistRow() {
            const tbody = document.getElementById('periodTableBody');
            const tr = document.createElement('tr');

            // Empty period columns
            for (let i = 0; i < PERIODS.length; i++) {
                const td = document.createElement('td');
                td.className = 'period-col' + (periodExpanded ? '' : ' hidden');
                tr.appendChild(td);
            }

            // Editable total column
            const td = document.createElement('td');
            td.className = 'total-col';
            td.innerHTML = '<div class="editable-artist">' +
                '<input type="text" class="artist-input" value="" placeholder="Artist name">' +
                '<input type="number" class="weight-input" value="100">' +
                '<button class="remove-btn" onclick="removeArtistRow(this)">\u00d7</button>' +
                '</div>';
            tr.appendChild(td);
            tbody.appendChild(tr);

            // Focus the new input
            tr.querySelector('.artist-input').focus();
        }

        function removeArtistRow(btn) {
            const tr = btn.closest('tr');
            tr.remove();
        }

        function getEditedArtists() {
            const artists = [];
            const rows = document.querySelectorAll('#periodTableBody tr');
            for (const row of rows) {
                const nameInput = row.querySelector('.artist-input');
                const weightInput = row.querySelector('.weight-input');
                if (nameInput && weightInput) {
                    const name = nameInput.value.trim();
                    const weight = parseInt(weightInput.value, 10) || 0;
                    if (name && weight > 0) {
                        artists.push({ name, weight });
                    }
                }
            }
            return artists;
        }

        function renderTable(artists, allSimilar) {
            const sorted = Array.from(allSimilar.values()).sort((a, b) => b.total - a.total).slice(0, 100);

            // Create set of top 30 input artist names (by weight) for filtering
            const top30 = [...artists].sort((a, b) => b.weight - a.weight).slice(0, 30);
            const inputArtistNames = new Set(top30.map(a => a.name.toLowerCase()));

            const hiddenClass = resultsExpanded ? '' : ' hidden';
            let html = '<thead><tr><th>Similar Artist</th>';
            for (const artist of artists) {
                html += '<th class="match-col' + hiddenClass + '">' + escapeHtml(artist.name) + '</th>';
            }
            html += '<th class="total match-col' + hiddenClass + '">Total</th></tr></thead><tbody>';

            for (const row of sorted) {
                // Skip artists that are in the top 30 input list
                if (inputArtistNames.has(row.artist.name.toLowerCase())) continue;

                // Get top 5 matching input artists
                const topMatches = Object.entries(row.matches)
                    .sort((a, b) => b[1] - a[1])
                    .slice(0, 5)
                    .map(([name]) => name);
                const similarTo = topMatches.length > 0
                    ? '<div class="similar-to">(Similar to: ' + topMatches.map(escapeHtml).join(', ') + ')</div>'
                    : '';

                html += '<tr data-artist="' + escapeHtml(row.artist.name) + '"><td>' +
                    escapeHtml(row.artist.name) + similarTo +
                    '<div class="artist-tags"></div></td>';
                for (const artist of artists) {
                    const match = row.matches[artist.name];
                    html += '<td class="match match-col' + hiddenClass + '">' + (match ? match.toFixed(2) : '') + '</td>';
                }
                html += '<td class="match total match-col' + hiddenClass + '">' + row.total.toFixed(2) + '</td></tr>';
            }
            html += '</tbody>';

            document.getElementById('resultsTable').innerHTML = html;
        }

        async function go() {
            // Read artists/weights from editable Total column inputs
            const artists = getEditedArtists();

            if (artists.length === 0) {
                alert('Please load artists first or add them manually');
                return;
            }

            const btn = document.getElementById('goBtn');
            const status = document.getElementById('status');
            btn.disabled = true;

            // Fetch similar artists for each
            const results = {};
            const allSimilar = new Map(); // name -> { artist data, matches by seed }
            const failedArtists = [];
            let lastRenderTime = 0;

            for (let i = 0; i < artists.length; i++) {
                const artist = artists[i];
                const baseStatus = 'Fetching ' + (i + 1) + '/' + artists.length + ': ' + artist.name;
                status.textContent = baseStatus;

                try {
                    const data = await fetchWithRetry(
                        './api/artist/similar?artist=' + encodeURIComponent(artist.name),
                        60,
                        (retryMsg) => { status.textContent = baseStatus + ' - ' + retryMsg; }
                    );
                    results[artist.name] = data.data.artists || [];

                    // Aggregate similar artists
                    for (const similar of results[artist.name]) {
                        if (!allSimilar.has(similar.name)) {
                            allSimilar.set(similar.name, {
                                artist: similar,
                                matches: {},
                                total: 0
                            });
                        }
                        const entry = allSimilar.get(similar.name);
                        const weightedMatch = similar.match * artist.weight;
                        entry.matches[artist.name] = weightedMatch;
                        entry.total += weightedMatch;
                    }

                    // Rate-limited progressive rendering (every 10 seconds)
                    if (Date.now() - lastRenderTime >= 10000) {
                        renderTable(artists, allSimilar);
                        lastRenderTime = Date.now();
                    }
                } catch (err) {
                    console.error('Error fetching', artist.name, err);
                    failedArtists.push(artist.name);
                }
            }

            // Final render
            renderTable(artists, allSimilar);
            if (failedArtists.length > 0) {
                status.textContent = 'Done. Failed to load: ' + failedArtists.join(', ');
            } else {
                status.textContent = 'Showing top 100 of ' + allSimilar.size + ' similar artists';
            }
            btn.disabled = false;

            // Fetch and display artist info (images and tags)
            populateArtistInfo();
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`
}
