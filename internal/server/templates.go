package server

const indexHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Music Recommendations</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            display: flex;
            gap: 20px;
            height: 100vh;
        }
        .left-panel {
            width: 300px;
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            gap: 10px;
        }
        .right-panel {
            flex: 1;
            overflow: auto;
        }
        .artist-entry {
            display: flex;
            gap: 5px;
            align-items: center;
        }
        .artist-entry input[type="text"] { flex: 1; }
        .artist-entry input[type="number"] { width: 70px; }
        .artist-entry button { width: 30px; }
        input, button {
            padding: 8px;
            font-size: 14px;
        }
        button { cursor: pointer; }
        #goBtn {
            background: #007bff;
            color: white;
            border: none;
            padding: 12px;
            font-size: 16px;
        }
        #goBtn:hover { background: #0056b3; }
        #goBtn:disabled { background: #ccc; }
        table {
            border-collapse: collapse;
            width: 100%;
            font-size: 13px;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 6px 10px;
            text-align: left;
        }
        th {
            background: #f5f5f5;
            position: sticky;
            top: 0;
        }
        td.match { text-align: right; }
        .total { font-weight: bold; background: #f0f8ff; }
        #status { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="left-panel">
        <h3>Artists</h3>
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

        // Start with a few empty rows
        addArtistRow();
        addArtistRow();
        addArtistRow();

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

            for (let i = 0; i < artists.length; i++) {
                const artist = artists[i];
                status.textContent = 'Fetching ' + (i + 1) + '/' + artists.length + ': ' + artist.name;

                try {
                    const resp = await fetch('/api/artist/similar?artist=' + encodeURIComponent(artist.name) + '&limit=50&autocorrect=true');
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
                } catch (err) {
                    console.error('Error fetching', artist.name, err);
                }
            }

            // Sort by total match descending
            const sorted = Array.from(allSimilar.values()).sort((a, b) => b.total - a.total);

            // Build table
            let html = '<thead><tr><th>Similar Artist</th>';
            for (const artist of artists) {
                html += '<th>' + escapeHtml(artist.name) + '</th>';
            }
            html += '<th class="total">Total</th></tr></thead><tbody>';

            for (const row of sorted) {
                html += '<tr><td>' + escapeHtml(row.artist.name) + '</td>';
                for (const artist of artists) {
                    const match = row.matches[artist.name];
                    html += '<td class="match">' + (match ? match.toFixed(2) : '') + '</td>';
                }
                html += '<td class="match total">' + row.total.toFixed(2) + '</td></tr>';
            }
            html += '</tbody>';

            document.getElementById('resultsTable').innerHTML = html;
            status.textContent = 'Found ' + sorted.length + ' similar artists';
            btn.disabled = false;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`
