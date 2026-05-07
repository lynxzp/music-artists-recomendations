package cache

import (
	"database/sql"
	"errors"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

const maxAge = 7 * 24 * time.Hour

type Cache struct {
	db *sql.DB
}

type Entry struct {
	Request   string
	Response  string
	Timestamp time.Time
}

func New(dbPath string) (*Cache, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cache (
			request   TEXT PRIMARY KEY,
			response  TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			ttl       INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Migrate existing databases that were created before the ttl column was added.
	_, _ = db.Exec(`ALTER TABLE cache ADD COLUMN ttl INTEGER NOT NULL DEFAULT 0`)

	c := &Cache{db: db}
	c.cleanup()

	return c, nil
}

func (c *Cache) Get(request string) (string, bool) {
	var response string
	var timestamp int64
	var ttl int64

	err := c.db.QueryRow(
		"SELECT response, timestamp, ttl FROM cache WHERE request = ?",
		request,
	).Scan(&response, &timestamp, &ttl)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("failed to query cache", "request", request, "error", err)
		}
		return "", false
	}

	entryTime := time.Unix(timestamp, 0)
	maxAgeForEntry := maxAge
	if ttl > 0 {
		maxAgeForEntry = time.Duration(ttl) * time.Second
	}
	if time.Since(entryTime) > maxAgeForEntry {
		if _, err := c.db.Exec("DELETE FROM cache WHERE request = ?", request); err != nil {
			slog.Warn("failed to delete expired cache entry", "request", request, "error", err)
		}
		return "", false
	}

	return response, true
}

func (c *Cache) Set(request, response string) {
	c.SetWithTTL(request, response, 0)
}

func (c *Cache) SetWithTTL(request, response string, ttl time.Duration) {
	var ttlSeconds int64
	if ttl > 0 {
		ttlSeconds = int64(ttl.Seconds())
	}
	if _, err := c.db.Exec(`
		INSERT OR REPLACE INTO cache (request, response, timestamp, ttl)
		VALUES (?, ?, ?, ?)
	`, request, response, time.Now().Unix(), ttlSeconds); err != nil {
		slog.Warn("failed to set cache entry", "request", request, "error", err)
	}
}

func (c *Cache) cleanup() {
	now := time.Now().Unix()
	defaultCutoff := time.Now().Add(-maxAge).Unix()
	if _, err := c.db.Exec(`
		DELETE FROM cache
		WHERE (ttl > 0 AND timestamp + ttl < ?)
		   OR (ttl = 0 AND timestamp < ?)
	`, now, defaultCutoff); err != nil {
		slog.Warn("failed to cleanup expired cache entries", "error", err)
	}
}

func (c *Cache) Close() error {
	return c.db.Close()
}
