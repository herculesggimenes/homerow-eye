package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadEyeCursor(t *testing.T) {
	now := time.Unix(1000, 0)
	path := writeEyeCursorFixture(t, `{
		"sessionId": "session-1",
		"ts": 999.75,
		"screenX": 100.4,
		"screenY": 200.6
	}`)

	target, err := readEyeCursor(path, 2*time.Second, now)
	if err != nil {
		t.Fatalf("readEyeCursor() error = %v", err)
	}

	if target.Point.X != 100 || target.Point.Y != 201 {
		t.Fatalf("readEyeCursor() point = %v, want (100,201)", target.Point)
	}

	if target.Payload.SessionID != "session-1" {
		t.Fatalf("readEyeCursor() session = %q, want session-1", target.Payload.SessionID)
	}
}

func TestReadEyeCursorRejectsStaleSample(t *testing.T) {
	now := time.Unix(1000, 0)
	path := writeEyeCursorFixture(t, `{
		"sessionId": "session-1",
		"ts": 990,
		"screenX": 100,
		"screenY": 200
	}`)

	_, err := readEyeCursor(path, 2*time.Second, now)
	if err == nil {
		t.Fatal("readEyeCursor() expected stale sample error")
	}

	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("readEyeCursor() error = %v, want stale sample error", err)
	}
}

func TestReadEyeCursorRejectsMissingSession(t *testing.T) {
	path := writeEyeCursorFixture(t, `{
		"ts": 1000,
		"screenX": 100,
		"screenY": 200
	}`)

	_, err := readEyeCursor(path, 2*time.Second, time.Unix(1000, 0))
	if err == nil {
		t.Fatal("readEyeCursor() expected missing session error")
	}

	if !strings.Contains(err.Error(), "sessionId") {
		t.Fatalf("readEyeCursor() error = %v, want sessionId error", err)
	}
}

func writeEyeCursorFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "latest-cursor.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return path
}
