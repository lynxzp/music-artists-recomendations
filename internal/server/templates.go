package server

import "fmt"

func indexHTML(similarArtistsLimit, topArtistsLimit int) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Music Recommendations</title>
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
            width: 100%%;
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
            width: 100%%;
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
            background: linear-gradient(135deg, #6366f1 0%%, #4f46e5 100%%);
            color: white;
            border: none;
            padding: 14px 16px;
            font-size: 15px;
            font-weight: 600;
            margin-top: 8px;
            box-shadow: 0 2px 4px rgba(99, 102, 241, 0.3);
        }
        #goBtn:hover {
            background: linear-gradient(135deg, #4f46e5 0%%, #4338ca 100%%);
            box-shadow: 0 4px 8px rgba(99, 102, 241, 0.4);
        }
        #goBtn:disabled {
            background: #cbd5e1;
            box-shadow: none;
            cursor: not-allowed;
        }
        table {
            border-collapse: collapse;
            width: 100%%;
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
            background: linear-gradient(135deg, #6366f1 0%%, #4f46e5 100%%);
            color: white;
            border: none;
            font-weight: 600;
            box-shadow: 0 2px 4px rgba(99, 102, 241, 0.3);
        }
        .username-section button:hover {
            background: linear-gradient(135deg, #4f46e5 0%%, #4338ca 100%%);
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
                <table class="period-table" id="periodTable">
                    <thead>
                        <tr>
                            <th class="period-col">Overall</th>
                            <th class="period-col">12 Month</th>
                            <th class="period-col">1 Month</th>
                            <th class="total-col">Total (editable)</th>
                        </tr>
                    </thead>
                    <tbody id="periodTableBody"></tbody>
                </table>
                <button class="add-row-btn" onclick="addArtistRow()">+ Add Artist</button>
            </div>
        </div>
        <button id="goBtn" onclick="go()">Find Similar Artists</button>
        <div id="status"></div>
    </div>
    <div class="right-panel">
        <h3>Recommended Artists</h3>
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

        async function loadUserArtists() {
            const username = document.getElementById('username').value.trim();
            if (!username) {
                alert('Please enter a username');
                return;
            }

            const status = document.getElementById('status');
            status.textContent = 'Loading artists for all periods...';

            try {
                const promises = PERIODS.map(p =>
                    fetch('/api/user/top-artists?user=' + encodeURIComponent(username) + '&limit=%d&period=' + p.key)
                        .then(r => r.json())
                        .then(data => ({ period: p.key, artists: data.data.artists || [] }))
                );

                const results = await Promise.all(promises);
                periodData = {};
                for (const r of results) {
                    periodData[r.period] = r.artists.map(a => ({ name: a.name, playcount: parseInt(a.playcount, 10) || 0 }));
                }

                renderPeriodTable();
                status.textContent = 'Loaded artists for ' + username;
            } catch (err) {
                console.error('Error loading user artists', err);
                alert('Failed to load artists for user');
                status.textContent = '';
            }
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
                    td.className = 'period-col';
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
                td.className = 'period-col';
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
            const sorted = Array.from(allSimilar.values()).sort((a, b) => b.total - a.total);

            // Create set of top 30 input artist names (by weight) for filtering
            const top30 = [...artists].sort((a, b) => b.weight - a.weight).slice(0, 30);
            const inputArtistNames = new Set(top30.map(a => a.name.toLowerCase()));

            let html = '<thead><tr><th>Similar Artist</th>';
            for (const artist of artists) {
                html += '<th>' + escapeHtml(artist.name) + '</th>';
            }
            html += '<th class="total">Total</th></tr></thead><tbody>';

            for (const row of sorted) {
                // Skip artists that are in the top 30 input list
                if (inputArtistNames.has(row.artist.name.toLowerCase())) continue;

                html += '<tr><td>' + escapeHtml(row.artist.name) + '</td>';
                for (const artist of artists) {
                    const match = row.matches[artist.name];
                    html += '<td class="match">' + (match ? match.toFixed(2) : '') + '</td>';
                }
                html += '<td class="match total">' + row.total.toFixed(2) + '</td></tr>';
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
            let lastRenderTime = 0;

            for (let i = 0; i < artists.length; i++) {
                const artist = artists[i];
                status.textContent = 'Fetching ' + (i + 1) + '/' + artists.length + ': ' + artist.name;

                try {
                    const resp = await fetch('/api/artist/similar?artist=' + encodeURIComponent(artist.name) + '&limit=%d&autocorrect=true');
                    const data = await resp.json();
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
                }
            }

            // Final render
            renderTable(artists, allSimilar);
            status.textContent = 'Found ' + allSimilar.size + ' similar artists';
            btn.disabled = false;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`, topArtistsLimit, similarArtistsLimit)
}
