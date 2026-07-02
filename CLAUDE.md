# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

[//]: # "SergV1.1"

## How you edit this project

You are a senior Go engineer who writes simple, readable, and efficient Go code.
You follow Go conventions strictly.

### Git
- Don't use git worktrees
- Instead of git commit, after checking correctness, stage all changes and continue next task. Only user can commit.
- Don't use `git -C`, use relative paths

### Core Principles

Simple is better than clever. Prefer code a junior engineer can understand quickly.
Accept interfaces, return structs. Define interfaces at the call site, not the implementation site.
Handle every error. If you truly want to ignore an error, assign it to _ and add a comment explaining why.
Do not abstract prematurely. Write concrete code first. Extract interfaces and generics only when you have two or more concrete implementations.

### Error Handling

Return errors as the last return value. Check them immediately with if err != nil.
Wrap errors with context using fmt.Errorf("operation failed: %w", err). Always use %w for wrapping.
Define sentinel errors with var ErrNotFound = errors.New("not found") for errors callers need to check.
Use errors.Is and errors.As for error inspection. Never compare error strings.
Create custom error types only when callers need structured information beyond the error message.

### Concurrency Patterns

Use goroutines for concurrent work. Always ensure goroutines can terminate. Never fire-and-forget.
Use channels for communication between goroutines. Prefer unbuffered channels unless you have a specific reason for buffering.
Use context.Context for cancellation, timeouts, and request-scoped values. Pass it as the first parameter.
Use errgroup.Group from golang.org/x/sync/errgroup for to wait for a group of goroutines to finish and concurrent operations that return errors.
Protect shared state with sync.Mutex. Keep the critical section as small as possible.
Use sync.Once for one-time initialization. Use sync.Map only for cache-like access patterns.

### Interfaces

Keep interfaces small. One to three methods is ideal.
Define interfaces where they are consumed, not where they are implemented.
Use io.Reader, io.Writer, fmt.Stringer, and other stdlib interfaces wherever possible.
Avoid interface pollution. If there is only one implementation, you do not need an interface.

### Testing

Write table-driven tests with t.Run for subtests.
Use testify/assert or testify/require for assertions. Use require when failure should stop the test.
Use httptest.NewServer for HTTP handler tests. Use httptest.NewRecorder for unit testing handlers.
Use t.Parallel() for tests that do not share state.
Mock external dependencies with interfaces. Do not use reflection-based mocking frameworks.
Write benchmarks with func BenchmarkX(b *testing.B) for performance-critical code.

## Code Security

- Never concatenate user input into SQL — use parameterized queries/prepared statements only
- Validate and sanitize all input at API boundary (type, length, range, allowed chars)
- Use an allowlist approach, not a denylist
- Check permissions server-side on every request — never trust client claims

## Project Overview

Go web application for fetching and analyzing music statistics from the Last.fm API. Provides HTTP endpoints for finding similar artists and user top artists with weighted aggregation.

## Build and Run Commands

```bash
# Build
go build -o music-recomendations .

# Run tests
go test ./... --race
```

## Configuration

Environment variables:

```bash
export API_KEY=your_lastfm_api_key
export CACHE_PATH=./cache.db  # optional, defaults to ./cache.db
```

## Architecture

- `main.go` - Entry point, calls `service.Run()`
- `internal/service/` - Service orchestration layer
  - `music-recomendations.go` - Initializes config, logging, and HTTP server; manages lifecycle
- `internal/server/` - HTTP server implementation
  - `server.go` - Server initialization, lifecycle, graceful shutdown, `MusicClient` interface
  - `handlers.go` - HTTP request handlers
  - `routes.go` - Route registration
  - `validation.go` - Input validation (artist names, usernames, periods)
  - `static.go` - Embeds `static/` via `go:embed`; serves `/static/` file server
  - `static/` - Frontend assets: `index.html`, `app.js`, `style.css`
- `lastfm/` - Last.fm API client package
  - `api.go` - Client configuration and HTTP client setup
  - `artist.go` - Artist.getSimilar and Artist.getInfo endpoints
  - `user.go` - User.getTopArtists endpoint
  - `cache/cache.go` - SQLite-based response cache (7-day TTL)
  - `ratelimit/ratelimit.go` - Token bucket rate limiter (1 req/sec)
  - `retry/retry.go` - HTTP transport with exponential backoff retry

## HTTP Endpoints

| Method | Path                    | Handler                    | Parameters                  |
|--------|-------------------------|----------------------------|-----------------------------|
| GET    | `/`                     | handleIndex                | —                           |
| GET    | `/api/artist/similar`   | handleArtistGetSimilar     | `artist` (query)            |
| GET    | `/api/artist/info`      | handleArtistGetInfo        | `artist`, `user` (query)    |
| GET    | `/api/user/top-artists` | handleUserGetTopArtists    | `user`, `period` (query)    |
| POST   | `/api/append`           | handleAppendSimilarArtists | JSON body: `{a, b, weight}` |
| GET    | `/static/{path...}`     | (embedded file server)     | —                           |

Server listens on `0.0.0.0:8080` with graceful shutdown support (30s timeout).

## Packages and Functions

### `main`

- `main()` — entry point, calls `service.Run()` and logs fatal on error

### `internal/service`

**Types:**
- `Config` — `APIKey`, `SimilarArtistsLimit`, `TopArtistsLimit`, `CachePath`

**Functions:**
- `Run() error` — loads config from env, initializes logger, creates and starts HTTP server

### `internal/server`

**Types:**
- `Config` — `APIKey`, `SimilarArtistsLimit`, `TopArtistsLimit`, `CachePath`, `Logger`
- `MusicClient` (interface) — `ArtistGetSimilar`, `ArtistGetInfo`, `UserGetTopArtists`; consumed by `Server`, satisfied by `*lastfm.Client`
- `Server` (unexported) — `client MusicClient`, `cache`, `config`, `logger`, `httpServer`
- `appendRequest` (unexported) — `A []SimilarArtist`, `B []SimilarArtist`, `Weight float64`

**Exported Functions:**
- `New(cfg Config) (*Server, error)` — creates Server with cache and rate limiter
- `(*Server) Start() error` — starts HTTP server with signal-based graceful shutdown
- `(*Server) Close() error` — closes the cache

**Unexported Functions:**
- `(*Server) registerRoutes(mux *http.ServeMux)` — registers all HTTP routes
- `(*Server) handleIndex(w, r)` — serves embedded `static/index.html`
- `(*Server) handleArtistGetSimilar(w, r)` — similar artists lookup
- `(*Server) handleArtistGetInfo(w, r)` — artist info/tags lookup, lowercases tag names
- `(*Server) handleAppendSimilarArtists(w, r)` — merges two artist lists with weighted aggregation
- `(*Server) handleUserGetTopArtists(w, r)` — user top artists by period
- `isValidArtistName(s string) bool` — validates artist name (max 256 chars, safe pattern)
- `isValidUsername(s string) bool` — validates username (max 64 chars, safe pattern)
- `isValidPeriod(s string) bool` — validates period (overall, 7day, 1month, 3month, 6month, 12month)
- `initStaticRoutes(mux *http.ServeMux)` — registers embedded static file server at `/static/`

### `lastfm`

**Constants:**
- `BaseURL = "https://ws.audioscrobbler.com/2.0/"`

**Types:**
- `Client` — `apiKey`, `baseURL`, `cache`, `limiter`, `httpClient`
- `SimilarArtist` — `Name`, `MBID`, `Match`, `URL`
- `ArtistTag` — `Name`, `URL`
- `ArtistInfo` — `Name`, `MBID`, `URL`, `Stats` (UserPlaycount), `Tags`
- `TopArtist` — `Name`, `MBID`, `Playcount`, `URL`
- `similarArtistsResponse` (unexported)
- `artistInfoResponse` (unexported)
- `topArtistsResponse` (unexported)

**Functions:**
- `NewClient(apiKey string) *Client` — basic client without cache/limiter
- `NewClientWithCache(apiKey string, c *cache.Cache) *Client` — client with cache
- `NewClientWithCacheAndLimiter(apiKey string, c *cache.Cache, l *ratelimit.Limiter) *Client` — fully configured client
- `(*Client) ArtistGetSimilar(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]SimilarArtist, error)` — fetches similar artists
- `(*Client) ArtistGetInfo(ctx context.Context, artist, mbid, username string, autocorrect bool) (*ArtistInfo, error)` — fetches artist info, tags, and user playcount stats (when username provided)
- `(*Client) UserGetTopArtists(ctx context.Context, user, period string, limit, page int) ([]TopArtist, error)` — fetches user's top artists
- `AppendSimilarArtists(a, b []SimilarArtist, weight float64) []SimilarArtist` — merges two slices, sums match values, applies weight to b

### `lastfm/cache`

**Constants:**
- `maxAge = 7 * 24 * time.Hour`

**Types:**
- `Cache` — wraps `*sql.DB`
- `Entry` — `Request`, `Response`, `Timestamp`

**Functions:**
- `New(dbPath string) (*Cache, error)` — opens/creates SQLite DB, initializes table, runs cleanup, migrates old schema
- `(*Cache) Get(request string) (string, bool)` — returns cached response if not expired (uses per-entry TTL if set, otherwise falls back to `maxAge`)
- `(*Cache) Set(request, response string)` — upserts cache entry with default TTL (0 → `maxAge`)
- `(*Cache) SetWithTTL(request, response string, ttl time.Duration)` — upserts cache entry with a per-entry TTL
- `(*Cache) Close() error` — closes DB connection
- `(*Cache) cleanup()` (unexported) — deletes expired entries based on per-entry TTL or global `maxAge`

### `lastfm/ratelimit`

**Types:**
- `Limiter` — `mu sync.Mutex`, `lastReq time.Time`, `interval time.Duration`

**Functions:**
- `New(interval time.Duration) *Limiter` — creates rate limiter
- `(*Limiter) Wait(ctx context.Context) error` — blocks until interval elapsed since last request, returns ctx.Err() on cancellation

### `lastfm/retry`

**Types:**
- `RetryTransport` — `Base http.RoundTripper`, `Delays []time.Duration`

**Variables:**
- `DefaultDelays` — exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 512s, 1024s

**Functions:**
- `(*RetryTransport) RoundTrip(req *http.Request) (*http.Response, error)` — retries on 429/5xx with configured delays
- `isRetryable(statusCode int) bool` (unexported) — returns true for 429 and 500-599

## Maintenance

When adding, removing, or changing any function, type, endpoint, or package, update the relevant section in this file to keep it in sync with the codebase.