package cache

import (
	"database/sql"
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
			timestamp INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	c := &Cache{db: db}
	c.cleanup()

	return c, nil
}

func (c *Cache) Get(request string) (string, bool) {
	var response string
	var timestamp int64

	err := c.db.QueryRow(
		"SELECT response, timestamp FROM cache WHERE request = ?",
		request,
	).Scan(&response, &timestamp)

	if err != nil {
		return "", false
	}

	entryTime := time.Unix(timestamp, 0)
	if time.Since(entryTime) > maxAge {
		c.db.Exec("DELETE FROM cache WHERE request = ?", request)
		return "", false
	}

	return response, true
}

func (c *Cache) Set(request, response string) {
	c.db.Exec(`
		INSERT OR REPLACE INTO cache (request, response, timestamp)
		VALUES (?, ?, ?)
	`, request, response, time.Now().Unix())
}

func (c *Cache) cleanup() {
	cutoff := time.Now().Add(-maxAge).Unix()
	c.db.Exec("DELETE FROM cache WHERE timestamp < ?", cutoff)
}

func (c *Cache) Close() error {
	return c.db.Close()
}
