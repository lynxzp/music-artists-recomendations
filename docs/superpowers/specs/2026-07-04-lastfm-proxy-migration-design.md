# Migrate `music-recomendations` onto `lastfm-proxy` — design

Date: 2026-07-04

## Goal

Route all Last.fm traffic from `music-recomendations` through the internal
`lastfm-proxy` service (`POST http://lastfm-proxy:8080/v1/query`) instead of
calling `https://ws.audioscrobbler.com/2.0/` directly. The proxy already owns the
Last.fm API key, caching (adaptive TTL + archive + negative caching), global
rate-limiting, single-flight, retry/cooldown, and stale-if-error — so
`music-recomendations` drops its own local cache / rate-limiter / retry entirely
and becomes a thin client.

Both containers are already on the `traefik-net` network. The proxy is not exposed
via Traefik (`traefik.enable=false`).

## Non-goals

- No change to the public HTTP API of `music-recomendations` (routes, request
  params, response envelopes) — the frontend is unaffected.
- No change to the proxy itself.
- `music-sorter` is out of scope (separate future migration).

## Approach

Rewrite the existing `lastfm` package **in place**. Keep the package name, all
exported types (`SimilarArtist`, `ArtistTag`, `ArtistInfo`, `TopArtist`, the
response structs) and `AppendSimilarArtists`, and keep the three method
signatures. Because the signatures are unchanged, the `server.MusicClient`
interface and every handler stay untouched — only the client's internals change
from "GET Last.fm + local cache" to "POST proxy".

Rejected alternative: a new `proxyclient` package replacing `lastfm`. More churn
(move types, rewrite the interface + handlers) for no benefit.

## Client contract (`lastfm` package)

`Client` becomes:

```go
type Client struct {
    proxyURL   string        // e.g. http://lastfm-proxy:8080 (no trailing slash)
    httpClient *http.Client
}

func NewClient(proxyURL string) *Client // http.Client{Timeout: 15 * time.Second}
```

- `NewClient` strips a trailing `/` from `proxyURL`. Timeout 15 s comfortably
  exceeds the proxy's 8 s foreground fetch timeout plus margin.
- Remove `NewClientWithCache` and `NewClientWithCacheAndLimiter`.

Request body type — named and JSON-tagged so the wire format is the lowercase
`{"method":...,"params":...}` the proxy documents, **not** relying on
`encoding/json`'s case-insensitive field matching (which is what makes an
untagged, capitalized struct decode "by accident" today and would break if the
proxy ever tightened validation):

```go
type proxyRequest struct {
    Method string            `json:"method"`
    Params map[string]string `json:"params"`
}
```

Private shared helper:

```go
func (c *Client) query(ctx context.Context, method string, params map[string]string) ([]byte, error)
```

1. `json.Marshal(proxyRequest{Method: method, Params: params})` → lowercase keys.
2. `http.NewRequestWithContext(ctx, POST, c.proxyURL+"/v1/query", body)` with
   header **`Content-Type: application/json`**.
3. `c.httpClient.Do`; read the full body with `io.ReadAll`; close the body.
4. Status mapping:
   - `200` or `404` → return the body, `nil` error. (404 is the proxy's
     not-found / negative-cache disposition; its Last.fm error-6 envelope
     unmarshals to an empty result — this preserves today's behavior where
     "artist/user not found" yields an empty list, not an error.)
   - anything else (`400`, `502`, `504`, other) → return `nil` and an error that
     includes the status code and a short prefix of the body. No client-side
     retry — the proxy owns retry/cooldown/stale-if-error; a `502`/`504` here maps
     to the handler's existing 500, exactly as when Last.fm failed outright before.

The three public methods keep their signatures and their empty-input guards
(`artist == "" && mbid == ""` → error; `user == ""` → error), build a
`map[string]string` of only the non-empty params, call `query`, and unmarshal the
returned body into their existing response struct.

The pre-migration `if len(body) == 0 → "empty response from API"` guard in each
method is **removed deliberately**: the proxy always returns a non-empty JSON body
on 200/404, and an (impossible) empty body would surface as a JSON decode error
anyway — so there is no behavior regression, just one fewer redundant check.

Param rules (differences from the pre-migration direct client):

- **Do not send** `api_key` or `format` — the proxy owns them and rejects them
  with 400 (`query.New` reserved names).
- Method stays lowercase (`artist.getsimilar`, `artist.getinfo`,
  `user.gettopartists`).
- `autocorrect` → `"1"` only when true (unchanged).
- `limit` / `page` → decimal string, only when `> 0` (unchanged).
- `artist` / `mbid` / `user` / `period` / `username` → set only when non-empty.

Delete the subpackages `lastfm/cache`, `lastfm/ratelimit`, `lastfm/retry` and
their tests.

## Server / service wiring

`internal/server/server.go`:

- Delete the `Server.cache` field, the `Server.Close()` method, and the `error`
  return of `New`. The only error source in `New` was `cache.New`; `NewClient`
  cannot fail. New signature: `func New(cfg Config) *Server`.
- `Config`: replace `APIKey` and `CachePath` with `ProxyURL`. `New` applies the
  default `http://lastfm-proxy:8080` when `cfg.ProxyURL == ""` (mirrors how the
  old `cachePath` default lived in `New`, and keeps `New` self-sufficient for
  tests).
- Construct the client with `lastfm.NewClient(proxyURL)`.

`internal/service/music-recomendations.go`:

- `Config`: replace `APIKey`/`CachePath` with `ProxyURL`, read from
  `os.Getenv("LASTFM_PROXY_URL")`.
- Drop `defer srv.Close()`; call `srv := server.New(...)` without an `err` (and
  without the `if err != nil` block).

## Configuration

New env var: `LASTFM_PROXY_URL` (default `http://lastfm-proxy:8080`).
Removed: `API_KEY`, `CACHE_PATH`. `music-recomendations` no longer needs the
Last.fm key at all — the proxy holds the only copy.

## `docker-compose.yml` (parent repo, tracked)

In the `music-recomendations` service:

- Remove `- API_KEY=${LASTFM_API_KEY}` and `- CACHE_PATH=/app/data/cache.db`.
- Add `- LASTFM_PROXY_URL=http://lastfm-proxy:8080`.
- Remove the `volumes:` block `- music-recomendations-cache:/app/data`.

In the top-level `volumes:` section:

- Remove `music-recomendations-cache: {}` (otherwise a dead named volume lingers).

Not adding `depends_on: [lastfm-proxy]`: it does not guarantee readiness, the
client tolerates a temporarily-absent proxy (transient 502 → 500 at startup),
and it matches the rest of the compose file's style.

## `go.mod`

`go mod tidy` after deleting the local cache removes `modernc.org/sqlite` and its
transitive dependencies (the cache was the only user). `testify` stays (tests).

## `Dockerfile`

Remove the two lines that provisioned the now-gone SQLite cache directory (goal:
leave no scaffolding for the local cache):

- Line 9: `RUN mkdir -p /app/data && chown -R scratchuser:scratchuser /app/data`
- Line 15: `COPY --from=builder --chown=10001:10001 /app/data /app/data`

## Tests

- `lastfm/artist_test.go`, `lastfm/user_test.go`:
  - `newTestClient(url)` builds `&Client{proxyURL: url, httpClient: &http.Client{}}`.
  - Keep the `_ValidResponse` tests — the `httptest` server returns the same
    Last.fm envelope the proxy passes through; decoding is unchanged.
  - Keep `_MalformedJSON` (200 + junk → decode error) and `_EmptyInputs` (guard).
  - **Change** the `_Non200Status` tests: a `404` must now return an empty result
    with **no error** (not-found). Add a separate case for `502`/`504` → error.
  - Add an assertion on the POSTed request body that locks the wire format:
    it contains the **lowercase** keys `"method"` and `"params"` (not the
    Go-capitalized `"Method"`/`"Params"`), the expected `method` value, and does
    **not** contain `api_key` or `format`.
  - `TestAppendSimilarArtists` is unaffected (pure function).
- `internal/server/static_test.go`: `s := New(Config{Logger: testLogger(t)})`
  (drop `APIKey: "test"`, drop `require.NoError`, drop the now-unused `require`
  import if it becomes unused).
- `internal/server/handlers_test.go`: mocks `MusicClient`; the interface is
  unchanged, so it should compile as-is — verify.
- Delete tests for the removed subpackages.
- `go test -race ./...` must pass.

## Documentation

- `services/music-recomendations/CLAUDE.md`: update the `lastfm` package section
  (remove `cache`/`ratelimit`/`retry` subpackages; update `Client` fields and
  constructors to `NewClient(proxyURL)`), the Configuration section (env vars:
  `LASTFM_PROXY_URL`; remove `API_KEY`/`CACHE_PATH`), and the Architecture section.
  Also reflect `New` no longer returning an error and the removal of
  `Server.cache`/`Server.Close`.
- Parent `CLAUDE.md`: change the `LASTFM_API_KEY` line from "consumed by
  music-recomendations and by lastfm-proxy" to consumed by `lastfm-proxy` only,
  and remove the `API_KEY` mention for the `music-recomendations` service.

## Deployment (out of band, user-gated)

Building/pushing the `lynxzp/music-recomendations` image (`make build && make
push`) and updating the running compose stack are performed by the user, as with
the other services. Per this service's `CLAUDE.md`, changes are staged only —
the user commits.
