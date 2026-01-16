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
            width: 320px;
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            gap: 14px;
            background: #fff;
            padding: 24px;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.05);
        }
        .left-panel h3 {
            margin: 0 0 4px 0;
            font-size: 18px;
            font-weight: 600;
            color: #1a202c;
        }
        .right-panel {
            flex: 1;
            overflow: auto;
            background: #fff;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.05);
            padding: 20px;
        }
        .artist-entry {
            display: flex;
            gap: 8px;
            align-items: center;
        }
        .artist-entry input[type="text"] { flex: 1; }
        .artist-entry input[type="number"] { width: 75px; }
        .artist-entry button {
            width: 32px;
            height: 38px;
            background: #f1f5f9;
            border: 1px solid #e2e8f0;
            color: #64748b;
            font-size: 16px;
            border-radius: 6px;
        }
        .artist-entry button:hover {
            background: #fee2e2;
            border-color: #fca5a5;
            color: #dc2626;
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
            top: 0;
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
        }
    </style>
</head>
<body>
    <div class="left-panel">
        <h3>Artists</h3>
        <div class="username-section">
            <input type="text" id="username" placeholder="Last.fm username">
            <button onclick="loadUserArtists()">Load</button>
        </div>
        <div id="artistList"></div>
        <button onclick="addArtistRow()">+ Add Artist</button>
        <button id="goBtn" onclick="go()">Go</button>
        <div id="status"></div>
    </div>
    <div class="right-panel">
        <table id="resultsTable"></table>
    </div>

    <script>
        function addArtistRow(name = '', weight = 1) {
            const div = document.createElement('div');
            div.className = 'artist-entry';
            div.innerHTML =
                '<input type="text" placeholder="Artist name" value="' + name + '">' +
                '<input type="number" placeholder="Weight" value="' + weight + '" step="0.1">' +
                '<button onclick="this.parentElement.remove()">×</button>';
            document.getElementById('artistList').appendChild(div);
        }

        async function loadUserArtists() {
            const username = document.getElementById('username').value.trim();
            if (!username) {
                alert('Please enter a username');
                return;
            }

            try {
                const resp = await fetch('/api/user/top-artists?user=' + encodeURIComponent(username) + '&limit=%d');
                const data = await resp.json();
                const artists = data.data.artists || [];

                document.getElementById('artistList').innerHTML = '';
                for (const artist of artists) {
                    addArtistRow(artist.name, artist.playcount);
                }
            } catch (err) {
                console.error('Error loading user artists', err);
                alert('Failed to load artists for user');
            }
        }

        // Start with a few empty rows
        addArtistRow();
        addArtistRow();
        addArtistRow();

        function renderTable(artists, allSimilar) {
            const sorted = Array.from(allSimilar.values()).sort((a, b) => b.total - a.total);

            // Create set of input artist names for filtering
            const inputArtistNames = new Set(artists.map(a => a.name.toLowerCase()));

            let html = '<thead><tr><th>Similar Artist</th>';
            for (const artist of artists) {
                html += '<th>' + escapeHtml(artist.name) + '</th>';
            }
            html += '<th class="total">Total</th></tr></thead><tbody>';

            for (const row of sorted) {
                // Skip artists that are in the input list
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
            const entries = document.querySelectorAll('.artist-entry');
            const artists = [];
            entries.forEach(entry => {
                const name = entry.querySelector('input[type="text"]').value.trim();
                const weight = parseFloat(entry.querySelector('input[type="number"]').value) || 1;
                if (name) artists.push({ name, weight });
            });

            if (artists.length === 0) {
                alert('Please enter at least one artist');
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
