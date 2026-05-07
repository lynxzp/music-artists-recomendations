package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	if c == nil {
		t.Fatal("New() returned nil cache")
	}
}

func TestSetThenGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	c.Set("req1", "resp1")

	got, ok := c.Get("req1")
	if !ok {
		t.Fatal("Get() returned ok=false, want true")
	}
	if got != "resp1" {
		t.Errorf("Get() = %q, want %q", got, "resp1")
	}
}

func TestGetMissingKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	got, ok := c.Get("nonexistent")
	if ok {
		t.Error("Get() returned ok=true for missing key")
	}
	if got != "" {
		t.Errorf("Get() = %q, want empty string", got)
	}
}

func TestGetExpiredEntry(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	oldTimestamp := time.Now().Add(-8 * 24 * time.Hour).Unix()
	_, err = c.db.Exec(
		"INSERT INTO cache (request, response, timestamp) VALUES (?, ?, ?)",
		"old_req", "old_resp", oldTimestamp,
	)
	if err != nil {
		t.Fatalf("failed to insert old entry: %v", err)
	}

	got, ok := c.Get("old_req")
	if ok {
		t.Error("Get() returned ok=true for expired entry")
	}
	if got != "" {
		t.Errorf("Get() = %q, want empty string", got)
	}
}

func TestSetOverwritesExistingKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	c.Set("req1", "resp_v1")
	c.Set("req1", "resp_v2")

	got, ok := c.Get("req1")
	if !ok {
		t.Fatal("Get() returned ok=false after upsert")
	}
	if got != "resp_v2" {
		t.Errorf("Get() = %q, want %q", got, "resp_v2")
	}
}

func TestCleanupRemovesExpiredEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	oldTimestamp := time.Now().Add(-8 * 24 * time.Hour).Unix()
	_, err = c.db.Exec(
		"INSERT INTO cache (request, response, timestamp) VALUES (?, ?, ?)",
		"expired_req", "expired_resp", oldTimestamp,
	)
	if err != nil {
		t.Fatalf("failed to insert expired entry: %v", err)
	}
	c.Set("fresh_req", "fresh_resp")

	c.cleanup()

	var count int
	err = c.db.QueryRow("SELECT COUNT(*) FROM cache WHERE request = ?", "expired_req").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 0 {
		t.Errorf("expired entry still exists after cleanup, count = %d", count)
	}

	got, ok := c.Get("fresh_req")
	if !ok {
		t.Error("fresh entry missing after cleanup")
	}
	if got != "fresh_resp" {
		t.Errorf("fresh entry = %q, want %q", got, "fresh_resp")
	}
}

func TestClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	err = c.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestSetWithTTLThenGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	c.SetWithTTL("req1", "resp1", 5*time.Minute)

	got, ok := c.Get("req1")
	if !ok {
		t.Fatal("Get() returned ok=false, want true")
	}
	if got != "resp1" {
		t.Errorf("Get() = %q, want %q", got, "resp1")
	}
}

func TestGetExpiredTTL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	c.SetWithTTL("req1", "resp1", 1*time.Second)
	time.Sleep(2 * time.Second)

	got, ok := c.Get("req1")
	if ok {
		t.Error("Get() returned ok=true for expired TTL entry")
	}
	if got != "" {
		t.Errorf("Get() = %q, want empty string", got)
	}
}

func TestCleanupRemovesTTLEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	c, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	oldTimestamp := time.Now().Add(-8 * 24 * time.Hour).Unix()
	_, err = c.db.Exec(
		"INSERT INTO cache (request, response, timestamp, ttl) VALUES (?, ?, ?, ?)",
		"expired_req", "expired_resp", oldTimestamp, 0,
	)
	if err != nil {
		t.Fatalf("failed to insert expired entry: %v", err)
	}

	c.SetWithTTL("short_ttl_req", "short_ttl_resp", 1*time.Second)
	time.Sleep(2 * time.Second)

	c.SetWithTTL("fresh_ttl_req", "fresh_ttl_resp", 10*time.Minute)

	c.cleanup()

	for _, req := range []string{"expired_req", "short_ttl_req"} {
		var count int
		err = c.db.QueryRow("SELECT COUNT(*) FROM cache WHERE request = ?", req).Scan(&count)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if count != 0 {
			t.Errorf("%s still exists after cleanup, count = %d", req, count)
		}
	}

	got, ok := c.Get("fresh_ttl_req")
	if !ok {
		t.Error("fresh TTL entry missing after cleanup")
	}
	if got != "fresh_ttl_resp" {
		t.Errorf("fresh TTL entry = %q, want %q", got, "fresh_ttl_resp")
	}
}
